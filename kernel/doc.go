// Package kernel 是数据面微内核：监听抽象、OpenAI 兼容面、上游连接、SSE、插件宿主、配置快照、审计队列。
//
// 热路径（/v1/chat/completions）实现见 SPEC-gateway-mvp T1 起；本包 T0 仅占位。
// 同一内核由 hosts/cluster 与 hosts/client 绑定不同默认监听。
package kernel
