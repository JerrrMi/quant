package executor

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/JerrrMi/quant/internal/domain/command"
	"github.com/JerrrMi/quant/internal/domain/report"
	"github.com/JerrrMi/quant/internal/infra/agentstate"
	"github.com/JerrrMi/quant/internal/infra/binance"
	"github.com/JerrrMi/quant/internal/infra/db/models"
	"github.com/JerrrMi/quant/internal/infra/ws"
	"gorm.io/gorm"
)

// Service maps TradeCommand to venue RPCs plus DeltaReport snapshots.
type Service struct {
	Venue       Venue
	Dedup       *agentstate.DedupStore
	MaxNotional float64
	MaxOrders   int
}

// NewService constructs an executor wired to optional dedup persistence.
func NewService(v Venue, dedup *agentstate.DedupStore, maxNotional float64, maxOrders int) *Service {
	return &Service{Venue: v, Dedup: dedup, MaxNotional: maxNotional, MaxOrders: maxOrders}
}

// HandleTradeCommand runs one command envelope.
func (s *Service) HandleTradeCommand(ctx context.Context, cmd command.TradeCommand, refSeq int64, nowUnixMs int64) (command.CommandAck, report.DeltaReport) {
	ack := command.CommandAck{
		CommandID:       cmd.CommandID,
		RefEnvelopeSeq:  refSeq,
		AgentTimeUnixMs: nowUnixMs,
	}
	rep := report.DeltaReport{
		ReportID:                uuid.NewString(),
		InstanceID:              cmd.InstanceID,
		ExchangeEventTimeUnixMs: nowUnixMs,
		Details:                 map[string]string{"venue": "binance_usdm"},
	}

	if s == nil || s.Venue == nil {
		ack.Status = command.CommandStatusRejected
		ack.Message = "executor: no venue client"
		rep.Errors = append(rep.Errors, ack.Message)
		return ack, rep
	}

	if nowUnixMs > 0 && cmd.DeadlineUnixMs > 0 && nowUnixMs > cmd.DeadlineUnixMs {
		ack.Status = command.CommandStatusExpired
		ack.Message = "deadline exceeded before venue call"
		return s.refreshSnapshot(ctx, cmd.InstanceID, cmd.Symbol, ack, &rep)
	}

	switch cmd.Kind {
	case command.CommandKindReplace:
		ack.Status = command.CommandStatusRejected
		ack.Message = "replace not implemented for this venue adapter"
		return s.refreshSnapshot(ctx, cmd.InstanceID, cmd.Symbol, ack, &rep)

	case command.CommandKindCancel:
		return s.handleCancel(ctx, cmd, refSeq, nowUnixMs)

	case command.CommandKindPlace:
		return s.handlePlace(ctx, cmd, refSeq, nowUnixMs)

	default:
		ack.Status = command.CommandStatusRejected
		ack.Message = fmt.Sprintf("unknown command kind %q", cmd.Kind)
		return s.refreshSnapshot(ctx, cmd.InstanceID, cmd.Symbol, ack, &rep)
	}
}

