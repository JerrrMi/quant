// Package config 承载各进程的配置结构体与从 YAML 加载逻辑。
// 由 cmd/* 在启动时调用 Load*；策略包不引用本包。
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// SaaSConfig 是控制面进程配置：HTTP、SaaS 侧 DB 等。
// 不得包含交易所 API Key；由 cmd/saas 加载后传入 app 层。
type SaaSConfig struct {
	HTTP struct {
		// ListenAddr 是 HTTP/WebSocket 服务监听地址，如 "127.0.0.1:8080"。
		ListenAddr string `yaml:"listen_addr"`
	} `yaml:"http"`

	Database struct {
		// DSN 供 GORM 打开数据库（SQLite 文件或 PostgreSQL URL 由 infra 解释）。
		DSN string `yaml:"dsn"`
	} `yaml:"database"`

	Logging struct {
		// Level 是 slog 级别名：debug|info|warn|error。
		Level string `yaml:"level"`
	} `yaml:"logging"`
}

// AgentConfig 是执行端配置：SaaS 地址、交易所名等。
// 密钥仅通过环境变量注入，不出现在本结构体字段中（防误序列化进 yaml）。
type AgentConfig struct {
	// SaasWSURL 是 Agent 连接控制面 WebSocket 的 URL。
	SaasWSURL string `yaml:"saas_ws_url"`

	Exchange struct {
		// Name 标识目标场所，如 "binance"（具体路由在 executor/infra 实现）。
		Name string `yaml:"name"`
	} `yaml:"exchange"`

	Logging struct {
		Level string `yaml:"level"`
	} `yaml:"logging"`
}

// AppConfig 预留多模式单二进制场景；当前各 cmd 分别使用 SaaSConfig/AgentConfig。
// 若未来合并入口，可将共有字段收拢于此。
type AppConfig struct {
	Mode string `yaml:"mode"` // "saas" | "agent" | "backtest" — 仅占位
}

// LoadSaaSConfig 从 path 读取 YAML，返回 SaaSConfig。
// 调用方：cmd/saas/run。
func LoadSaaSConfig(path string) (SaaSConfig, error) {
	var c SaaSConfig
	if err := loadYAML(path, &c); err != nil {
		return SaaSConfig{}, err
	}
	return c, nil
}

// LoadAgentConfig 从 path 读取 YAML，返回 AgentConfig。
// 调用方：cmd/agent/run。
func LoadAgentConfig(path string) (AgentConfig, error) {
	var c AgentConfig
	if err := loadYAML(path, &c); err != nil {
		return AgentConfig{}, err
	}
	return c, nil
}

func loadYAML(path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config %q: %w", path, err)
	}
	if err := yaml.Unmarshal(b, out); err != nil {
		return fmt.Errorf("parse yaml %q: %w", path, err)
	}
	return nil
}
