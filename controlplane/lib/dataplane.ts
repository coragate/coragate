/**
 * 控制面只访问数据面的规则 / 沙盒 / 审计接口。
 * 禁止拼接或请求 /v1/chat/completions（ADR-0002）。
 */
const chatCompletionsPath = "/v1/chat/completions";

export function dataplaneBaseURL(): string {
  const base = process.env.DATAPLANE_BASE_URL ?? "http://127.0.0.1:8080";
  return base.replace(/\/$/, "");
}

export function dataplaneURL(path: string): string {
  if (path.includes("chat/completions") || path === chatCompletionsPath) {
    throw new Error("控制面不得转发聊天流");
  }
  if (!path.startsWith("/")) {
    throw new Error("数据面路径必须以 / 开头");
  }
  return `${dataplaneBaseURL()}${path}`;
}

export type RuleRow = {
  id: string;
  plugin?: string;
  pattern: string;
};

export type RuleSnapshot = {
  schema_version: number;
  rules: RuleRow[];
};

export type InspectResult = {
  hit: boolean;
  rule_id?: string;
  engine_error?: string;
};

export type AuditItem = {
  schema_version: number;
  time: string;
  duration_ms: number;
  rule_id: string;
  prompt_hash: string;
  policy_mode: string;
  outcome?: string;
  engine_error?: string;
};

async function dataplaneFetch(path: string, init?: RequestInit): Promise<Response> {
  const res = await fetch(dataplaneURL(path), {
    ...init,
    cache: "no-store",
  });
  return res;
}

export async function fetchRules(): Promise<RuleSnapshot> {
  const res = await dataplaneFetch("/v1/rules");
  if (!res.ok) {
    throw new Error(`读取规则失败: ${res.status}`);
  }
  return (await res.json()) as RuleSnapshot;
}

export async function putRules(snap: RuleSnapshot): Promise<RuleSnapshot> {
  const res = await dataplaneFetch("/v1/rules", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(snap),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`保存规则失败: ${text}`);
  }
  return (await res.json()) as RuleSnapshot;
}

export async function inspectText(text: string): Promise<InspectResult> {
  const res = await dataplaneFetch("/v1/inspect", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ text }),
  });
  if (!res.ok) {
    const raw = await res.text();
    throw new Error(`沙盒检测失败: ${raw}`);
  }
  return (await res.json()) as InspectResult;
}

export async function fetchAudit(limit = 50): Promise<AuditItem[]> {
  const res = await dataplaneFetch(`/v1/audit?limit=${limit}`);
  if (!res.ok) {
    throw new Error(`读取审计失败: ${res.status}`);
  }
  const payload = (await res.json()) as { items?: AuditItem[] };
  return payload.items ?? [];
}
