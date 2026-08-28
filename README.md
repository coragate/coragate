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

当前 T0 仅脚手架：Go 宿主打印占位后退出；面板是空壳。代理从 T1 开始。

```bash
# 集群宿主 stub
go run ./hosts/cluster/cmd/coragate

# 本机宿主 stub
go run ./hosts/client/cmd/coragate

# 控制面空壳（无规则/沙盒，那些是 T7）
cd controlplane && pnpm dev
```
