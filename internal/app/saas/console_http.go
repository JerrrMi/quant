package saas

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	domaincommand "github.com/JerrrMi/quant/internal/domain/command"
	"github.com/JerrrMi/quant/internal/infra/db/models"
	"github.com/JerrrMi/quant/internal/infra/db/repository"
	"github.com/JerrrMi/quant/internal/infra/ws"
	"github.com/JerrrMi/quant/internal/scheduler"
	"gorm.io/gorm"
)

// ConsoleHandlers 暴露控制面临界 REST（模板库与实例编排）；不含策略计算。
type ConsoleHandlers struct {
	DB   *gorm.DB
	Hub  *ws.AgentHub
	Orch *scheduler.StepOrchestrator
	Log  *slog.Logger

	backtestCancel sync.Map // job ID (uint) -> context.CancelFunc
}

// RegisterConsoleRoutes 将 /v1/console/* 注册到主 mux。
func RegisterConsoleRoutes(mux *http.ServeMux, h *ConsoleHandlers) {
	if mux == nil || h == nil || h.DB == nil {
		return
	}
	mux.HandleFunc("GET /v1/console/templates", h.listTemplates)
	mux.HandleFunc("GET /v1/console/templates/{id}", h.getTemplate)
	mux.HandleFunc("GET /v1/console/instances", h.listInstances)
	mux.HandleFunc("POST /v1/console/instances", h.createInstance)
	mux.HandleFunc("GET /v1/console/instances/{id}", h.getInstance)
	mux.HandleFunc("PATCH /v1/console/instances/{id}", h.patchInstance)
	mux.HandleFunc("POST /v1/console/instances/{id}/actions", h.instanceActions)

	mux.HandleFunc("GET /v1/console/backtests", h.listBacktests)
	mux.HandleFunc("POST /v1/console/backtests", h.createBacktest)
	mux.HandleFunc("GET /v1/console/backtests/{id}", h.getBacktest)
	mux.HandleFunc("POST /v1/console/backtests/{id}/actions", h.backtestActions)

	mux.HandleFunc("GET /v1/console/commands", h.listCommands)
	mux.HandleFunc("GET /v1/console/audit", h.listAudit)
}

func (h *ConsoleHandlers) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *ConsoleHandlers) writeErr(w http.ResponseWriter, status int, msg string) {
	h.writeJSON(w, status, map[string]string{"error": msg})
}

func (h *ConsoleHandlers) consoleUserID(ctx context.Context) (uint, error) {
	var u models.User
	if err := h.DB.WithContext(ctx).Where("email = ?", consoleUserEmail).First(&u).Error; err != nil {
		return 0, err
	}
	return u.ID, nil
}

func splitMarketsCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToLower(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (h *ConsoleHandlers) listTemplates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	repo := repository.NewGormStrategyRepository(h.DB)
	rows, err := repo.ListCatalog(ctx, 500)
	if err != nil {
		if h.Log != nil {
			h.Log.Error("console list templates", "err", err)
		}
		h.writeErr(w, http.StatusInternalServerError, "list templates failed")
		return
	}
	type rowDTO struct {
		ID            uint     `json:"id"`
		Name          string   `json:"name"`
		Kind          string   `json:"kind"`
		Markets       []string `json:"markets"`
		Description   string   `json:"description"`
		AllowLive     bool     `json:"allow_live"`
		AllowBacktest bool     `json:"allow_backtest"`
		UpdatedAt     string   `json:"updated_at"`
	}
	out := make([]rowDTO, 0, len(rows))
	for i := range rows {
		s := &rows[i]
		out = append(out, rowDTO{
			ID:            s.ID,
			Name:          s.Name,
			Kind:          s.Kind,
			Markets:       splitMarketsCSV(s.Markets),
			Description:   s.Description,
			AllowLive:     s.AllowLive,
			AllowBacktest: s.AllowBacktest,
			UpdatedAt:     s.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"templates": out})
}

func (h *ConsoleHandlers) getTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "invalid template id")
		return
	}
	repo := repository.NewGormStrategyRepository(h.DB)
	s, err := repo.GetByID(ctx, uint(id64))
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "load template failed")
		return
	}
	if s == nil || !s.IsCatalog {
		h.writeErr(w, http.StatusNotFound, "template not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"id":             s.ID,
		"name":           s.Name,
		"kind":           s.Kind,
		"markets":        splitMarketsCSV(s.Markets),
		"description":    s.Description,
		"allow_live":     s.AllowLive,
		"allow_backtest": s.AllowBacktest,
		"config_json":    json.RawMessage(s.ConfigJSON),
		"updated_at":     s.UpdatedAt.UTC().Format(time.RFC3339),
	})
}

