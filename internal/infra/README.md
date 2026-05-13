# internal/infra

基础设施：日志、GORM、外部 REST/WebSocket 客户端实现。**GORM `AutoMigrate`** 在此或上层统一触发；不使用 SQL migration 文件。

**Canonical**：`database.go`。
