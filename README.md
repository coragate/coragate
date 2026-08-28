# CoraGate

A low-latency, streaming-first LLM security proxy (dataplane + control plane monorepo).

Chinese translation: [README.zh-CN.md](README.zh-CN.md). English is the source (ADR-0009).

**Spec source of truth (do not fork a conflicting architecture here):** [SPEC-gateway-mvp](https://github.com/coragate/coragate-docs/tree/main/docs/specs/gateway-mvp)

Org docs: https://github.com/coragate/coragate-docs

## Self-host vs Cloud

**Self-host is first-class** ([ADR-0010](https://github.com/coragate/coragate-docs/blob/main/docs/adrs/0010-opensource-and-cloud-revenue.md)). This repository is the open-source core: dataplane kernel, built-in plugins, and the open control plane. You do not need a CoraGate Cloud account to proxy LLM traffic.

A hosted cloud product (multi-tenant, SLA, managed control plane) may exist later as a commercial option. It is not the only way to run CoraGate. The client default remains a **local** dataplane (`hosts/client`), not a mandatory cloud URL.

## Where to point clients

The dataplane is a **fleet**, not a site-wide funnel. Point at an instance near the caller (local process, sidecar, or cluster replica)—not one URL for the whole company.

| Surface | Host | Default listen | Example `baseURL` |
|---------|------|----------------|-------------------|
| Client / on-device | `hosts/client` | `127.0.0.1:8080` | `http://127.0.0.1:8080/v1` |
| Cluster / service | `hosts/cluster` | `0.0.0.0:8080` | `http://<this-cluster-dataplane>:8080/v1` |

Same kernel, two default binds. Override with `CORAGATE_LISTEN`. Point the OpenAI-compatible `baseURL` at **the instance you chose**.

## Processes

- **Dataplane (Go):** hot path. Only Go writes SSE for `/v1/chat/completions`. The gateway still proxies if the panel is down.
- **Control plane (Next.js App Router + shadcn/ui):** rules / sandbox / hits. `controlplane/` must not BFF the chat stream.

The dataplane already proxies `POST /v1/chat/completions`: sync inspect on input; sliding-window scan on output SSE; async audit via a file adapter (`data/audit.jsonl`—envelope JSON, not a SQLite schema). Rules are a versioned JSON snapshot (default `data/rules.json`), reloaded on a short poll, `POST /v1/reload`, or `PUT /v1/rules`. Config snapshots also carry `schema_version` (see `examples/config.json`). `enforce` blocks on input hits before upstream; `observe` still forwards and audits. If the detection engine is down, the default is **`fail_open`** (forward and tag `engine_error`); only `CORAGATE_FAIL_MODE=fail_closed` rejects. Sandbox: `POST /v1/inspect`. Version: `coragate --version` or `GET /health`. The panel edits rules, calls the dataplane sandbox, and lists hits; it does not forward chat.

```bash
export CORAGATE_UPSTREAM_BASE_URL=https://api.openai.com
# optional: CORAGATE_POLICY_MODE=enforce
# optional: CORAGATE_FAIL_MODE=fail_open
# optional: CORAGATE_RULES_PATH=data/rules.json
go run ./hosts/client/cmd/coragate
```

Version:

```bash
go run ./hosts/client/cmd/coragate --version
curl -sS http://127.0.0.1:8080/health
```

Demo block: a user message containing `coragate-block-me` returns 403 and the upstream is not called. Sandbox:

```bash
curl -sS http://127.0.0.1:8080/v1/inspect \
  -H 'Content-Type: application/json' \
  -d '{"text":"please coragate-block-me"}'
```

Change rules: edit `data/rules.json` (see `examples/rules.json`), wait ~1s, or reload:

```bash
curl -sS -X POST http://127.0.0.1:8080/v1/reload
```

Configure an OpenAI-compatible upstream (official API or local vLLM):

```bash
export CORAGATE_UPSTREAM_BASE_URL=https://api.openai.com
# client host
go run ./hosts/client/cmd/coragate

# cluster host
go run ./hosts/cluster/cmd/coragate

# control plane: rules / sandbox / hits (DATAPLANE_BASE_URL defaults to http://127.0.0.1:8080)
cd controlplane && pnpm dev
```

Point the OpenAI SDK `baseURL` at the instance in the table, e.g. `http://127.0.0.1:8080/v1`. The dataplane still proxies if the panel is not running.

## License

[Apache License 2.0](LICENSE). See [NOTICE](NOTICE).
