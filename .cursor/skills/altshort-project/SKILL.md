---
name: altshort-project
description: AltShort 量化仓库的结构、约束与协作流程；涉及本仓库 Go、配置、入口、回测、策略、WebSocket 时启用。
---

# AltShort 工程 Skill

## 名称与用途

用于 **AltShort** 做空山寨币量化系统的协作与实现：对齐目录、领域模型、测试与安全边界，避免策略污染执行层与密钥扩散。

## 适用范围

- 本仓库内 `cmd/`、`internal/`、`configs/`、`docs/`、`.cursor/rules/` 相关改动。  
- Go 代码、测试、启动器、回测骨架、策略接口、Agent↔SaaS WebSocket 协议讨论与实现。

## 触发条件（自动启用）

当任务涉及以下任一时，**先读本 Skill 再动手**：

- 目录结构、包边界、领域类型  
- Go 代码、测试、Makefile/脚本、配置与启动器  
- 回测引擎、历史数据加载、策略 `Step()`  
- WebSocket 或 Binance 集成、Agent 编排  

## 工作流程

1. 阅读 `CLAUDE.md`、`AGENTS.md` 与 `.cursor/rules/*`。  
2. 定位并阅读相关 **canonical** 源文件（如 `internal/config/config.go`、`internal/app/bootstrap.go`、对应 `cmd/*/main.go`）。  
3. 确认本次 **单一清晰任务**；列出将修改的文件清单（路径级）。  
4. 只做 **最小必要** 的编辑，保持策略接口稳定；不跨 Phase 混改。  

## 输出要求

- **一次只做一个清晰任务**；在动工前列出将修改的文件。  
- 完成后按 `AGENTS.md` 附：新增文件、变更文件、未完成项、下一步建议。  

## 安全与架构检查清单

- [ ] 未引入与 `CLAUDE.md` / rules 冲突的实现（如策略层 I/O、SaaS 侧交易所 Key）。  
- [ ] 未将 **策略逻辑** 写入 `internal/executor`（执行适配层）。  
- [ ] **API Key** 不进入 SaaS 配置、不进入 `cmd/saas` 依赖链。  
- [ ] **未添加** SQL migration 文件；库表演进走 GORM `AutoMigrate`。  
- [ ] 策略相关变更优先考虑 **接口不变** 或显式兼容层。  

## 参考路径（查阅优先）

| 主题 | 文件 |
|------|------|
| 协作与命令 | `CLAUDE.md` |
| Phase 与产出说明 | `AGENTS.md` |
| 分层与域 | `.cursor/rules/00-project-overview.mdc`、`01-go-architecture.mdc` |
| 安全与测试 | `.cursor/rules/02-workflow-and-safety.mdc` |
| 架构说明 | `docs/architecture.md` |
