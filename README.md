# CoraGate

极低延迟、流式优先的 LLM 安全代理网关（数据面 + 控制面 Monorepo）。

**规范真源（不要在本仓另写冲突架构）：** [SPEC-gateway-mvp](https://github.com/coragate/coragate-docs/tree/main/docs/specs/gateway-mvp)

组织文档仓：https://github.com/coragate/coragate-docs

## 怎么指过来

数据面是舰队，不是全站漏斗。选靠近调用方的实例（本机进程、sidecar、集群副本都可以），而不是全公司共用一个 URL。

| 形态 | 宿主 | 默认监听 | 示例 `baseURL` |
|------|------|----------|----------------|
| C 端本机 | `hosts/client` | `127.0.0.1:8080` | `http://127.0.0.1:8080/v1` |
| 集群/服务 | `hosts/cluster` | `0.0.0.0:8080` | `http://<本集群数据面>:8080/v1` |

同一内核，两套默认监听；可用 `CORAGATE_LISTEN` 覆盖。客户端改 OpenAI 兼容 `baseURL` 指向**你选的那一个实例**即可。

## 进程

- **数据面（Go）**：热路径。`/v1/chat/completions` 只由 Go 写 SSE。面板没启动时数据面仍应能代理（T1 起）。
- **控制面（Next.js App Router + shadcn/ui）**：规则 / 沙盒 / 看板。`controlplane/` 禁止做聊天流 BFF。

当前数据面已代理 `POST /v1/chat/completions`：输入同步检测；输出 SSE 滑动窗口边读边扫；审计异步写入文件适配器（`data/audit.jsonl`，规范是 envelope JSON，不是 SQLite 表）。规则来自带 `schema_version` 的 JSON 快照（默认 `data/rules.json`），改文件后短轮询加载，也可 `POST /v1/reload`。`enforce` 输入命中不打上游；`observe` 命中仍转发并审计。检测引擎不可用时**默认 `fail_open`**（放行，审计打 `engine_error`）；只有显式 `CORAGATE_FAIL_MODE=fail_closed` 才拒绝。沙盒：`POST /v1/inspect`。控制面仍是空壳（T7）。

```bash
export CORAGATE_UPSTREAM_BASE_URL=https://api.openai.com
# 可选：CORAGATE_POLICY_MODE=enforce
# 可选：CORAGATE_FAIL_MODE=fail_open
# 可选：CORAGATE_RULES_PATH=data/rules.json
go run ./hosts/client/cmd/coragate
```

演示拦截：用户消息含 `coragate-block-me` 时返回 403，上游不会收到请求。沙盒：

```bash
curl -sS http://127.0.0.1:8080/v1/inspect \
  -H 'Content-Type: application/json' \
  -d '{"text":"please coragate-block-me"}'
```

改规则：编辑 `data/rules.json`（示例见 `examples/rules.json`），等最多约 1 秒或手动刷新：

```bash
curl -sS -X POST http://127.0.0.1:8080/v1/reload
```

需要配置上游（OpenAI 兼容 `baseURL`），例如官方 API 或本地 vLLM：

```bash
export CORAGATE_UPSTREAM_BASE_URL=https://api.openai.com
# 本机宿主
go run ./hosts/client/cmd/coragate

# 集群宿主
go run ./hosts/cluster/cmd/coragate

# 控制面空壳（无规则/沙盒，那些是 T7）
cd controlplane && pnpm dev
```

客户端把 OpenAI SDK 的 `baseURL` 指到上表对应实例，例如 `http://127.0.0.1:8080/v1`。面板未启动时数据面仍可代理。
