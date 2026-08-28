// Package kernel is the dataplane microkernel: listen abstraction, OpenAI-compatible
// surface, upstream client, and SSE.
//
// The same kernel is bound by hosts/cluster and hosts/client with different default
// listen addresses. The hot path runs inspect plugins, then forwards; the sandbox is
// POST /v1/inspect and does not call upstream. Rule and config snapshots carry
// schema_version. GET /health and --version report the binary version.
package kernel
