package saas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/JerrrMi/quant/internal/backtest"
	"github.com/JerrrMi/quant/internal/config"
	"github.com/JerrrMi/quant/internal/infra/db/models"
	"github.com/JerrrMi/quant/internal/infra/db/repository"
)

// backtestRequestSnapshot 与控制台表单/前端 `BacktestRequestDTO` 字段对齐，存入 BacktestJob.RequestJSON。
type backtestRequestSnapshot struct {
	SourceKind string `json:"source_kind"` // template | instance
	TemplateID uint   `json:"template_id"`
	InstanceID uint   `json:"instance_id"`

	Symbol         string `json:"symbol"`
	MarketKind     string `json:"market_kind"`
	DataProvider   string `json:"data_provider"`
	DataPath       string `json:"data_path"`
	WindowStart    string `json:"window_start"`
	WindowEnd      string `json:"window_end"`
	WarmupBars     int    `json:"warmup_bars"`
	BarStride      int    `json:"bar_stride"`
	InitialQuote   string `json:"initial_quote"`
	Currency       string `json:"currency"`
	MakerBps       float64 `json:"maker_bps"`
	TakerBps       float64 `json:"taker_bps"`
	UseTakerFees   bool   `json:"use_taker_fees"`
	SlippageBps    float64 `json:"slippage_bps"`
	FundingBpsPerDay float64 `json:"funding_bps_per_day"`
	LPPLEnabled    bool   `json:"lppl_enabled"`
	LPPLBubble01   float64 `json:"lppl_bubble_metric_01"`
	LPPLJobID      string `json:"lppl_job_id"`
	FailureRate    float64 `json:"failure_rate"`
	RNGSeed        int64  `json:"rng_seed"`

	ExternalFeatures *struct {
		LPPL *bool `json:"lppl,omitempty"`
	} `json:"external_features,omitempty"`
}

type backtestProgressDTO struct {
	Done   int     `json:"done"`
	Total  int     `json:"total"`
	Pct01  float64 `json:"pct_01"`
}

func modelParamsFromTemplateJSON(raw []byte, kind string) config.ModelParams {
	out := config.ModelParams{
		ID:     strings.TrimSpace(kind),
		Values: map[string]float64{},
	}
	if len(raw) == 0 {
		if out.ID == "" {
			out.ID = "backtest_model"
		}
		return out
	}
	var top map[string]json.RawMessage
	if json.Unmarshal(raw, &top) != nil {
		if out.ID == "" {
			out.ID = "backtest_model"
		}
		return out
	}
	td, ok := top["template_defaults"]
	if !ok {
		if out.ID == "" {
			out.ID = "backtest_model"
		}
		return out
	}
	var def map[string]float64
	if json.Unmarshal(td, &def) != nil || def == nil {
		if out.ID == "" {
			out.ID = "backtest_model"
		}
		return out
	}
	out.Values = def
	if out.ID == "" {
		out.ID = "backtest_model"
	}
	return out
}

func (snap *backtestRequestSnapshot) effectiveLPPL() (enabled bool, bubble float64, jobID string) {
	enabled = snap.LPPLEnabled
	if snap.ExternalFeatures != nil && snap.ExternalFeatures.LPPL != nil && *snap.ExternalFeatures.LPPL {
		enabled = true
	}
	return enabled, snap.LPPLBubble01, strings.TrimSpace(snap.LPPLJobID)
}

