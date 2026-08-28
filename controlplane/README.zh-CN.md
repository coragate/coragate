# 控制面

Next.js App Router + shadcn/ui（图标 Lucide）。规则编辑、沙盒、只读命中列表。

英文源：[README.md](README.md)（ADR-0009）。

**禁止** Route Handler / BFF 转发 `/v1/chat/completions`（ADR-0002）。沙盒只调用数据面 `POST /v1/inspect`；规则走 `GET/PUT /v1/rules`；看板走 `GET /v1/audit`。

```bash
# 先起数据面
export CORAGATE_UPSTREAM_BASE_URL=https://api.openai.com
go run ./hosts/client/cmd/coragate

# 再起控制面（默认同机 8080）
cd controlplane
# DATAPLANE_BASE_URL=http://127.0.0.1:8080
pnpm dev
```

规范：[SPEC-gateway-mvp](https://github.com/coragate/coragate-docs/tree/main/docs/specs/gateway-mvp)
