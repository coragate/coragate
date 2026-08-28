import { Console } from "@/components/console";
import { fetchAudit, fetchRules, type AuditItem, type RuleSnapshot } from "@/lib/dataplane";

export const dynamic = "force-dynamic";

export default async function Home() {
  let rules: RuleSnapshot = { schema_version: 1, rules: [] };
  let hits: AuditItem[] = [];
  let dataplaneError: string | undefined;
  try {
    rules = await fetchRules();
    hits = await fetchAudit(50);
  } catch (e) {
    dataplaneError = e instanceof Error ? e.message : "未知错误";
  }

  return (
    <main className="mx-auto flex min-h-svh max-w-4xl flex-col gap-6 p-8">
      <header className="flex flex-col gap-1">
        <h1 className="text-2xl font-semibold">CoraGate 控制面</h1>
        <p className="text-muted-foreground text-sm">
          规则 / 沙盒 / 看板。本进程不转发{" "}
          <code className="font-mono">/v1/chat/completions</code>。
        </p>
      </header>
      <Console rules={rules} hits={hits} dataplaneError={dataplaneError} />
    </main>
  );
}
