# CoraGate

极低延迟、流式优先的 LLM 安全代理网关（数据面 + 控制面 Monorepo）。

**规范真源（不要在本仓另写冲突架构）：** [SPEC-gateway-mvp](https://github.com/coragate/coragate-docs/tree/main/docs/specs/gateway-mvp)

组织文档仓：https://github.com/coragate/coragate-docs

## 怎么指过来

数据面是舰队，不是全站漏斗。选靠近调用方的实例，而不是全公司共用一个 URL。

| 形态 | 宿主 | 示例 `baseURL` |
|------|------|----------------|
| C 端本机 | `hosts/client` | `http://127.0.0.1:8080/v1` |
| 集群/服务 | `hosts/cluster` | `http://<本集群数据面>:8080/v1` |

同一内核，两套默认监听。客户端改 OpenAI 兼容 `baseURL` 即可。

## 进程

- **数据面（Go）**：热路径。`/v1/chat/completions` 只由 Go 写 SSE。面板没启动时数据面仍应能代理（T1 起）。
- **控制面（Next.js App Router + shadcn/ui）**：规则 / 沙盒 / 看板。`controlplane/` 禁止做聊天流 BFF。

当前数据面已代理 `POST /v1/chat/completions`：输入同步检测；输出 SSE 滑动窗口边读边扫；审计异步写入文件适配器（`data/audit.jsonl`，规范是 envelope JSON，不是 SQLite 表）。`enforce` 输入命中不打上游。沙盒：`POST /v1/inspect`。控制面仍是空壳（T7）。

```bash
export CORAGATE_UPSTREAM_BASE_URL=https://api.openai.com
# 可选：CORAGATE_POLICY_MODE=enforce
# 可选：CORAGATE_DETECT_PATTERN='(?i)coragate-block-me'
go run ./hosts/client/cmd/coragate
```

演示拦截：用户消息含 `coragate-block-me` 时返回 403，上游不会收到请求。沙盒：

```bash
curl -sS http://127.0.0.1:8080/v1/inspect \
  -H 'Content-Type: application/json' \
  -d '{"text":"please coragate-block-me"}'
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
