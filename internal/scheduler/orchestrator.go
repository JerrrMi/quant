// Package scheduler 负责定时编排：组装 Step 输入、调用纯策略、持久化与下发命令；不含择时公式与交易所直连。
package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/JerrrMi/quant/internal/config"
	"github.com/JerrrMi/quant/internal/domain/command"
	"github.com/JerrrMi/quant/internal/domain/strategy"
	"github.com/JerrrMi/quant/internal/infra/db/models"
	"github.com/JerrrMi/quant/internal/infra/db/repository"
	"github.com/JerrrMi/quant/internal/infra/lppl"
	"github.com/JerrrMi/quant/internal/infra/marketdata"
)

// CommandDispatcher 将已落库的指令投递到 Agent 通道（如 WebSocket）；仅编排，不做策略判断。
type CommandDispatcher interface {
	Dispatch(ctx context.Context, instance *models.Instance, cmd command.TradeCommand) error
}

// StepOrchestrator 单次决策链编排器：数据装配 → Step → 命令持久化与派发。
type StepOrchestrator struct {
	Logger       *slog.Logger
	Stepper      strategy.Stepper
	Bars         *marketdata.DBBarSeriesReader
	LPPL         *lppl.InputAugmentor
	Runs         repository.StrategyRunRepository
	Instances    repository.InstanceRepository
	Commands     repository.CommandRepository
	Audit        repository.AuditRepository
	Dispatcher   CommandDispatcher
	Model        config.ModelParams
	DefaultSym   string
	Deadline     time.Duration
}

// Tick 对全部 active 实例执行一轮 Step 链路。
func (o *StepOrchestrator) Tick(ctx context.Context) error {
	if o == nil {
		return fmt.Errorf("scheduler: nil orchestrator")
	}
	log := o.Logger
	if log == nil {
		log = slog.Default()
	}
	if o.Stepper == nil {
		return fmt.Errorf("scheduler: stepper is nil")
	}
	if o.Instances == nil || o.Runs == nil {
		return fmt.Errorf("scheduler: missing repositories")
	}
	instances, err := o.Instances.ListActive(ctx)
	if err != nil {
		return err
	}
	windowBars := 96
	if o.Model.Values != nil {
		if v, ok := o.Model.Values["signal_lookback"]; ok {
			windowBars = int(v)
		}
	}
	if windowBars <= 0 {
		windowBars = 96
	}
	featSpec := marketdata.DefaultFeatureWindowSpec(windowBars)
	deadline := o.Deadline
	if deadline <= 0 {
		deadline = 2 * time.Minute
	}
	for i := range instances {
		inst := &instances[i]
		if err := o.tickInstance(ctx, inst, featSpec, deadline, log); err != nil {
			log.Warn("scheduler tick instance failed", "instance_id", inst.ID, "err", err)
		}
	}
	return nil
}

func (o *StepOrchestrator) tickInstance(ctx context.Context, inst *models.Instance, featSpec marketdata.FeatureWindowSpec, deadline time.Duration, log *slog.Logger) error {
	symbol := o.DefaultSym
	if symbol == "" {
		return fmt.Errorf("scheduler: empty default symbol")
	}
	run, err := o.Runs.EnsureRunningRun(ctx, inst.ID, inst.StrategyID)
	if err != nil {
		return err
	}
	if run == nil {
		return fmt.Errorf("scheduler: strategy run not available")
	}
	nextSeq := run.LastStepSequence + 1
	now := time.Now().UTC()
	nowMs := now.UnixMilli()

	if o.Bars == nil {
		return fmt.Errorf("scheduler: bars reader is nil")
	}
	window, err := o.Bars.RecentClosedBars(ctx, symbol, featSpec.WindowBars+1)
	if err != nil {
		return err
	}
	if len(window) < 2 {
		return o.skipNoData(ctx, inst, run, symbol, nextSeq, nowMs, "no_market_window")
	}

	barCurrent := window[len(window)-1]
	priorClose := 0.0
	if pc, ok := marketdata.PriorClose(window); ok {
		priorClose = pc
	}
	features, err := marketdata.BuildFeatureSnapshot(window, featSpec)
	if err != nil {
		return err
	}

	in := strategy.AltShortStrategyInput{
		Symbol:              symbol,
		NetPositionQty:      0,
		ShortOpenedAtUnixMs: 0,
		PriorBarClose:       priorClose,
		BarCurrent:          barCurrent,
		Features:            features,
		Risk:                strategy.RiskSnapshot{},
		NowUnixMs:           nowMs,
		StepSequence:        nextSeq,
	}
	if o.LPPL != nil {
		_ = o.LPPL.ApplyLatest(ctx, &in, symbol)
	}
	if err := o.traceInput(ctx, inst, run, in); err != nil {
		log.Warn("audit step input failed", "err", err)
	}

	out, err := o.Stepper.Step(in)
	if err != nil {
		return err
	}
	if err := o.Runs.UpdateLastStepSequence(ctx, run.ID, nextSeq); err != nil {
		return err
	}
	if err := o.traceOutput(ctx, inst, run, nextSeq, out); err != nil {
		log.Warn("audit step output failed", "err", err)
	}
	return o.persistAndDispatchIntents(ctx, inst, run, in, out, nowMs, deadline, log)
}