func (s *Service) handlePlace(ctx context.Context, cmd command.TradeCommand, refSeq, nowUnixMs int64) (command.CommandAck, report.DeltaReport) {
	ack := command.CommandAck{
		CommandID:       cmd.CommandID,
		RefEnvelopeSeq:  refSeq,
		AgentTimeUnixMs: nowUnixMs,
	}
	rep := report.DeltaReport{
		ReportID:                uuid.NewString(),
		InstanceID:              cmd.InstanceID,
		ExchangeEventTimeUnixMs: nowUnixMs,
		Details:                 map[string]string{"venue": "binance_usdm", "op": "place"},
	}

	idem := ws.CommandDedupeKey(cmd)
	clientOID := sanitizeClientOID(idem)
	if clientOID == "" {
		clientOID = sanitizeClientOID(cmd.CommandID)
	}

	if row, err := s.dedupGet(ctx, idem); err == nil && row != nil && strings.TrimSpace(row.ClientOrderID) != "" {
		ov, qerr := withRetry(ctx, func(ctx context.Context) (*OrderView, error) {
			return s.Venue.QueryByClientOrder(ctx, cmd.Symbol, row.ClientOrderID)
		})
		if qerr == nil && ov != nil {
			ack.Status = mapVenueStatusToCommand(ov.Status)
			ack.ExchangeOrderID = formatOrderID(ov.VenueOrderID)
			ack.Message = "idempotent replay (local dedup)"
			s.fillRepFromOrder(&rep, ov)
			a, r := s.refreshSnapshot(ctx, cmd.InstanceID, cmd.Symbol, ack, &rep)
			if len(r.Fills) == 0 && ov.VenueOrderID != 0 {
				if fills, ferr := withRetry(ctx, func(ctx context.Context) ([]report.FillRecord, error) {
					return s.Venue.UserTrades(ctx, cmd.Symbol, ov.VenueOrderID)
				}); ferr == nil {
					r.Fills = fills
				}
			}
			return a, r
		}
		ack.Status = command.CommandStatusRejected
		if qerr != nil {
			ack.Message = fmt.Sprintf("dedup state present but venue query failed: %v", qerr)
		} else {
			ack.Message = "dedup state present but venue returned empty order view"
		}
		return s.refreshSnapshot(ctx, cmd.InstanceID, cmd.Symbol, ack, &rep)
	}

	if s.MaxOrders > 0 {
		if oo, err := s.Venue.OpenOrders(ctx, ""); err == nil && len(oo) >= s.MaxOrders {
			ack.Status = command.CommandStatusRejected
			ack.Message = fmt.Sprintf("max open orders exceeded (%d >= %d)", len(oo), s.MaxOrders)
			return s.refreshSnapshot(ctx, cmd.InstanceID, cmd.Symbol, ack, &rep)
		}
	}

	qty, estNotional, msg := s.computeBaseQty(ctx, cmd)
	if msg != "" {
		ack.Status = command.CommandStatusRejected
		ack.Message = msg
		return s.refreshSnapshot(ctx, cmd.InstanceID, cmd.Symbol, ack, &rep)
	}
	if qty <= 0 {
		ack.Status = command.CommandStatusRejected
		ack.Message = "quantity after rounding is zero"
		return s.refreshSnapshot(ctx, cmd.InstanceID, cmd.Symbol, ack, &rep)
	}
	if s.MaxNotional > 0 && estNotional > s.MaxNotional {
		ack.Status = command.CommandStatusRejected
		ack.Message = fmt.Sprintf("notional %.6f exceeds max %.6f", estNotional, s.MaxNotional)
		return s.refreshSnapshot(ctx, cmd.InstanceID, cmd.Symbol, ack, &rep)
	}

	var ov *OrderView
	err := backoffVenue(ctx, func(ctx context.Context) error {
		var e error
		ov, e = s.Venue.PlaceMarket(ctx, PlaceMarketIn{
			Symbol:           cmd.Symbol,
			Side:             cmd.Side,
			QuantityBase:     qty,
			ReduceOnly:       cmd.ReduceOnly,
			NewClientOrderID: clientOID,
		})
		if e == nil {
			return nil
		}
		if binance.IsDuplicateClientOrder(e) {
			qov, qe := s.Venue.QueryByClientOrder(ctx, cmd.Symbol, clientOID)
			if qe == nil && qov != nil {
				ov = qov
				return nil
			}
		}
		return e
	})

	if err != nil {
		ack.Status = command.CommandStatusRejected
		ack.Message = err.Error()
		if sug := retrySuggestion(err); sug != "" {
			rep.Details["retry_hint"] = sug
		}
		rep.Errors = append(rep.Errors, err.Error())
		return s.refreshSnapshot(ctx, cmd.InstanceID, cmd.Symbol, ack, &rep)
	}

	ack.Status = mapVenueStatusToCommand(ov.Status)
	ack.ExchangeOrderID = formatOrderID(ov.VenueOrderID)
	s.fillRepFromOrder(&rep, ov)

	_ = s.dedupUpsert(ctx, &models.AgentDedupKey{
		CorrelationKey:  idem,
		CommandID:       cmd.CommandID,
		ClientOrderID:   clientOID,
		ExchangeOrderID: formatOrderID(ov.VenueOrderID),
		LastStatus:      string(ack.Status),
	})

	if ov.VenueOrderID != 0 {
		if fills, ferr := withRetry(ctx, func(ctx context.Context) ([]report.FillRecord, error) {
			return s.Venue.UserTrades(ctx, cmd.Symbol, ov.VenueOrderID)
		}); ferr == nil {
			rep.Fills = fills
		}
	}

	outAck, outRep := s.refreshSnapshot(ctx, cmd.InstanceID, cmd.Symbol, ack, &rep)
	if outAck.Status == ack.Status || outAck.Status == "" {
		outAck.Status = ack.Status
	}
	if outAck.ExchangeOrderID == "" {
		outAck.ExchangeOrderID = ack.ExchangeOrderID
	}
	mergeFills(&outRep.Fills, rep.Fills)
	return outAck, outRep
}