func (h *ConsoleHandlers) listInstances(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := h.consoleUserID(ctx)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "console user missing — run SaaS seed")
		return
	}
	instRepo := repository.NewGormInstanceRepository(h.DB)
	strRepo := repository.NewGormStrategyRepository(h.DB)
	cmdRepo := repository.NewGormCommandRepository(h.DB)
	rows, err := instRepo.ListByUserID(ctx, userID, 500)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "list instances failed")
		return
	}
	type instDTO struct {
		ID                 uint   `json:"id"`
		DisplayName        string `json:"display_name"`
		TemplateID         uint   `json:"template_id"`
		TemplateName       string `json:"template_name"`
		Symbol             string `json:"symbol"`
		MarketKind         string `json:"market_kind"`
		RunMode            string `json:"run_mode"`
		Status             string `json:"status"`
		DerivedRuntime     string `json:"derived_runtime"`
		AgentKey           string `json:"agent_key"`
		AgentConnected     bool   `json:"agent_connected"`
		LastHeartbeatAt    string `json:"last_heartbeat_at,omitempty"`
		LastCommandSummary string `json:"last_command_summary,omitempty"`
		LastReportSummary  string `json:"last_report_summary,omitempty"`
		RiskStatus         string `json:"risk_status"`
		UpdatedAt          string `json:"updated_at"`
	}
	out := make([]instDTO, 0, len(rows))
	for i := range rows {
		inst := &rows[i]
		tpl, _ := strRepo.GetByID(ctx, inst.StrategyID)
		tplName := ""
		if tpl != nil {
			tplName = tpl.Name
		}
		derived, risk := h.deriveRuntimeAndRisk(ctx, inst)
		agentOK := h.Hub != nil && h.Hub.IsConnected(inst.AgentKey)
		lastCmd := ""
		cmds, _ := cmdRepo.ListRecentByInstance(ctx, inst.ID, 1)
		if len(cmds) > 0 {
			lastCmd = summarizeCommand(&cmds[0])
		}
		lastRep := ""
		repRepo := repository.NewGormReportRepository(h.DB)
		reps, _ := repRepo.ListRecentByInstance(ctx, inst.ID, 1)
		if len(reps) > 0 {
			lastRep = summarizeReport(&reps[0])
		}
		hb := ""
		if inst.LastHeartbeatAt != nil {
			hb = inst.LastHeartbeatAt.UTC().Format(time.RFC3339)
		}
		out = append(out, instDTO{
			ID:                 inst.ID,
			DisplayName:        inst.DisplayName,
			TemplateID:         inst.StrategyID,
			TemplateName:       tplName,
			Symbol:             inst.Symbol,
			MarketKind:         inst.MarketKind,
			RunMode:            inst.RunMode,
			Status:             inst.Status,
			DerivedRuntime:     derived,
			AgentKey:           inst.AgentKey,
			AgentConnected:     agentOK,
			LastHeartbeatAt:    hb,
			LastCommandSummary: lastCmd,
			LastReportSummary:  lastRep,
			RiskStatus:         risk,
			UpdatedAt:          inst.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"instances": out})
}

