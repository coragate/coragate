import { Button } from "@/components/ui/button";

export default function Home() {
  return (
    <main className="mx-auto flex min-h-svh max-w-xl flex-col justify-center gap-4 p-8">
      <h1 className="text-2xl font-semibold">CoraGate 控制面</h1>
      <p className="text-muted-foreground text-sm">
        T0 空壳。规则、沙盒、看板在 T7。本进程不转发{" "}
        <code className="font-mono">/v1/chat/completions</code>
        。规范：
        <a
          className="underline"
          href="https://github.com/coragate/coragate-docs/tree/main/docs/specs/gateway-mvp"
        >
          SPEC-gateway-mvp
        </a>
      </p>
      <Button asChild variant="outline">
        <a href="https://github.com/coragate/coragate-docs">打开文档仓</a>
      </Button>
    </main>
  );
}
