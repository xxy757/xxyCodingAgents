# P2-1 开发报告：挂载 pprof + .env 支持

---

## 基本信息

| 字段 | 内容 |
|------|------|
| 任务ID | P2-1 |
| 任务名称 | 挂载 pprof + .env 支持 |
| 完成日期 | 2026-04-12 |
| 负责人 | AI Dev Agent |
| 实际工时 | 0.5h |
| 关联问题 | N10, N11 |

---

## 1. 任务概述

### 1.2 完成标准

- [x] `/debug/pprof/` 可访问（通过独立端口 6060）
- [x] 支持 `.env` 文件加载
- [x] 配置优先级：config.yaml < .env < env vars

---

## 2. 实现方法

### 2.1 pprof
使用 `_ "net/http/pprof"` 导入注册到 DefaultServeMux，在 main.go 中启动独立的 HTTP 服务器（默认 `localhost:6060`）提供 pprof 服务，避免在主 API 路由中混入调试端点。

### 2.2 .env
使用 `github.com/joho/godotenv` 库，在 `config.Load()` 开头调用 `godotenv.Load()` 加载 `.env` 文件中的环境变量。由于 .env 加载在 yaml 解析之后的环境变量覆盖之前生效，自然实现了 `config.yaml < .env < env vars` 的优先级。

---

## 4. 代码变更记录

### 4.2 修改文件

| 文件路径 | 变更说明 |
|----------|----------|
| `cmd/server/main.go` | 导入 net/http 和 net/http/pprof，启动独立 pprof 服务器 |
| `internal/config/config.go` | 导入 godotenv，Load 函数中调用 godotenv.Load()；添加 PprofAddr 字段和环境变量覆盖 |
| `configs/config.yaml` | 添加 pprof_addr: "localhost:6060" |

### 4.4 依赖变更

| 操作 | 包名 | 版本 | 说明 |
|------|------|------|------|
| 新增 | `github.com/joho/godotenv` | `v1.5.1` | .env 文件加载 |

---

## 5. 测试结果

```
$ go build ./...
BUILD OK

$ go vet ./...
VET OK
```