func summarizeCommand(c *models.TradeCommandRecord) string {
	return fmt.Sprintf("%s · %s · %s", c.Kind, c.Status, strings.TrimSpace(truncateRunes(c.PayloadJSON, 120)))
}

func summarizeReport(rep *models.AgentReportRecord) string {
	return fmt.Sprintf("%s · %s", rep.ReportType, strings.TrimSpace(truncateRunes(rep.PayloadJSON, 120)))
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func (h *ConsoleHandlers) deriveRuntimeAndRisk(ctx context.Context, inst *models.Instance) (runtime string, risk string) {
	runs := repository.NewGormStrategyRunRepository(h.DB)
	run, _ := runs.GetRunningByInstance(ctx, inst.ID)
	risk = riskFromParams(inst.ParamsJSON)

	agentOK := h.Hub != nil && h.Hub.IsConnected(inst.AgentKey)
	switch {
	case !agentOK:
		return "agent_disconnected", risk
	case inst.Status == "paused" || inst.Status == "draining":
		return "paused", risk
	case run != nil && run.Status == "running":
		return "running", risk
	default:
		return "idle", risk
	}
}

func riskFromParams(paramsJSON string) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(paramsJSON), &m); err != nil || len(m) == 0 {
		return "未配置（请在实例参数中填写风控字段）"
	}
	if v, ok := m["risk_summary"].(string); ok && strings.TrimSpace(v) != "" {
		return v
	}
	parts := []string{}
	if v, ok := m["max_daily_loss_quote"]; ok {
		parts = append(parts, fmt.Sprintf("单日亏损上限 %v", v))
	}
	if v, ok := m["max_position_fraction"]; ok {
		parts = append(parts, fmt.Sprintf("仓位上限比例 %v", v))
	}
	if len(parts) == 0 {
		return "已从实例参数加载（未标注摘要字段）"
	}
	return strings.Join(parts, " · ")
}

type createInstanceBody struct {
	DisplayName string         `json:"display_name"`
	StrategyID  uint           `json:"strategy_id"`
	Symbol      string         `json:"symbol"`
	MarketKind  string         `json:"market_kind"`
	RunMode     string         `json:"run_mode"`
	AgentKey    string         `json:"agent_key"`
	Params      map[string]any `json:"params"`
}

