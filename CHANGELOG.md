# Changelog

User-visible changes follow [SemVer](https://semver.org/) ([SPEC-gateway-mvp](https://github.com/coragate/coragate-docs/tree/main/docs/specs/gateway-mvp) AC-13 / ADR-0011).

Breaking plugin-host APIs, rule snapshots, or config semantics → major; compatible additions → minor; fixes → patch. No cross-major auto-migrator in phase 1.

Untagged binaries default to `0.1.0-dev`. Override with `-ldflags "-X github.com/coragate/coragate/kernel.Version=x.y.z"`.

Chinese notes for historical T0–T10 work remain in git history; new changelog entries are English (ADR-0009).

## Unreleased

- Host plugin catalog `GET /v1/plugins`; compiler Factory Method map (unknown plugin rejected, not coerced to keyword); SSE copy Template Method; inspect JSON lists `matches[]`; audit queue drop counter (ADR-0016).
- Built-in `injection` detector (first jailbreak / instruction-override fixtures); default action is **block**. `hosts/client` and `hosts/cluster` compile the same plugin; this is not a Cloud-only guardrail (SPEC-prompt-injection).
- Control plane: pick `plugin=injection` (entity type `prompt_injection`); matching stays in the Go plugin.

- Control plane: English/zh-CN UI via cookie `coragate_locale` (no locale in the URL path) (SPEC-controlplane-i18n).
- Control plane: placeholder operator in the sidebar footer; language lives on Settings (`/?view=settings`).

- Breaking: detection plugins return all matches (`InspectResult.Matches`); the host merges every plugin instead of stopping at the first hit (SPEC-pii-entities).
- Built-in `pii` detector (email, CN phone, CN id card, bank card); default action is redact. Input enforce+redact rebuilds `messages[].content` JSON; prompt hash still uses the original body.
- Streaming output redact holds back up to 254 runes and rewrites complete `data:` lines so entity plaintext is not flushed (SPEC-pii-entities AC-4).
- `hosts/client` and `hosts/cluster` compile the same `pii` plugin; PII is not a Cloud-only DLP product.
- Control plane: pick plugin, entity type, and block/redact (Shark UI); sandbox still calls dataplane `POST /v1/inspect`.

- Apache License 2.0 (`LICENSE` / `NOTICE`) (ADR-0014).
- Public docs: English README as source, Chinese in `README.zh-CN.md` (ADR-0009).
- CONTRIBUTING / SECURITY; README states self-host is first-class (ADR-0010).
- T10: AGENTS.md and dataplane rule: no Next chat BFF, no funnel narrative.
- T9: Review ACs from design §4 (no Next chat BFF, default fail_open in docs, dual-host README, no SQLite tables in kernel).
- T8: Dataplane still proxies and enforces when the control plane is not running.
- T7: Control plane rules editor, sandbox via dataplane `/v1/inspect`, read-only hits; dataplane `GET/PUT /v1/rules` and `GET /v1/audit`.
- T6c: `--version` and `GET /health`; config / rule snapshots / audit envelopes carry `schema_version`.
- T6b: Same kernel, two host defaults (client `127.0.0.1`, cluster `0.0.0.0`).
- T6: Detection engine down defaults to fail-open (audit `engine_error`); explicit `fail_closed` rejects; observe hits do not block.
- T5: Rule JSON snapshot load/reload (file poll / `POST /v1/reload`); new requests use new rules.
- T4: Async audit queue + versioned envelope; default JSONL file adapter.
- T3: SSE output sliding window scan (plugin `InspectOutputWindow`).
- T2: Plugin host + built-in keyword/regex; enforce hits skip upstream; `POST /v1/inspect` sandbox.
- T1: OpenAI-compatible `POST /v1/chat/completions` SSE proxy.
- T0: Monorepo scaffold (kernel / hosts / plugins + control-plane shell).
