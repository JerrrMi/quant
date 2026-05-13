// Package config 承载各进程的配置结构体与从 YAML 加载逻辑。
// 由 cmd/* 在启动时调用 Load*；策略包不引用本包。
package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// --- SaaS（控制面）---

// SaaSConfig 是控制面进程配置：数据库、缓存、WebSocket 监听、调度与模型元数据。
// 不得包含交易所 API Key 或任何仅属于 Agent 的密钥材料。
type SaaSConfig struct {
	Database struct {
		// DSN 供 GORM 打开数据库（SQLite 文件或 PostgreSQL URL 由 infra 解释）。
		DSN string `yaml:"dsn"`
	} `yaml:"database"`

	Redis struct {
		// Enable 为 true 时后续 Phase 将在此 DSN/地址上建立缓存客户端。
		Enable bool `yaml:"enable"`
		// Addr 为 host:port，例如 "127.0.0.1:6379"。
		Addr string `yaml:"addr"`
		// DB 为 Redis 逻辑库索引。
		DB int `yaml:"db"`
		// Username 为 ACL 用户名；无 ACL 时留空。
		Username string `yaml:"username"`
		// PasswordEnv 为 **环境变量名**，其值才是 Redis 密码；不要把密码写进 YAML。
		PasswordEnv string `yaml:"password_env"`
	} `yaml:"redis"`

	WebSocket struct {
		// ListenAddr 为控制面 WebSocket 监听地址，如 "0.0.0.0:8080"。
		ListenAddr string `yaml:"listen_addr"`
		// AgentPath 为 Agent 入站路径，如 "/v1/agent"。
		AgentPath string `yaml:"agent_path"`
	} `yaml:"websocket"`

	Scheduler struct {
		// Enable 控制是否注册全局调度节拍（骨架阶段仅占位）。
		Enable bool `yaml:"enable"`
		// TickIntervalSecs 为策略评估节拍周期（秒）；启用调度时须为正。
		TickIntervalSecs int `yaml:"tick_interval_seconds"`
	} `yaml:"scheduler"`

	// Model 描述与推理服务或注册表关联的模型标识及无量纲超参。
	Model ModelParams `yaml:"model"`

	Logging struct {
		// Level 是 slog 级别名：debug|info|warn|error。
		Level string `yaml:"level"`
	} `yaml:"logging"`
}

// Validate 检查 SaaS 必填项与明显无效组合。
func (c *SaaSConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("saas config: nil")
	}
	if strings.TrimSpace(c.WebSocket.ListenAddr) == "" {
		return fmt.Errorf("saas config: websocket.listen_addr is required")
	}
	if strings.TrimSpace(c.Logging.Level) == "" {
		return fmt.Errorf("saas config: logging.level is required")
	}
	if c.Redis.Enable && strings.TrimSpace(c.Redis.Addr) == "" {
		return fmt.Errorf("saas config: redis.addr is required when redis.enable is true")
	}
	if c.Scheduler.Enable && c.Scheduler.TickIntervalSecs <= 0 {
		return fmt.Errorf("saas config: scheduler.tick_interval_seconds must be positive when scheduler.enable is true")
	}
	return nil
}

// --- Agent（执行端）---

