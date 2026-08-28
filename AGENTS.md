# Agent guide

This repo implements [SPEC-gateway-mvp](https://github.com/coragate/coragate-docs/tree/main/docs/specs/gateway-mvp). Architecture source of truth is **coragate-docs**—do not write a conflicting story here.

Public docs are **English-primary** (ADR-0009). Chinese lives in `README.zh-CN.md`. Production godoc and Go test names/messages are English.

## Hard constraints

- Only the Go dataplane writes SSE for `/v1/chat/completions`. `controlplane/` must not Route-Handler / BFF the chat stream.
- Dataplane is a fleet: `hosts/client` (default `127.0.0.1`) and `hosts/cluster` (default `0.0.0.0`). Do not document a site-wide single entry as the only topology.
- New behavior needs a spec in the docs repo first. Commits cite `SPEC-gateway-mvp#AC-n`.
- Install control-plane components with the shadcn CLI; icons are Lucide.