func (snap *backtestRequestSnapshot) toBacktestConfig(tpl *models.Strategy) (config.BacktestConfig, error) {
	var cfg config.BacktestConfig
	cfg.Data.Provider = strings.TrimSpace(snap.DataProvider)
	cfg.Data.Path = strings.TrimSpace(snap.DataPath)
	cfg.Data.Symbol = strings.TrimSpace(strings.ToUpper(snap.Symbol))
	cfg.Fees.MakerBps = snap.MakerBps
	cfg.Fees.TakerBps = snap.TakerBps
	cfg.Slippage.Bps = snap.SlippageBps
	cfg.Replay.Window.Start = strings.TrimSpace(snap.WindowStart)
	cfg.Replay.Window.End = strings.TrimSpace(snap.WindowEnd)
	cfg.Replay.WarmupBars = snap.WarmupBars
	if snap.BarStride > 0 {
		cfg.Replay.BarStride = snap.BarStride
	}
	cfg.Capital.InitialQuote = strings.TrimSpace(snap.InitialQuote)
	cfg.Capital.Currency = strings.TrimSpace(snap.Currency)
	cfg.Logging.Level = "info"

	cfg.Model = modelParamsFromTemplateJSON([]byte(tpl.ConfigJSON), tpl.Kind)

	cfg.Simulation.UseTakerFees = snap.UseTakerFees
	cfg.Simulation.FailureRate = snap.FailureRate
	cfg.Simulation.LotStep = 0.001
	cfg.Simulation.CommandDeadlineMs = 120_000
	cfg.Simulation.FundingBpsPerDay = snap.FundingBpsPerDay
	cfg.Simulation.RNGSeed = snap.RNGSeed

	lpplOn, bubble, jobID := snap.effectiveLPPL()
	cfg.LPPL.Enabled = lpplOn
	cfg.LPPL.BubbleMetric01 = bubble
	cfg.LPPL.JobID = jobID

	if err := cfg.Validate(); err != nil {
		return config.BacktestConfig{}, err
	}
	return cfg, nil
}

func (snap *backtestRequestSnapshot) validate(resolver string) error {
	sk := strings.TrimSpace(strings.ToLower(snap.SourceKind))
	if sk != "template" && sk != "instance" {
		return fmt.Errorf("source_kind must be template or instance")
	}
	if sk == "template" && snap.TemplateID == 0 {
		return fmt.Errorf("template_id required")
	}
	if sk == "instance" && snap.InstanceID == 0 {
		return fmt.Errorf("instance_id required")
	}
	if strings.TrimSpace(snap.Symbol) == "" {
		return fmt.Errorf("symbol required")
	}
	mk := strings.TrimSpace(strings.ToLower(snap.MarketKind))
	if mk != "spot" && mk != "futures" {
		return fmt.Errorf("market_kind must be spot or futures")
	}
	if strings.TrimSpace(snap.DataProvider) == "" || strings.TrimSpace(snap.DataPath) == "" {
		return fmt.Errorf("data_provider and data_path required")
	}
	if strings.TrimSpace(snap.InitialQuote) == "" || strings.TrimSpace(snap.Currency) == "" {
		return fmt.Errorf("initial_quote and currency required")
	}
	_ = resolver
	return nil
}

func (h *ConsoleHandlers) resolveBacktestSnapshot(ctx context.Context, userID uint, snap *backtestRequestSnapshot) (
	tpl *models.Strategy,
	inst *models.Instance,
	tplName string,
	err error,
) {
	strRepo := repository.NewGormStrategyRepository(h.DB)
	instRepo := repository.NewGormInstanceRepository(h.DB)

	switch strings.TrimSpace(strings.ToLower(snap.SourceKind)) {
	case "template":
		tpl, err = strRepo.GetByID(ctx, snap.TemplateID)
		if err != nil || tpl == nil || !tpl.IsCatalog {
			return nil, nil, "", fmt.Errorf("template not found")
		}
		return tpl, nil, tpl.Name, nil
	case "instance":
		inst, err = instRepo.GetByID(ctx, snap.InstanceID)
		if err != nil || inst == nil || inst.UserID != userID {
			return nil, nil, "", fmt.Errorf("instance not found")
		}
		tpl, err = strRepo.GetByID(ctx, inst.StrategyID)
		if err != nil || tpl == nil || !tpl.IsCatalog {
			return nil, nil, "", fmt.Errorf("template not found")
		}
		return tpl, inst, tpl.Name, nil
	default:
		return nil, nil, "", fmt.Errorf("invalid source_kind")
	}
}

func (h *ConsoleHandlers) listBacktests(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := h.consoleUserID(ctx)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "console user missing — run SaaS seed")
		return
	}
	var rows []models.BacktestJob
	if err := h.DB.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("id desc").
		Limit(500).
		Find(&rows).Error; err != nil {
		h.writeErr(w, http.StatusInternalServerError, "list backtests failed")
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for i := range rows {
		j := &rows[i]
		item := h.jobToListItem(j)
		out = append(out, item)
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"jobs": out})
}

