# 贡献指南

英文源：[CONTRIBUTING.md](CONTRIBUTING.md)（[ADR-0009](https://github.com/coragate/coragate-docs/blob/main/docs/adrs/0009-language-oss-english-primary.md)）。

本仓库是**开源产品仓**。架构与验收在 [coragate-docs](https://github.com/coragate/coragate-docs)，不要在这里另写一套冲突设计。

## 规范先行

1. 新行为 / 改验收 → 先在 `coragate-docs` 立规范（`/specify` → `/plan` → `/tasks`）。
2. 在本仓实现。Commit 必须引用 `SPEC-<slug>#AC-n`。
3. 豁免：拼写、格式、依赖升级、无行为变更的重构。

## 硬约束

- `/v1/chat/completions` 的 SSE 只由 Go 数据面写。Next 控制面禁止做聊天流 BFF。
- 数据面是舰队：文档必须同时给出 `hosts/client`（`127.0.0.1`）与 `hosts/cluster`（`0.0.0.0`）。
- 控制面组件用 `shadcn add @shark/<component>`（Shark UI）；图标用 Lucide。
- 持久化走存储适配器；禁止把 SQLite DDL 焊进 `kernel/`。
- 现链能承接则沿链路支持；承接不住则重做设计，禁止打补丁（ADR-0016）。动手前先按 GoF 模式选型。

## Pull request

使用 `.github/PULL_REQUEST_TEMPLATE.md`。贴 spec 链接（或声明豁免）。

## 许可证

[Apache License 2.0](LICENSE)（[ADR-0014](https://github.com/coragate/coragate-docs/blob/main/docs/adrs/0014-apache-2-license.md)）。对本仓的贡献默认按同一许可证授权，除非另有书面协议。