func (s *Service) handleCancel(ctx context.Context, cmd command.TradeCommand, refSeq, nowUnixMs int64) (command.CommandAck, report.DeltaReport) {
	ack := command.CommandAck{
		CommandID:       cmd.CommandID,
		RefEnvelopeSeq:  refSeq,
		AgentTimeUnixMs: nowUnixMs,
	}
	rep := report.DeltaReport{
		ReportID:                uuid.NewString(),
		InstanceID:              cmd.InstanceID,
		ExchangeEventTimeUnixMs: nowUnixMs,
		Details:                 map[string]string{"venue": "binance_usdm", "op": "cancel"},
	}

	idem := ws.CommandDedupeKey(cmd)

	if row, err := s.dedupGet(ctx, idem); err == nil && row != nil {
		ack.Status = command.CommandStatusAccepted
		ack.Message = "cancel already applied (dedup replay)"
		if row.ExchangeOrderID != "" {
			ack.ExchangeOrderID = row.ExchangeOrderID
		}
		return s.refreshSnapshot(ctx, cmd.InstanceID, cmd.Symbol, ack, &rep)
	}

	cancelIn := parseCancelTarget(cmd)
	if !cancelIn.VenueOrderIDValid && strings.TrimSpace(cancelIn.ClientOrderID) == "" {
		ack.Status = command.CommandStatusRejected
		ack.Message = "cancel: set Intent.intent_id to venue numeric order id, or alphanumeric clientOrderId string"
		return s.refreshSnapshot(ctx, cmd.InstanceID, cmd.Symbol, ack, &rep)
	}

	var ov *OrderView
	err := backoffVenue(ctx, func(ctx context.Context) error {
		var cerr error
		ov, cerr = s.Venue.Cancel(ctx, cancelIn)
		return cerr
	})
	if err != nil {
		ack.Status = command.CommandStatusRejected
		ack.Message = err.Error()
		if sug := retrySuggestion(err); sug != "" {
			rep.Details["retry_hint"] = sug
		}
		rep.Errors = append(rep.Errors, err.Error())
		return s.refreshSnapshot(ctx, cmd.InstanceID, cmd.Symbol, ack, &rep)
	}

	ack.Status = command.CommandStatusCanceled
	if !strings.HasPrefix(strings.ToUpper(ov.Status), "CANCEL") && strings.EqualFold(ov.Status, "FILLED") {
		ack.Status = command.CommandStatusCompleted
	}
	ack.ExchangeOrderID = formatOrderID(ov.VenueOrderID)

	_ = s.dedupUpsert(ctx, &models.AgentDedupKey{
		CorrelationKey:  idem,
		CommandID:       cmd.CommandID,
		ClientOrderID:   ov.ClientOrderID,
		ExchangeOrderID: formatOrderID(ov.VenueOrderID),
		LastStatus:      string(ack.Status),
	})

	return s.refreshSnapshot(ctx, cmd.InstanceID, cmd.Symbol, ack, &rep)
}