func (h *ConsoleHandlers) createInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := h.consoleUserID(ctx)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "console user missing — run SaaS seed")
		return
	}
	var body createInstanceBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}
	body.Symbol = strings.TrimSpace(strings.ToUpper(body.Symbol))
	body.MarketKind = strings.TrimSpace(strings.ToLower(body.MarketKind))
	body.RunMode = strings.TrimSpace(strings.ToLower(body.RunMode))
	body.AgentKey = strings.TrimSpace(body.AgentKey)
	body.DisplayName = strings.TrimSpace(body.DisplayName)
	if body.DisplayName == "" || body.StrategyID == 0 || body.Symbol == "" || body.MarketKind == "" || body.RunMode == "" || body.AgentKey == "" {
		h.writeErr(w, http.StatusBadRequest, "missing required fields")
		return
	}
	if body.MarketKind != "spot" && body.MarketKind != "futures" {
		h.writeErr(w, http.StatusBadRequest, "market_kind must be spot or futures")
		return
	}
	if body.RunMode != "backtest" && body.RunMode != "paper" && body.RunMode != "live" {
		h.writeErr(w, http.StatusBadRequest, "run_mode must be backtest, paper or live")
		return
	}
	strRepo := repository.NewGormStrategyRepository(h.DB)
	tpl, err := strRepo.GetByID(ctx, body.StrategyID)
	if err != nil || tpl == nil || !tpl.IsCatalog {
		h.writeErr(w, http.StatusBadRequest, "strategy_id must reference a catalog template")
		return
	}
	markets := splitMarketsCSV(tpl.Markets)
	if !sliceContains(markets, body.MarketKind) {
		h.writeErr(w, http.StatusBadRequest, "template does not support this market_kind")
		return
	}
	if body.RunMode == "live" && !tpl.AllowLive {
		h.writeErr(w, http.StatusBadRequest, "template forbids live trading")
		return
	}
	if body.RunMode == "backtest" && !tpl.AllowBacktest {
		h.writeErr(w, http.StatusBadRequest, "template forbids backtest mode")
		return
	}
	paramsBytes, err := json.Marshal(body.Params)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "params must be json-serializable")
		return
	}
	if len(paramsBytes) > 64_000 {
		h.writeErr(w, http.StatusBadRequest, "params too large")
		return
	}

	row := models.Instance{
		UserID:      userID,
		StrategyID:  tpl.ID,
		AgentKey:    body.AgentKey,
		DisplayName: body.DisplayName,
		Symbol:      body.Symbol,
		MarketKind:  body.MarketKind,
		RunMode:     body.RunMode,
		ParamsJSON:  string(paramsBytes),
		Status:      "paused",
	}
	instRepo := repository.NewGormInstanceRepository(h.DB)
	if err := instRepo.Create(ctx, &row); err != nil {
		if h.Log != nil {
			h.Log.Error("console create instance", "err", err)
		}
		h.writeErr(w, http.StatusInternalServerError, "create instance failed")
		return
	}
	audit := repository.NewGormAuditRepository(h.DB)
	_ = audit.Append(ctx, &models.AuditEvent{
		ActorType:    "user",
		ActorID:      strconv.FormatUint(uint64(userID), 10),
		Action:       "console.instance.create",
		ResourceType: "instance",
		ResourceID:   strconv.FormatUint(uint64(row.ID), 10),
		PayloadJSON:  string(paramsBytes),
		OccurredAt:   time.Now().UTC(),
	})
	h.writeJSON(w, http.StatusCreated, map[string]any{"id": row.ID})
}