func (h *ConsoleHandlers) jobToListItem(j *models.BacktestJob) map[string]any {
	var durMs *int64
	if j.StartedAt != nil && j.FinishedAt != nil && j.FinishedAt.After(*j.StartedAt) {
		ms := j.FinishedAt.Sub(*j.StartedAt).Milliseconds()
		durMs = &ms
	}
	var progress any
	if strings.TrimSpace(j.ProgressJSON) != "" {
		_ = json.Unmarshal([]byte(j.ProgressJSON), &progress)
	}
	m := map[string]any{
		"id":             j.ID,
		"status":         j.Status,
		"template_id":    j.TemplateID,
		"template_name":  j.TemplateName,
		"instance_id":    j.InstanceID,
		"symbol":         j.Symbol,
		"market_kind":    j.MarketKind,
		"window_start":   j.WindowStart,
		"window_end":     j.WindowEnd,
		"created_at":     j.CreatedAt.UTC().Format(time.RFC3339),
		"updated_at":     j.UpdatedAt.UTC().Format(time.RFC3339),
		"started_at":     rfcPtr(j.StartedAt),
		"finished_at":    rfcPtr(j.FinishedAt),
		"duration_ms":    durMs,
		"progress":       progress,
		"error_message":  j.ErrorMessage,
	}
	return m
}

func rfcPtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func (h *ConsoleHandlers) createBacktest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := h.consoleUserID(ctx)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "console user missing — run SaaS seed")
		return
	}
	var snap backtestRequestSnapshot
	if err := json.NewDecoder(r.Body).Decode(&snap); err != nil {
		h.writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}
	snap.Symbol = strings.TrimSpace(strings.ToUpper(snap.Symbol))
	snap.MarketKind = strings.TrimSpace(strings.ToLower(snap.MarketKind))
	snap.SourceKind = strings.TrimSpace(strings.ToLower(snap.SourceKind))

	if err := snap.validate("create"); err != nil {
		h.writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	tpl, inst, tplName, err := h.resolveBacktestSnapshot(ctx, userID, &snap)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if !tpl.AllowBacktest {
		h.writeErr(w, http.StatusBadRequest, "template forbids backtest")
		return
	}
	markets := splitMarketsCSV(tpl.Markets)
	if !sliceContains(markets, snap.MarketKind) {
		h.writeErr(w, http.StatusBadRequest, "template does not support this market_kind")
		return
	}
	if _, err := snap.toBacktestConfig(tpl); err != nil {
		h.writeErr(w, http.StatusBadRequest, fmt.Sprintf("invalid backtest params: %v", err))
		return
	}
	snap.TemplateID = tpl.ID
	if inst != nil {
		snap.InstanceID = inst.ID
	}
	reqBytes, err := json.Marshal(snap)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "request marshal failed")
		return
	}
	var instPtr *uint
	if inst != nil {
		u := inst.ID
		instPtr = &u
	}
	job := models.BacktestJob{
		UserID:       userID,
		Status:       "pending",
		TemplateID:   tpl.ID,
		TemplateName: tplName,
		InstanceID:   instPtr,
		Symbol:       snap.Symbol,
		MarketKind:   snap.MarketKind,
		WindowStart:  snap.WindowStart,
		WindowEnd:    snap.WindowEnd,
		RequestJSON:  string(reqBytes),
		LogJSON:      "[]",
		ProgressJSON: "",
	}
	if err := h.DB.WithContext(ctx).Create(&job).Error; err != nil {
		if h.Log != nil {
			h.Log.Error("console create backtest", "err", err)
		}
		h.writeErr(w, http.StatusInternalServerError, "create backtest job failed")
		return
	}
	go h.executeBacktestJob(job.ID)
	h.writeJSON(w, http.StatusCreated, map[string]any{"id": job.ID, "status": job.Status})
}