func parseCancelTarget(cmd command.TradeCommand) CancelIn {
	in := CancelIn{Symbol: cmd.Symbol}
	raw := strings.TrimSpace(cmd.Intent.IntentID)
	if raw == "" {
		return in
	}
	if isAllDigits(raw) {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil && id > 0 {
			in.VenueOrderID = id
			in.VenueOrderIDValid = true
		}
		return in
	}
	in.ClientOrderID = raw
	return in
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (s *Service) computeBaseQty(ctx context.Context, cmd command.TradeCommand) (qty float64, estNotional float64, errmsg string) {
	var targetQty *float64
	if cmd.TargetPosition != nil {
		targetQty = cmd.TargetPosition
	} else if cmd.Intent.TargetPositionQty != nil {
		targetQty = cmd.Intent.TargetPositionQty
	}

	if targetQty != nil {
		q := math.Abs(*targetQty)
		step, err := withRetry(ctx, func(ctx context.Context) (float64, error) {
			return s.Venue.LotStep(ctx, cmd.Symbol)
		})
		if err != nil {
			return 0, 0, fmt.Sprintf("lot step: %v", err)
		}
		qrounded := roundDownToStep(q, step)
		px, perr := withRetry(ctx, func(ctx context.Context) (float64, error) {
			return s.Venue.MarkPrice(ctx, cmd.Symbol)
		})
		if perr != nil || px <= 0 {
			return 0, 0, fmt.Sprintf("mark price: %v", perr)
		}
		return qrounded, qrounded * px, ""
	}

	var not float64
	if cmd.TargetNotional != nil {
		not = math.Abs(*cmd.TargetNotional)
	} else if cmd.Intent.TargetNotionalUSDT != nil {
		not = math.Abs(*cmd.Intent.TargetNotionalUSDT)
	} else {
		return 0, 0, "need target_position or target_notional (command or intent)"
	}

	px, err := withRetry(ctx, func(ctx context.Context) (float64, error) {
		return s.Venue.MarkPrice(ctx, cmd.Symbol)
	})
	if err != nil || px <= 0 {
		return 0, 0, fmt.Sprintf("mark price: %v", err)
	}

	step, serr := withRetry(ctx, func(ctx context.Context) (float64, error) {
		return s.Venue.LotStep(ctx, cmd.Symbol)
	})
	if serr != nil {
		return 0, 0, fmt.Sprintf("lot step: %v", serr)
	}

	q := not / px
	q = roundDownToStep(q, step)
	return q, not, ""
}

func roundDownToStep(q, step float64) float64 {
	if step <= 0 {
		return q
	}
	return math.Floor(q/step) * step
}

func sanitizeClientOID(raw string) string {
	raw = strings.TrimSpace(raw)
	n := 0
	var b strings.Builder
	for _, r := range raw {
		if n >= 36 {
			break
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			n++
		}
	}
	out := strings.ToUpper(b.String())
	if out == "" {
		return sanitizeClientOID(strings.ReplaceAll(strings.ReplaceAll(raw, "-", ""), "_", ""))
	}
	return out
}

func formatOrderID(id int64) string {
	if id == 0 {
		return ""
	}
	return fmt.Sprintf("%d", id)
}

func (s *Service) dedupGet(ctx context.Context, key string) (*models.AgentDedupKey, error) {
	if s == nil || s.Dedup == nil || key == "" {
		return nil, gorm.ErrRecordNotFound
	}
	return s.Dedup.Get(ctx, key)
}

func (s *Service) dedupUpsert(ctx context.Context, row *models.AgentDedupKey) error {
	if s == nil || s.Dedup == nil || row == nil || row.CorrelationKey == "" {
		return nil
	}
	return s.Dedup.Upsert(ctx, row)
}

func (s *Service) refreshSnapshot(ctx context.Context, instanceID string, symbol string, ack command.CommandAck, base *report.DeltaReport) (command.CommandAck, report.DeltaReport) {
	_ = instanceID
	acct, err := withRetry(ctx, func(ctx context.Context) (*VenueAccountView, error) {
		return s.Venue.AccountAndPositions(ctx)
	})
	if err == nil && acct != nil {
		base.Account = &report.AccountSnapshot{
			EquityUSDT:                acct.EquityUSDT,
			WalletBalanceUSDT:         acct.WalletBalanceUSDT,
			AvailableBalanceUSDT:      acct.AvailableBalanceUSDT,
			UsedMarginUSDT:            math.Max(acct.WalletBalanceUSDT-acct.AvailableBalanceUSDT, 0),
			ExchangeAccountTimeUnixMs: acct.ExchangeTimeUnixMs,
		}
		base.Positions = filterPositions(symbol, acct.Positions)
		base.ExchangeEventTimeUnixMs = acct.ExchangeTimeUnixMs
	} else if err != nil {
		base.Errors = append(base.Errors, err.Error())
	}

	oo, oerr := withRetry(ctx, func(ctx context.Context) ([]OpenOrderView, error) {
		return s.Venue.OpenOrders(ctx, "")
	})
	if oerr == nil {
		base.OpenOrders = mapOpenOrders(oo)
	} else {
		base.Errors = append(base.Errors, oerr.Error())
	}

	return ack, *base
}

func filterPositions(symbol string, in []report.PositionSnapshot) []report.PositionSnapshot {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	if sym == "" {
		return in
	}
	out := make([]report.PositionSnapshot, 0)
	for _, p := range in {
		if strings.EqualFold(p.Symbol, sym) {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return in
	}
	return out
}

func mapOpenOrders(in []OpenOrderView) []report.OpenOrderSnapshot {
	out := make([]report.OpenOrderSnapshot, 0, len(in))
	for _, o := range in {
		out = append(out, report.OpenOrderSnapshot{
			ExchangeOrderID:          formatOrderID(o.VenueOrderID),
			Symbol:                   o.Symbol,
			Side:                     o.Side,
			Price:                    o.Price,
			Quantity:                 o.Quantity,
			FilledQty:                o.FilledQty,
			Status:                   o.Status,
			ReduceOnly:               o.ReduceOnly,
			ExchangeUpdateTimeUnixMs: o.UpdateUnixMs,
		})
	}
	return out
}

func (s *Service) fillRepFromOrder(rep *report.DeltaReport, ov *OrderView) {
	if rep == nil || ov == nil {
		return
	}
	rep.Details["exchange_order_id"] = formatOrderID(ov.VenueOrderID)
	rep.Details["order_status"] = ov.Status
}

func mergeFills(dst *[]report.FillRecord, extra []report.FillRecord) {
	if dst == nil || len(extra) == 0 {
		return
	}
	*dst = ws.MergeFillSnapshots(*dst, extra)
}

func mapVenueStatusToCommand(st string) command.CommandStatus {
	switch strings.ToUpper(strings.TrimSpace(st)) {
	case "FILLED":
		return command.CommandStatusCompleted
	case "NEW", "PARTIALLY_FILLED":
		return command.CommandStatusWorking
	case "CANCELED", "CANCELLED":
		return command.CommandStatusCanceled
	case "EXPIRED":
		return command.CommandStatusExpired
	case "REJECTED":
		return command.CommandStatusRejected
	default:
		return command.CommandStatusWorking
	}
}

func retrySuggestion(err error) string {
	var api *binance.APIError
	if !binance.AsAPIError(err, &api) {
		return ""
	}
	if api.RetryReason != "" {
		return api.RetryReason
	}
	if api.Retryable {
		return "retry_with_backoff"
	}
	return ""
}

func backoffVenue(ctx context.Context, fn func(context.Context) error) error {
	var last error
	delay := 200 * time.Millisecond
	for attempt := 0; attempt < 4; attempt++ {
		last = fn(ctx)
		if last == nil {
			return nil
		}
		var api *binance.APIError
		if binance.AsAPIError(last, &api) && api.Retryable {
			sleep := delay
			if api.RetryAfter > 0 {
				sleep = api.RetryAfter
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(sleep):
			}
			delay *= 2
			if delay > 3*time.Second {
				delay = 3 * time.Second
			}
			continue
		}
		return last
	}
	return last
}

func withRetry[T any](ctx context.Context, call func(context.Context) (T, error)) (T, error) {
	var zero T
	var lastErr error
	delay := 150 * time.Millisecond
	for attempt := 0; attempt < 4; attempt++ {
		v, err := call(ctx)
		if err == nil {
			return v, nil
		}
		lastErr = err
		var api *binance.APIError
		if binance.AsAPIError(err, &api) && api.Retryable {
			sleep := delay
			if api.RetryAfter > 0 {
				sleep = api.RetryAfter
			}
			select {
			case <-ctx.Done():
				return zero, ctx.Err()
			case <-time.After(sleep):
			}
			delay *= 2
			continue
		}
		return zero, err
	}
	return zero, lastErr
}
