// Package kernel 是数据面微内核：监听抽象、OpenAI 兼容面、上游连接、SSE。
//
// 同一内核由 hosts/cluster 与 hosts/client 绑定不同默认监听。
// 检测插件、审计、fail-open 见后续 SPEC-gateway-mvp 任务；T1 只做透传代理。
package kernel
