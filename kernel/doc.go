// Package kernel 是数据面微内核：监听抽象、OpenAI 兼容面、上游连接、SSE。
//
// 同一内核由 hosts/cluster 与 hosts/client 绑定不同默认监听。
// 热路径由插件做输入检测后再转发；沙盒走 POST /v1/inspect，不打上游。
// 规则与配置快照带 schema_version；GET /health 与 --version 可查二进制版本。
package kernel