func (h *ConsoleHandlers) getBacktest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := h.consoleUserID(ctx)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "console user missing — run SaaS seed")
		return
	}
	idStr := r.PathValue("id")
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "invalid job id")
		return
	}
	var job models.BacktestJob
	if err := h.DB.WithContext(ctx).First(&job, uint(id64)).Error; err != nil || job.UserID != userID {
		h.writeErr(w, http.StatusNotFound, "backtest not found")
		return
	}
	var reqObj any
	_ = json.Unmarshal([]byte(job.RequestJSON), &reqObj)
	var repObj any
	if strings.TrimSpace(job.ReportJSON) != "" {
		_ = json.Unmarshal([]byte(job.ReportJSON), &repObj)
	}
	var logs []string
	_ = json.Unmarshal([]byte(job.LogJSON), &logs)
	var progress any
	if strings.TrimSpace(job.ProgressJSON) != "" {
		_ = json.Unmarshal([]byte(job.ProgressJSON), &progress)
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"job": map[string]any{
			"id":             job.ID,
			"status":         job.Status,
			"template_id":    job.TemplateID,
			"template_name":  job.TemplateName,
			"instance_id":    job.InstanceID,
			"symbol":         job.Symbol,
			"market_kind":    job.MarketKind,
			"window_start":   job.WindowStart,
			"window_end":     job.WindowEnd,
			"created_at":     job.CreatedAt.UTC().Format(time.RFC3339),
			"updated_at":     job.UpdatedAt.UTC().Format(time.RFC3339),
			"started_at":     rfcPtr(job.StartedAt),
			"finished_at":    rfcPtr(job.FinishedAt),
			"duration_ms":    durationMs(job.StartedAt, job.FinishedAt),
			"progress":       progress,
			"error_message":  job.ErrorMessage,
			"request":        reqObj,
		},
		"report": repObj,
		"logs":   logs,
	})
}

func durationMs(started, finished *time.Time) any {
	if started == nil || finished == nil || !finished.After(*started) {
		return nil
	}
	return finished.Sub(*started).Milliseconds()
}

type backtestActionBody struct {
	Action string `json:"action"`
}

func (h *ConsoleHandlers) backtestActions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := h.consoleUserID(ctx)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "console user missing — run SaaS seed")
		return
	}
	idStr := r.PathValue("id")
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "invalid job id")
		return
	}
	var body backtestActionBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}
	act := strings.TrimSpace(strings.ToLower(body.Action))

	var job models.BacktestJob
	if err := h.DB.WithContext(ctx).First(&job, uint(id64)).Error; err != nil || job.UserID != userID {
		h.writeErr(w, http.StatusNotFound, "backtest not found")
		return
	}

	switch act {
	case "pause", "terminate", "cancel":
		if job.Status != "running" {
			h.writeJSON(w, http.StatusOK, map[string]string{"status": "noop"})
			return
		}
		if v, ok := h.backtestCancel.Load(job.ID); ok {
			if cancel, ok2 := v.(context.CancelFunc); ok2 {
				cancel()
			}
		}
		h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	case "rerun":
		var snap backtestRequestSnapshot
		if err := json.Unmarshal([]byte(job.RequestJSON), &snap); err != nil {
			h.writeErr(w, http.StatusBadRequest, "cannot parse stored request")
			return
		}
		tpl, inst, tplName, err := h.resolveBacktestSnapshot(ctx, userID, &snap)
		if err != nil {
			h.writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if !tpl.AllowBacktest {
			h.writeErr(w, http.StatusBadRequest, "template forbids backtest")
			return
		}
		reqBytes, _ := json.Marshal(snap)
		var instPtr *uint
		if inst != nil {
			u := inst.ID
			instPtr = &u
		}
		newJob := models.BacktestJob{
			UserID:       userID,
			Status:       "pending",
			TemplateID:   tpl.ID,
			TemplateName: tplName,
			InstanceID:   instPtr,
			Symbol:       snap.Symbol,
			MarketKind:   snap.MarketKind,
			WindowStart:  snap.WindowStart,
			WindowEnd:    snap.WindowEnd,
			RequestJSON:  string(reqBytes),
			LogJSON:      "[]",
		}
		if err := h.DB.WithContext(ctx).Create(&newJob).Error; err != nil {
			h.writeErr(w, http.StatusInternalServerError, "rerun create failed")
			return
		}
		go h.executeBacktestJob(newJob.ID)
		h.writeJSON(w, http.StatusOK, map[string]any{"id": newJob.ID, "status": newJob.Status})
	default:
		h.writeErr(w, http.StatusBadRequest, "unknown action")
	}
}