func (o *StepOrchestrator) skipNoData(ctx context.Context, inst *models.Instance, run *models.StrategyRun, symbol string, nextSeq int64, nowMs int64, reason string) error {
	if o.Audit == nil {
		return fmt.Errorf("marketdata: need %s for symbol %s", reason, symbol)
	}
	payload, _ := json.Marshal(map[string]any{
		"reason":       reason,
		"symbol":       symbol,
		"instance_id":  inst.ID,
		"run_id":       run.ID,
		"step_seq":     nextSeq,
		"now_unix_ms":  nowMs,
	})
	return o.Audit.Append(ctx, &models.AuditEvent{
		ActorType:    "system",
		ActorID:      "scheduler",
		Action:       "strategy.step_skipped",
		ResourceType: "instance",
		ResourceID:   strconv.FormatUint(uint64(inst.ID), 10),
		PayloadJSON:  string(payload),
		OccurredAt:   time.Now().UTC(),
	})
}

func (o *StepOrchestrator) traceInput(ctx context.Context, inst *models.Instance, run *models.StrategyRun, in strategy.AltShortStrategyInput) error {
	if o.Audit == nil {
		return nil
	}
	b, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return o.Audit.Append(ctx, &models.AuditEvent{
		ActorType:    "system",
		ActorID:      "scheduler",
		Action:       "strategy.step_input",
		ResourceType: "strategy_run",
		ResourceID:   strconv.FormatUint(uint64(run.ID), 10),
		PayloadJSON:  string(b),
		OccurredAt:   time.Now().UTC(),
	})
}

func (o *StepOrchestrator) traceOutput(ctx context.Context, inst *models.Instance, run *models.StrategyRun, _ int64, out strategy.AltShortStrategyOutput) error {
	_ = inst
	if o.Audit == nil {
		return nil
	}
	b, err := json.Marshal(out)
	if err != nil {
		return err
	}
	return o.Audit.Append(ctx, &models.AuditEvent{
		ActorType:    "system",
		ActorID:      "scheduler",
		Action:       "strategy.step_output",
		ResourceType: "strategy_run",
		ResourceID:   strconv.FormatUint(uint64(run.ID), 10),
		PayloadJSON:  string(b),
		OccurredAt:   time.Now().UTC(),
	})
}

func (o *StepOrchestrator) persistAndDispatchIntents(
	ctx context.Context,
	inst *models.Instance,
	run *models.StrategyRun,
	in strategy.AltShortStrategyInput,
	out strategy.AltShortStrategyOutput,
	nowMs int64,
	deadline time.Duration,
	log *slog.Logger,
) error {
	if len(out.Intents) == 0 {
		return nil
	}
	if o.Commands == nil {
		return fmt.Errorf("scheduler: command repo nil")
	}
	strategyIDStr := strconv.FormatUint(uint64(inst.StrategyID), 10)
	instanceIDStr := strconv.FormatUint(uint64(inst.ID), 10)
	runID := run.ID
	for i := range out.Intents {
		intent := out.Intents[i]
		idem := command.IdempotencyKey(fmt.Sprintf("inst:%d:run:%d:step:%d:intent:%s", inst.ID, runID, in.StepSequence, intent.IntentID))
		existing, err := o.Commands.GetByCorrelationID(ctx, string(idem))
		if err != nil {
			return err
		}
		if existing != nil && existing.Status == "dispatched" {
			continue
		}

		var tc command.TradeCommand
		row := models.TradeCommandRecord{}
		if existing != nil {
			if err := json.Unmarshal([]byte(existing.PayloadJSON), &tc); err != nil {
				return err
			}
			row = *existing
		} else {
			cmdID := uuid.NewString()
			tc = command.TradeCommand{
				CommandID:      cmdID,
				InstanceID:     instanceIDStr,
				StrategyID:     strategyIDStr,
				Symbol:         in.Symbol,
				Side:           intent.Side,
				Intent:         intent,
				TargetNotional: intent.TargetNotionalUSDT,
				TargetPosition: intent.TargetPositionQty,
				ReduceOnly:     intent.IsReduceOnly,
				DeadlineUnixMs: nowMs + deadline.Milliseconds(),
				Nonce:          uuid.NewString(),
				IdempotencyKey: idem,
				Kind:           command.CommandKindPlace,
			}
			payload, err := json.Marshal(tc)
			if err != nil {
				return err
			}
			row = models.TradeCommandRecord{
				ID:            cmdID,
				CorrelationID: string(idem),
				InstanceID:    inst.ID,
				StrategyRunID: &runID,
				Kind:          string(tc.Kind),
				Status:        "pending",
				PayloadJSON:   string(payload),
			}
			if err := o.Commands.SaveCommand(ctx, &row); err != nil {
				return err
			}
			if o.Audit != nil {
				auditPayload, _ := json.Marshal(map[string]any{"command_id": tc.CommandID, "correlation_id": string(idem)})
				_ = o.Audit.Append(ctx, &models.AuditEvent{
					ActorType:    "system",
					ActorID:      "scheduler",
					Action:       "command.persisted",
					ResourceType: "trade_command",
					ResourceID:   tc.CommandID,
					PayloadJSON:  string(auditPayload),
					OccurredAt:   time.Now().UTC(),
				})
			}
		}

		if o.Dispatcher != nil {
			if err := o.Dispatcher.Dispatch(ctx, inst, tc); err != nil {
				log.Warn("command dispatch failed", "command_id", tc.CommandID, "err", err)
				continue
			}
			t := time.Now().UTC()
			row.Status = "dispatched"
			row.DispatchedAt = &t
			if err := o.Commands.SaveCommand(ctx, &row); err != nil {
				return err
			}
			if o.Audit != nil {
				auditPayload, _ := json.Marshal(map[string]any{"command_id": tc.CommandID})
				_ = o.Audit.Append(ctx, &models.AuditEvent{
					ActorType:    "system",
					ActorID:      "scheduler",
					Action:       "command.dispatched",
					ResourceType: "trade_command",
					ResourceID:   tc.CommandID,
					PayloadJSON:  string(auditPayload),
					OccurredAt:   time.Now().UTC(),
				})
			}
		}
	}
	return nil
}