func sliceContains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func (h *ConsoleHandlers) getInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := h.consoleUserID(ctx)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "console user missing — run SaaS seed")
		return
	}
	idStr := r.PathValue("id")
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "invalid instance id")
		return
	}
	instRepo := repository.NewGormInstanceRepository(h.DB)
	inst, err := instRepo.GetByID(ctx, uint(id64))
	if err != nil || inst == nil || inst.UserID != userID {
		h.writeErr(w, http.StatusNotFound, "instance not found")
		return
	}
	strRepo := repository.NewGormStrategyRepository(h.DB)
	tpl, _ := strRepo.GetByID(ctx, inst.StrategyID)
	derived, risk := h.deriveRuntimeAndRisk(ctx, inst)
	agentOK := h.Hub != nil && h.Hub.IsConnected(inst.AgentKey)

	cmdRepo := repository.NewGormCommandRepository(h.DB)
	cmds, _ := cmdRepo.ListRecentByInstance(ctx, inst.ID, 15)
	cmdSym := inst.Symbol
	cmdViews := make([]map[string]any, 0, len(cmds))
	for i := range cmds {
		c := &cmds[i]
		cmdViews = append(cmdViews, h.commandToConsoleView(c, cmdSym))
	}

	repRepo := repository.NewGormReportRepository(h.DB)
	reps, _ := repRepo.ListRecentByInstance(ctx, inst.ID, 15)
	repViews := make([]map[string]string, 0, len(reps))
	lastErr := ""
	for i := range reps {
		x := &reps[i]
		if lastErr == "" && x.ReportType == "execution_summary" {
			var payload map[string]any
			if json.Unmarshal([]byte(x.PayloadJSON), &payload) == nil {
				if em, ok := payload["last_error"].(string); ok && em != "" {
					lastErr = em
				}
			}
		}
		repViews = append(repViews, map[string]string{
			"id":          x.ID,
			"type":        x.ReportType,
			"received_at": x.ReceivedAt.UTC().Format(time.RFC3339),
			"summary":     summarizeReport(x),
		})
	}

	auditRepo := repository.NewGormAuditRepository(h.DB)
	timeline, _ := auditRepo.ListRecentForResource(ctx, "instance", strconv.FormatUint(uint64(inst.ID), 10), 40)

	tlViews := make([]map[string]string, 0, len(timeline))
	for i := range timeline {
		ev := &timeline[i]
		tlViews = append(tlViews, map[string]string{
			"action":      ev.Action,
			"occurred_at": ev.OccurredAt.UTC().Format(time.RFC3339),
			"payload":     truncateRunes(ev.PayloadJSON, 200),
		})
	}

	var tplConfig json.RawMessage
	tplDesc := ""
	tplMarkets := []string{}
	tplKind := ""
	tplName := ""
	if tpl != nil {
		tplConfig = json.RawMessage(tpl.ConfigJSON)
		tplDesc = tpl.Description
		tplMarkets = splitMarketsCSV(tpl.Markets)
		tplKind = tpl.Kind
		tplName = tpl.Name
	}

	var instParams json.RawMessage
	if strings.TrimSpace(inst.ParamsJSON) != "" {
		instParams = json.RawMessage(inst.ParamsJSON)
	}

	hb := ""
	if inst.LastHeartbeatAt != nil {
		hb = inst.LastHeartbeatAt.UTC().Format(time.RFC3339)
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"id":                   inst.ID,
		"display_name":         inst.DisplayName,
		"template_id":          inst.StrategyID,
		"template_name":        tplName,
		"template_kind":        tplKind,
		"template_markets":     tplMarkets,
		"template_description": tplDesc,
		"template_config_json": tplConfig,
		"symbol":               inst.Symbol,
		"market_kind":          inst.MarketKind,
		"run_mode":             inst.RunMode,
		"status":               inst.Status,
		"derived_runtime":      derived,
		"agent_key":            inst.AgentKey,
		"agent_connected":      agentOK,
		"last_heartbeat_at":    hb,
		"risk_status":          risk,
		"instance_params_json": instParams,
		"recent_commands":      cmdViews,
		"recent_reports":       repViews,
		"recent_errors_hint":   lastErr,
		"timeline":             tlViews,
		"updated_at":           inst.UpdatedAt.UTC().Format(time.RFC3339),
	})
}

type patchInstanceBody struct {
	DisplayName *string        `json:"display_name"`
	Symbol      *string        `json:"symbol"`
	MarketKind  *string        `json:"market_kind"`
	RunMode     *string        `json:"run_mode"`
	AgentKey    *string        `json:"agent_key"`
	Params      map[string]any `json:"params"`
}

