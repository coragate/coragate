# Changelog

本仓库按 [SemVer](https://semver.org/) 记录用户可见变更（[SPEC-gateway-mvp](https://github.com/coragate/coragate-docs/tree/main/docs/specs/gateway-mvp) AC-13 / ADR-0011）。

破坏插件宿主 API、规则快照或配置语义 → major；兼容新增 → minor；修复 → patch。第一期不提供跨 major 自动迁移器。

当前未打 tag 的二进制默认版本为 `0.1.0-dev`，可用 `-ldflags "-X github.com/coragate/coragate/kernel.Version=x.y.z"` 覆盖。

## Unreleased

- T6c：`--version` 与 `GET /health` 可查版本；配置 / 规则快照 / 审计 envelope 均带 `schema_version`。
- T6b：同一内核两套宿主默认监听（本机 `127.0.0.1`、集群 `0.0.0.0`）。
- T6：检测引擎不可用默认 fail-open（审计 `engine_error`）；显式 `fail_closed` 才拒绝；observe 命中不阻断。
- T5：规则 JSON 快照加载与 reload（文件短轮询 / `POST /v1/reload`）；新请求用新规则。
- T4：审计异步队列 + versioned envelope；默认文件 JSONL 适配器。
- T3：输出 SSE 滑动窗口边读边扫（插件 `InspectOutputWindow`）。
- T2：插件宿主 + 内置关键字/正则；enforce 命中不打上游；`POST /v1/inspect` 沙盒。
- T1：OpenAI 兼容 `POST /v1/chat/completions` SSE 透传代理。
- T0：Monorepo 脚手架（内核/宿主/插件占位 + 控制面空壳）。
