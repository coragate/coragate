# Agent guide

This repo implements [SPEC-gateway-mvp](https://github.com/coragate/coragate-docs/tree/main/docs/specs/gateway-mvp), [SPEC-pii-entities](https://github.com/coragate/coragate-docs/tree/main/docs/specs/pii-entities), and [SPEC-controlplane-i18n](https://github.com/coragate/coragate-docs/tree/main/docs/specs/controlplane-i18n). Architecture source of truth is **coragate-docs**—do not write a conflicting story here.

Public docs are **English-primary** (ADR-0009). Chinese lives in `README.zh-CN.md`. Production godoc and Go test names/messages are English.

## Hard constraints

- Only the Go dataplane writes SSE for `/v1/chat/completions`. `controlplane/` must not Route-Handler / BFF the chat stream.
- Dataplane is a fleet: `hosts/client` (default `127.0.0.1`) and `hosts/cluster` (default `0.0.0.0`). Do not document a site-wide single entry as the only topology.
- New behavior needs a spec in the docs repo first. Commits cite `SPEC-gateway-mvp#AC-n`, `SPEC-pii-entities#AC-n`, or `SPEC-controlplane-i18n#AC-n`.
- Control plane locale is a cookie (`coragate_locale`), never a URL prefix. Do not add `app/[locale]`.
- Install control-plane components with `shadcn add @shark/<component>` (Shark UI / Ark registry, ADR-0015). Icons are Lucide. Read the project `shark-ui` skill before writing UI.
