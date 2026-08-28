# Control plane

Next.js App Router + Shark UI (Ark registry, Lucide icons). Rules editor, sandbox, read-only hit list.

Chinese: [README.zh-CN.md](README.zh-CN.md).

**Do not** BFF `/v1/chat/completions` from a Route Handler (ADR-0002). Sandbox calls dataplane `POST /v1/inspect` only; rules use `GET/PUT /v1/rules`; the board uses `GET /v1/audit`.

```bash
# dataplane first
export CORAGATE_UPSTREAM_BASE_URL=https://api.openai.com
go run ./hosts/client/cmd/coragate

# then this app (defaults to dataplane on :8080)
cd controlplane
# DATAPLANE_BASE_URL=http://127.0.0.1:8080
pnpm dev
```

Spec: [SPEC-gateway-mvp](https://github.com/coragate/coragate-docs/tree/main/docs/specs/gateway-mvp)
