# Agent 指南

本仓库是 [SPEC-gateway-mvp](https://github.com/coragate/coragate-docs/tree/main/docs/specs/gateway-mvp) 的落点。架构真源在 **coragate-docs**，不要在本仓另写冲突故事。

## 硬约束

- `/v1/chat/completions` 只由 Go 数据面写 SSE。`controlplane/` 禁止 Route Handler / BFF 转发聊天流。
- 数据面是舰队：`hosts/client`（默认 `127.0.0.1`）与 `hosts/cluster`（默认 `0.0.0.0`）。禁止把「全站唯一入口」写成默认拓扑。
- 新行为先有 docs 仓 spec，再改代码。commit 引用 `SPEC-gateway-mvp#AC-n`。
- 注释与单测描述用中文。控制面组件用 shadcn CLI，图标用 Lucide。