// AgentConfig 是执行端配置：场所连接、重连、风控以及与密钥相关的 **环境变量名**。
// 真实 API Key/Secret 只能通过环境注入，不得出现在 YAML 明文字段中。
type AgentConfig struct {
	Binance struct {
		// APIKeyEnv 为读取 API Key 的环境变量名（例如 "ALTSHORT_BINANCE_API_KEY"）。
		APIKeyEnv string `yaml:"api_key_env"`
		// APISecretEnv 为读取 API Secret 的环境变量名。
		APISecretEnv string `yaml:"api_secret_env"`
		// UseTestnet 为 true 时连接测试网端点（由 executor/infra 解释）。
		UseTestnet bool `yaml:"use_testnet"`
	} `yaml:"binance"`

	Connection struct {
		// SaasWSURL 是 Agent 连接控制面 WebSocket 的 URL。
		SaasWSURL string `yaml:"saas_ws_url"`
		// LocalBindAddr 为出站本地绑定地址占位，如 "0.0.0.0:0"。
		LocalBindAddr string `yaml:"local_bind_addr"`
		// DialTimeoutSecs 为建立 SaaS WebSocket 的拨号超时（秒）。
		DialTimeoutSecs int `yaml:"dial_timeout_seconds"`
		// TLSInsecureSkipVerify 仅用于开发环境；生产应为 false。
		TLSInsecureSkipVerify bool `yaml:"tls_insecure_skip_verify"`
	} `yaml:"connection"`

	Exchange struct {
		// Name 标识目标场所，如 "binance"。
		Name string `yaml:"name"`
	} `yaml:"exchange"`

	Reconnect struct {
		// MaxAttempts 为连续重连最大尝试次数；0 表示使用默认策略占位。
		MaxAttempts int `yaml:"max_attempts"`
		// InitialBackoffSecs 为首次重连退避时间（秒）。
		InitialBackoffSecs int `yaml:"initial_backoff_seconds"`
		// MaxBackoffSecs 为退避上限（秒）。
		MaxBackoffSecs int `yaml:"max_backoff_seconds"`
		// JitterRatio 为 [0,1] 间抖动比例，避免惊群。
		JitterRatio float64 `yaml:"jitter_ratio"`
	} `yaml:"reconnect"`

	Risk struct {
		// MaxOpenOrders 为同时未终结订单数上限。
		MaxOpenOrders int `yaml:"max_open_orders"`
		// MaxNotionalQuotePerOrder 为单笔名义金额上限（报价货币，字符串避免精度问题）。
		MaxNotionalQuotePerOrder string `yaml:"max_notional_quote_per_order"`
		// MaxDailyLossQuote 为当日最大可接受亏损（占位，由风控模块解释）。
		MaxDailyLossQuote string `yaml:"max_daily_loss_quote"`
	} `yaml:"risk"`

	LocalStore struct {
		// SQLiteDSN 为 Agent 本地状态库；空表示内存 SQLite（骨架默认）。
		SQLiteDSN string `yaml:"sqlite_dsn"`
	} `yaml:"local_store"`

	Logging struct {
		Level string `yaml:"level"`
	} `yaml:"logging"`
}

// Validate 检查 Agent 必填项。
func (c *AgentConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("agent config: nil")
	}
	if strings.TrimSpace(c.Binance.APIKeyEnv) == "" {
		return fmt.Errorf("agent config: binance.api_key_env is required (name of env var, not the secret)")
	}
	if strings.TrimSpace(c.Binance.APISecretEnv) == "" {
		return fmt.Errorf("agent config: binance.api_secret_env is required (name of env var, not the secret)")
	}
	if strings.TrimSpace(c.Connection.SaasWSURL) == "" {
		return fmt.Errorf("agent config: connection.saas_ws_url is required")
	}
	if strings.TrimSpace(c.Exchange.Name) == "" {
		return fmt.Errorf("agent config: exchange.name is required")
	}
	if strings.TrimSpace(c.Logging.Level) == "" {
		return fmt.Errorf("agent config: logging.level is required")
	}
	if c.Risk.MaxOpenOrders < 0 {
		return fmt.Errorf("agent config: risk.max_open_orders must be non-negative")
	}
	if c.Reconnect.JitterRatio < 0 || c.Reconnect.JitterRatio > 1 {
		return fmt.Errorf("agent config: reconnect.jitter_ratio must be in [0,1]")
	}
	if c.Reconnect.MaxAttempts < 0 {
		return fmt.Errorf("agent config: reconnect.max_attempts must be non-negative")
	}
	if c.Connection.DialTimeoutSecs < 0 {
		return fmt.Errorf("agent config: connection.dial_timeout_seconds must be non-negative")
	}
	if c.Reconnect.InitialBackoffSecs < 0 || c.Reconnect.MaxBackoffSecs < 0 {
		return fmt.Errorf("agent config: reconnect backoff seconds must be non-negative")
	}
	if c.Reconnect.MaxBackoffSecs > 0 && c.Reconnect.InitialBackoffSecs > c.Reconnect.MaxBackoffSecs {
		return fmt.Errorf("agent config: reconnect.initial_backoff_seconds must not exceed max_backoff_seconds")
	}
	return nil
}

// --- Backtest ---

