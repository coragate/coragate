# Contributing

English is the source ([ADR-0009](https://github.com/coragate/coragate-docs/blob/main/docs/adrs/0009-language-oss-english-primary.md)). Chinese translation: [CONTRIBUTING.zh-CN.md](CONTRIBUTING.zh-CN.md).

This repo is the **open-source product**. Architecture and acceptance criteria live in [coragate-docs](https://github.com/coragate/coragate-docs)—do not fork a conflicting design here.

## Spec first

1. New behavior / changed acceptance criteria → spec in `coragate-docs` (`/specify` → `/plan` → `/tasks`).
2. Implement here. Commits must cite `SPEC-<slug>#AC-n`.
3. Exempt: spelling, formatting, dependency bumps, refactors with no behavior change.

## Hard constraints

- Only the Go dataplane writes SSE for `/v1/chat/completions`. The Next control plane must not BFF the chat stream.
- The dataplane is a fleet: document both `hosts/client` (`127.0.0.1`) and `hosts/cluster` (`0.0.0.0`).
- Install UI components with the shadcn CLI; icons are Lucide.
- Persistence goes through storage adapters; do not weld SQLite DDL into `kernel/`.

## Pull requests

Use `.github/PULL_REQUEST_TEMPLATE.md`. Link the spec (or state an exemption).

## License

The license is still unchosen ([ADR-0003](https://github.com/coragate/coragate-docs/blob/main/docs/adrs/0003-mvp-scope-cuts.md)). It will be published in the same milestone as making this repository public.