func (h *ConsoleHandlers) executeBacktestJob(jobID uint) {
	ctxBG := context.Background()
	log := h.Log
	if log == nil {
		log = slog.Default()
	}
	var snap backtestRequestSnapshot

	runCtx, cancel := context.WithCancel(context.Background())
	h.backtestCancel.Store(jobID, cancel)
	defer func() {
		cancel()
		h.backtestDeleteCancel(jobID)
	}()

	var job models.BacktestJob
	if err := h.DB.WithContext(ctxBG).First(&job, jobID).Error; err != nil {
		return
	}
	if err := json.Unmarshal([]byte(job.RequestJSON), &snap); err != nil {
		h.finishBacktestJob(jobID, "failed", "", err.Error(), nil)
		return
	}
	userID := job.UserID
	tpl, _, _, err := h.resolveBacktestSnapshot(ctxBG, userID, &snap)
	if err != nil {
		h.finishBacktestJob(jobID, "failed", "", err.Error(), nil)
		return
	}
	cfg, err := snap.toBacktestConfig(tpl)
	if err != nil {
		h.finishBacktestJob(jobID, "failed", "", err.Error(), nil)
		return
	}

	now := time.Now().UTC()
	_ = h.DB.WithContext(ctxBG).Model(&models.BacktestJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":     "running",
		"started_at": &now,
		"updated_at": now,
	}).Error
	h.appendJobLog(jobID, "回测已开始")

	var lastTick atomic.Int64
	progress := func(done, total int) {
		if total <= 0 {
			return
		}
		ms := time.Now().UnixMilli()
		prev := lastTick.Load()
		if done < total && ms-prev < 320 {
			return
		}
		lastTick.Store(ms)
		pct := float64(done) / float64(total)
		b, _ := json.Marshal(backtestProgressDTO{Done: done, Total: total, Pct01: pct})
		_ = h.DB.WithContext(ctxBG).Model(&models.BacktestJob{}).Where("id = ?", jobID).Updates(map[string]any{
			"progress_json": string(b),
			"updated_at":    time.Now().UTC(),
		}).Error
	}

	rep, err := backtest.BacktestFromConfig(runCtx, cfg, log, progress)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			h.finishBacktestJob(jobID, "cancelled", "", "任务已取消或暂停", nil)
			return
		}
		h.finishBacktestJob(jobID, "failed", "", err.Error(), nil)
		return
	}
	rb, err := json.Marshal(rep)
	if err != nil {
		h.finishBacktestJob(jobID, "failed", "", "report marshal failed", nil)
		return
	}
	h.finishBacktestJob(jobID, "finished", string(rb), "", rep)
}

func (h *ConsoleHandlers) backtestDeleteCancel(jobID uint) {
	h.backtestCancel.Delete(jobID)
}

func (h *ConsoleHandlers) finishBacktestJob(jobID uint, status, reportJSON, errMsg string, rep *backtest.BacktestReport) {
	ctx := context.Background()
	now := time.Now().UTC()
	fields := map[string]any{
		"status":      status,
		"updated_at":  now,
		"finished_at": &now,
	}
	if reportJSON != "" {
		fields["report_json"] = reportJSON
	}
	if errMsg != "" {
		if len(errMsg) > 500 {
			errMsg = errMsg[:500]
		}
		fields["error_message"] = errMsg
	}
	_ = h.DB.WithContext(ctx).Model(&models.BacktestJob{}).Where("id = ?", jobID).Updates(fields).Error

	msg := fmt.Sprintf("回测结束：%s", status)
	if rep != nil {
		msg = fmt.Sprintf("回测完成：total_return=%.4f max_dd=%.4f win_rate=%.4f",
			rep.Metrics.TotalReturn, rep.Metrics.MaxDrawdown01, rep.Metrics.WinRate)
	}
	h.appendJobLog(jobID, msg)
}

func (h *ConsoleHandlers) appendJobLog(jobID uint, line string) {
	ctx := context.Background()
	var job models.BacktestJob
	if err := h.DB.WithContext(ctx).First(&job, jobID).Error; err != nil {
		return
	}
	var logs []string
	_ = json.Unmarshal([]byte(job.LogJSON), &logs)
	logs = append(logs, time.Now().UTC().Format(time.RFC3339)+"  "+line)
	const maxLines = 400
	if len(logs) > maxLines {
		logs = logs[len(logs)-maxLines:]
	}
	b, err := json.Marshal(logs)
	if err != nil {
		return
	}
	_ = h.DB.WithContext(ctx).Model(&models.BacktestJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"log_json":   string(b),
		"updated_at": time.Now().UTC(),
	}).Error
}
