# Agent guide

This repo implements specs from **coragate-docs** (`SPEC-<slug>`: gateway-mvp, pii-entities, controlplane-i18n, prompt-injection, and later slugs). Do not write a conflicting architecture story here.

Public docs are **English-primary** (ADR-0009). Chinese lives in `README.zh-CN.md`. Production godoc and Go test names/messages are English.

## Hard constraints

- Only the Go dataplane writes SSE for `/v1/chat/completions`. `controlplane/` must not Route-Handler / BFF the chat stream.
- Dataplane is a fleet: `hosts/client` (default `127.0.0.1`) and `hosts/cluster` (default `0.0.0.0`). Do not document a site-wide single entry as the only topology.
- New behavior needs a spec in the docs repo first. Commits cite `SPEC-<slug>#AC-n`.
- If the current seam can carry the change, extend it. If it cannot, redesign—do not patch the hot path with special cases (ADR-0016). Name the GoF pattern first (see the ADR appendix).
- Control plane locale is a cookie (`coragate_locale`), never a URL prefix. Do not add `app/[locale]`.
- Install control-plane components with `shadcn add @shark/<component>` (Shark UI / Ark registry, ADR-0015). Icons are Lucide. Read the project `shark-ui` skill before writing UI.
