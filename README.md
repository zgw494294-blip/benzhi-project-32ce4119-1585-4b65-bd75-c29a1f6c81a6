# 湿地修复苗木投放核验工作台

本项目是面向苗圃交接员、湿地生态技术员和项目复核负责人的浏览器工作台。它把苗木投放前的批次建档、来源与运输证据、驯化方案、连续观察、适生性判断、整改复验、独立复核、冻结清单和签名凭据串成一条可追溯流程，避免证据不足或健康风险未消除的苗木进入目标湿地区块。

服务由 Go 直接提供响应式 HTML、CSS、JavaScript 和同源 JSON API，不需要 Node 构建链。业务数据保存在本地 SQLite，启用 WAL、乐观版本控制和命令幂等约束；审计事件与业务状态在同一事务中保存。

## 构建与运行

标准构建：

```text
go build ./cmd/server
```

默认启动在高位回环地址 `127.0.0.1:19081`：

```text
go run ./cmd/server
```

可以显式指定回环地址与数据库路径：

```text
go run ./cmd/server -addr=127.0.0.1:19181 -db=data/wetland-release.db
```

也可以通过 `PORT` 提供端口号，服务会绑定 `127.0.0.1:<PORT>`。为保护工作台，`-addr` 拒绝非回环地址及低于 `1024` 的端口。启动后访问 `http://127.0.0.1:19081/`。

## 自检与测试

自检会在真实回环监听上启动 HTTP 服务，经同源 API 完成批次建档、证据登记、驯化方案、观察、独立批准和凭据验证，然后主动关闭：

```text
go run ./cmd/server -selfcheck -addr=127.0.0.1:19081
```

运行全部测试：

```text
go test ./...
```

每个写请求都需要 `idempotencyKey` 和当前 `expectedVersion`。浏览器工作台会自动携带这两个字段；API 调用方遇到 `VERSION_CONFLICT` 时应重新获取批次，再由用户确认后重试。

工作台还提供跨批次整改队列 `GET /api/remediation-queue`、物种连续观察趋势 `GET /api/batches/{id}/trends/{speciesCode}`、联合复验 `POST /api/batches/{id}/issues/joint-remediate` 和批准前冻结清单预览 `GET /api/batches/{id}/review-manifest`。批准请求需携带预览返回的 `confirmedManifestDigest`。
