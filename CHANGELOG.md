# Changelog

本文件位置满足 [SPEC-gateway-mvp](https://github.com/coragate/coragate-docs/tree/main/docs/specs/gateway-mvp) AC-13 占位。T6c 起按 SemVer 记录用户可见变更。

## Unreleased

- T3：输出 SSE 滑动窗口边读边扫（插件 `InspectOutputWindow`）。
- T2：插件宿主 + 内置关键字/正则；enforce 命中不打上游；`POST /v1/inspect` 沙盒。
- T1：OpenAI 兼容 `POST /v1/chat/completions` SSE 透传代理。
- T0：Monorepo 脚手架（内核/宿主/插件占位 + 控制面空壳）。