func (h *ConsoleHandlers) patchInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := h.consoleUserID(ctx)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "console user missing — run SaaS seed")
		return
	}
	idStr := r.PathValue("id")
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "invalid instance id")
		return
	}
	instRepo := repository.NewGormInstanceRepository(h.DB)
	inst, err := instRepo.GetByID(ctx, uint(id64))
	if err != nil || inst == nil || inst.UserID != userID {
		h.writeErr(w, http.StatusNotFound, "instance not found")
		return
	}
	var body patchInstanceBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if body.DisplayName != nil {
		inst.DisplayName = strings.TrimSpace(*body.DisplayName)
	}
	if body.Symbol != nil {
		inst.Symbol = strings.TrimSpace(strings.ToUpper(*body.Symbol))
	}
	if body.MarketKind != nil {
		mk := strings.TrimSpace(strings.ToLower(*body.MarketKind))
		if mk != "spot" && mk != "futures" {
			h.writeErr(w, http.StatusBadRequest, "market_kind must be spot or futures")
			return
		}
		inst.MarketKind = mk
	}
	if body.RunMode != nil {
		rm := strings.TrimSpace(strings.ToLower(*body.RunMode))
		if rm != "backtest" && rm != "paper" && rm != "live" {
			h.writeErr(w, http.StatusBadRequest, "run_mode invalid")
			return
		}
		strRepo := repository.NewGormStrategyRepository(h.DB)
		tpl, err := strRepo.GetByID(ctx, inst.StrategyID)
		if err != nil || tpl == nil {
			h.writeErr(w, http.StatusBadRequest, "template missing")
			return
		}
		if rm == "live" && !tpl.AllowLive {
			h.writeErr(w, http.StatusBadRequest, "template forbids live trading")
			return
		}
		if rm == "backtest" && !tpl.AllowBacktest {
			h.writeErr(w, http.StatusBadRequest, "template forbids backtest mode")
			return
		}
		inst.RunMode = rm
	}
	if body.AgentKey != nil {
		inst.AgentKey = strings.TrimSpace(*body.AgentKey)
	}
	if body.Params != nil {
		b, err := json.Marshal(body.Params)
		if err != nil {
			h.writeErr(w, http.StatusBadRequest, "params invalid")
			return
		}
		if len(b) > 64_000 {
			h.writeErr(w, http.StatusBadRequest, "params too large")
			return
		}
		inst.ParamsJSON = string(b)
	}
	if err := instRepo.Update(ctx, inst); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "update failed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type actionBody struct {
	Action string `json:"action"`
}