// BacktestConfig 回测进程配置：数据源、成本模型、回放窗口与资金。
type BacktestConfig struct {
	Data struct {
		// Provider 为数据源类型：file|database|parquet（具体实现由 internal/backtest 接入）。
		Provider string `yaml:"provider"`
		// Path 为本地根路径或清单文件；database 模式下可为 DSN 或查询描述占位。
		Path string `yaml:"path"`
		// Symbol 为默认回测交易对，如 "BTCUSDT"。
		Symbol string `yaml:"symbol"`
	} `yaml:"data"`

	Fees struct {
		// MakerBps 为做市手续费（基点）。
		MakerBps float64 `yaml:"maker_bps"`
		// TakerBps 为吃单手续费（基点）。
		TakerBps float64 `yaml:"taker_bps"`
	} `yaml:"fees"`

	Slippage struct {
		// Bps 为成交价上的滑点（基点，占位）。
		Bps float64 `yaml:"bps"`
	} `yaml:"slippage"`

	Replay struct {
		// Window 描述回放时间窗；零值表示使用数据全集。
		Window TimeWindow `yaml:"window"`
		// WarmupBars 为正式统计前额外加载的预热 K 线根数。
		WarmupBars int `yaml:"warmup_bars"`
	} `yaml:"replay"`

	Capital struct {
		// InitialQuote 为初始资金（报价货币）。
		InitialQuote string `yaml:"initial_quote"`
		// Currency 为资金币种代码，如 "USDT"。
		Currency string `yaml:"currency"`
	} `yaml:"capital"`

	Model ModelParams `yaml:"model"`

	Logging struct {
		Level string `yaml:"level"`
	} `yaml:"logging"`
}

// TimeWindow 表示半开区间 [Start, End)；空字符串表示不限制该端。
type TimeWindow struct {
	Start string `yaml:"start"`
	End   string `yaml:"end"`
}

// Validate 检查回测必填项。
func (c *BacktestConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("backtest config: nil")
	}
	if strings.TrimSpace(c.Data.Provider) == "" {
		return fmt.Errorf("backtest config: data.provider is required")
	}
	if strings.TrimSpace(c.Data.Path) == "" {
		return fmt.Errorf("backtest config: data.path is required")
	}
	if strings.TrimSpace(c.Data.Symbol) == "" {
		return fmt.Errorf("backtest config: data.symbol is required")
	}
	if c.Fees.MakerBps < 0 || c.Fees.TakerBps < 0 {
		return fmt.Errorf("backtest config: fees must be non-negative")
	}
	if c.Slippage.Bps < 0 {
		return fmt.Errorf("backtest config: slippage.bps must be non-negative")
	}
	if strings.TrimSpace(c.Capital.InitialQuote) == "" {
		return fmt.Errorf("backtest config: capital.initial_quote is required")
	}
	if strings.TrimSpace(c.Capital.Currency) == "" {
		return fmt.Errorf("backtest config: capital.currency is required")
	}
	if strings.TrimSpace(c.Logging.Level) == "" {
		return fmt.Errorf("backtest config: logging.level is required")
	}
	if c.Replay.WarmupBars < 0 {
		return fmt.Errorf("backtest config: replay.warmup_bars must be non-negative")
	}
	return nil
}

// ModelParams 描述模型标识与无量纲超参（具体键由策略/模型实现解释）。
type ModelParams struct {
	ID string `yaml:"id"`
	// Values 为浮点超参；需要离散选项时可在后续扩展为 map[string]any。
	Values map[string]float64 `yaml:"values"`
}

// LoadSaaSConfig 从 path 读取 YAML 并返回 SaaSConfig。
// 调用方：cmd/saas/run。
func LoadSaaSConfig(path string) (SaaSConfig, error) {
	var c SaaSConfig
	if err := loadYAML("saas", path, &c); err != nil {
		return SaaSConfig{}, err
	}
	return c, nil
}

// LoadAgentConfig 从 path 读取 YAML 并返回 AgentConfig。
// 调用方：cmd/agent/run。
func LoadAgentConfig(path string) (AgentConfig, error) {
	var c AgentConfig
	if err := loadYAML("agent", path, &c); err != nil {
		return AgentConfig{}, err
	}
	return c, nil
}

// LoadBacktestConfig 从 path 读取 YAML 并返回 BacktestConfig。
// 调用方：cmd/backtest/run。
func LoadBacktestConfig(path string) (BacktestConfig, error) {
	var c BacktestConfig
	if err := loadYAML("backtest", path, &c); err != nil {
		return BacktestConfig{}, err
	}
	return c, nil
}

func loadYAML(kind, path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s config file %q: read: %w", kind, path, err)
	}
	if err := yaml.Unmarshal(b, out); err != nil {
		return fmt.Errorf("%s config file %q: parse yaml: %w", kind, path, err)
	}
	return nil
}