func (h *ConsoleHandlers) instanceActions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := h.consoleUserID(ctx)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "console user missing — run SaaS seed")
		return
	}
	idStr := r.PathValue("id")
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "invalid instance id")
		return
	}
	var body actionBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}
	action := strings.TrimSpace(strings.ToLower(body.Action))
	instRepo := repository.NewGormInstanceRepository(h.DB)
	inst, err := instRepo.GetByID(ctx, uint(id64))
	if err != nil || inst == nil || inst.UserID != userID {
		h.writeErr(w, http.StatusNotFound, "instance not found")
		return
	}
	runs := repository.NewGormStrategyRunRepository(h.DB)
	audit := repository.NewGormAuditRepository(h.DB)

	appendAudit := func(act string, payload any) {
		var b []byte
		if payload != nil {
			b, _ = json.Marshal(payload)
		}
		_ = audit.Append(ctx, &models.AuditEvent{
			ActorType:    "user",
			ActorID:      strconv.FormatUint(uint64(userID), 10),
			Action:       act,
			ResourceType: "instance",
			ResourceID:   strconv.FormatUint(uint64(inst.ID), 10),
			PayloadJSON:  string(b),
			OccurredAt:   time.Now().UTC(),
		})
	}

	switch action {
	case "start":
		inst.Status = "active"
		if err := instRepo.Update(ctx, inst); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "persist failed")
			return
		}
		if _, err := runs.EnsureRunningRun(ctx, inst.ID, inst.StrategyID); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "ensure run failed")
			return
		}
		appendAudit("console.instance.start", nil)
	case "stop":
		inst.Status = "paused"
		if err := instRepo.Update(ctx, inst); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "persist failed")
			return
		}
		_ = runs.StopRunningRunsForInstance(ctx, inst.ID)
		appendAudit("console.instance.stop", nil)
	case "pause":
		inst.Status = "paused"
		if err := instRepo.Update(ctx, inst); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "persist failed")
			return
		}
		_ = runs.StopRunningRunsForInstance(ctx, inst.ID)
		appendAudit("console.instance.pause", nil)
	case "resume":
		inst.Status = "active"
		if err := instRepo.Update(ctx, inst); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "persist failed")
			return
		}
		if _, err := runs.EnsureRunningRun(ctx, inst.ID, inst.StrategyID); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "ensure run failed")
			return
		}
		appendAudit("console.instance.resume", nil)
	case "restart":
		_ = runs.StopRunningRunsForInstance(ctx, inst.ID)
		inst.Status = "active"
		if err := instRepo.Update(ctx, inst); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "persist failed")
			return
		}
		if _, err := runs.EnsureRunningRun(ctx, inst.ID, inst.StrategyID); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "ensure run failed")
			return
		}
		appendAudit("console.instance.restart", nil)
	case "run_once":
		if h.Orch == nil {
			h.writeErr(w, http.StatusServiceUnavailable, "orchestrator unavailable")
			return
		}
		appendAudit("console.instance.run_once.request", nil)
		if err := h.Orch.RunOnce(ctx, inst.ID); err != nil {
			if h.Log != nil {
				h.Log.Warn("run_once failed", "instance_id", inst.ID, "err", err)
			}
			h.writeErr(w, http.StatusBadRequest, fmt.Sprintf("run_once failed: %v", err))
			return
		}
		appendAudit("console.instance.run_once.done", nil)
	case "terminate":
		_ = runs.StopRunningRunsForInstance(ctx, inst.ID)
		if err := instRepo.SoftDelete(ctx, inst.ID); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "terminate failed")
			return
		}
		appendAudit("console.instance.terminate", nil)
	default:
		h.writeErr(w, http.StatusBadRequest, "unknown action")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *ConsoleHandlers) commandToConsoleView(c *models.TradeCommandRecord, instSymbol string) map[string]any {
	var tc domaincommand.TradeCommand
	_ = json.Unmarshal([]byte(c.PayloadJSON), &tc)
	symbol := instSymbol
	if strings.TrimSpace(tc.Symbol) != "" {
		symbol = tc.Symbol
	}
	intent := ""
	if tc.Intent.IntentID != "" {
		intent = fmt.Sprintf("%s · %s · ro=%v", tc.Intent.IntentID, tc.Intent.Side, tc.Intent.IsReduceOnly)
	}
	dispatch := ""
	if c.DispatchedAt != nil {
		dispatch = c.DispatchedAt.UTC().Format(time.RFC3339)
	}
	ack := ""
	if c.AckedAt != nil {
		ack = c.AckedAt.UTC().Format(time.RFC3339)
	}
	reportAt := ""
	if !c.UpdatedAt.IsZero() {
		reportAt = c.UpdatedAt.UTC().Format(time.RFC3339)
	}
	return map[string]any{
		"command_id":    c.ID,
		"instance_id":   c.InstanceID,
		"symbol":        symbol,
		"intent":        intent,
		"kind":          c.Kind,
		"status":        c.Status,
		"issued_at":     c.CreatedAt.UTC().Format(time.RFC3339),
		"dispatched_at": dispatch,
		"acked_at":      ack,
		"report_at":     reportAt,
		"error":         c.ErrorMessage,
		"summary":       summarizeCommand(c),
		// 兼容旧前端字段
		"id":         c.ID,
		"created_at": c.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (h *ConsoleHandlers) listCommands(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := h.consoleUserID(ctx)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "console user missing — run SaaS seed")
		return
	}
	limit := 50
	if s := strings.TrimSpace(r.URL.Query().Get("limit")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			limit = n
		}
	}
	var filterInst *uint
	if s := strings.TrimSpace(r.URL.Query().Get("instance_id")); s != "" {
		n64, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			h.writeErr(w, http.StatusBadRequest, "invalid instance_id")
			return
		}
		instRepo := repository.NewGormInstanceRepository(h.DB)
		inst, err := instRepo.GetByID(ctx, uint(n64))
		if err != nil || inst == nil || inst.UserID != userID {
			h.writeErr(w, http.StatusNotFound, "instance not found")
			return
		}
		u := uint(n64)
		filterInst = &u
	}
	instRepo := repository.NewGormInstanceRepository(h.DB)
	insts, err := instRepo.ListByUserID(ctx, userID, 500)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "list instances failed")
		return
	}
	symByID := map[uint]string{}
	for i := range insts {
		symByID[insts[i].ID] = insts[i].Symbol
	}
	cmdRepo := repository.NewGormCommandRepository(h.DB)
	rows, err := cmdRepo.ListRecentForConsoleUser(ctx, userID, filterInst, limit)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "list commands failed")
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for i := range rows {
		sym := symByID[rows[i].InstanceID]
		out = append(out, h.commandToConsoleView(&rows[i], sym))
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"commands":    out,
		"server_time": time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *ConsoleHandlers) listAudit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := h.consoleUserID(ctx)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "console user missing — run SaaS seed")
		return
	}
	instRepo := repository.NewGormInstanceRepository(h.DB)
	insts, err := instRepo.ListByUserID(ctx, userID, 500)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "list instances failed")
		return
	}
	q := r.URL.Query()
	agentKey := strings.TrimSpace(q.Get("agent_key"))
	instanceFilter := strings.TrimSpace(q.Get("instance_id"))

	var scoped []uint
	for i := range insts {
		in := &insts[i]
		if agentKey != "" && in.AgentKey != agentKey {
			continue
		}
		if instanceFilter != "" {
			if strconv.FormatUint(uint64(in.ID), 10) != instanceFilter {
				continue
			}
		}
		scoped = append(scoped, in.ID)
	}

	flt := repository.AuditConsoleListFilter{
		ActionPrefix: strings.TrimSpace(q.Get("action")),
		ResourceType: strings.TrimSpace(q.Get("module")),
		ResourceID:   strings.TrimSpace(q.Get("resource_id")),
		ActorID:      strings.TrimSpace(q.Get("actor_id")),
		Level:        strings.TrimSpace(q.Get("level")),
	}
	if s := strings.TrimSpace(q.Get("limit")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			flt.Limit = n
		}
	}
	if ts := strings.TrimSpace(q.Get("from")); ts != "" {
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			h.writeErr(w, http.StatusBadRequest, "invalid from (use RFC3339)")
			return
		}
		flt.Since = &t
	}
	if ts := strings.TrimSpace(q.Get("to")); ts != "" {
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			h.writeErr(w, http.StatusBadRequest, "invalid to (use RFC3339)")
			return
		}
		flt.Until = &t
	}

	auditRepo := repository.NewGormAuditRepository(h.DB)
	rows, err := auditRepo.ListConsoleVisible(ctx, scoped, flt)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "list audit failed")
		return
	}
	type rowDTO struct {
		ID           uint   `json:"id"`
		ActorType    string `json:"actor_type"`
		ActorID      string `json:"actor_id"`
		Action       string `json:"action"`
		ResourceType string `json:"resource_type"`
		ResourceID   string `json:"resource_id"`
		Level        string `json:"level"`
		Module       string `json:"module"`
		PayloadJSON  string `json:"payload_json"`
		OccurredAt   string `json:"occurred_at"`
	}
	out := make([]rowDTO, 0, len(rows))
	for i := range rows {
		ev := &rows[i]
		lvl := auditLevelHeuristic(ev.Action)
		mod := ev.ResourceType
		if mod == "" {
			mod = "unknown"
		}
		out = append(out, rowDTO{
			ID:           ev.ID,
			ActorType:    ev.ActorType,
			ActorID:      ev.ActorID,
			Action:       ev.Action,
			ResourceType: ev.ResourceType,
			ResourceID:   ev.ResourceID,
			Level:        lvl,
			Module:       mod,
			PayloadJSON:  ev.PayloadJSON,
			OccurredAt:   ev.OccurredAt.UTC().Format(time.RFC3339),
		})
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"events":      out,
		"server_time": time.Now().UTC().Format(time.RFC3339),
	})
}

func auditLevelHeuristic(action string) string {
	a := strings.ToLower(action)
	if strings.Contains(a, "fail") || strings.Contains(a, "error") || strings.Contains(a, "terminate") {
		return "error"
	}
	if strings.Contains(a, "warn") || strings.Contains(a, "pause") || strings.Contains(a, "drain") {
		return "warn"
	}
	return "info"
}
