"use client";

import { useActionState } from "react";
import { Shield, FlaskConical, List } from "lucide-react";
import { inspectAction, saveRulesAction, type ActionState } from "@/app/actions";
import type { AuditItem, RuleSnapshot } from "@/lib/dataplane";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

const idle: ActionState = { ok: true, message: "" };

export function Console({
  rules,
  hits,
  dataplaneError,
}: {
  rules: RuleSnapshot;
  hits: AuditItem[];
  dataplaneError?: string;
}) {
  const [saveState, saveAction, saving] = useActionState(saveRulesAction, idle);
  const [inspectState, inspectFormAction, inspecting] = useActionState(
    inspectAction,
    idle
  );
  const snapshotJSON = JSON.stringify(
    {
      schema_version: rules.schema_version || 1,
      rules: rules.rules?.length
        ? rules.rules
        : [{ id: "demo-keyword", plugin: "keyword", pattern: "(?i)coragate-block-me" }],
    },
    null,
    2
  );

  return (
    <Tabs defaultValue="rules">
      <TabsList>
        <TabsTrigger value="rules">
          <Shield data-icon="inline-start" />
          规则
        </TabsTrigger>
        <TabsTrigger value="sandbox">
          <FlaskConical data-icon="inline-start" />
          沙盒
        </TabsTrigger>
        <TabsTrigger value="hits">
          <List data-icon="inline-start" />
          命中
        </TabsTrigger>
      </TabsList>

      {dataplaneError ? (
        <p className="mt-4 text-sm text-destructive">
          数据面不可达（{dataplaneError}）。请先启动 Go 数据面，并设置{" "}
          <code className="font-mono">DATAPLANE_BASE_URL</code>。
        </p>
      ) : null}

      <TabsContent value="rules" className="mt-4">
        <Card>
          <CardHeader>
            <CardTitle>规则快照</CardTitle>
            <CardDescription>
              写入数据面 JSON 快照（含 schema_version），由数据面加载；本页不实现匹配。
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form action={saveAction} className="flex flex-col gap-3">
              <Label htmlFor="snapshot">规则 JSON</Label>
              <Textarea
                id="snapshot"
                name="snapshot"
                defaultValue={snapshotJSON}
                className="min-h-48 font-mono text-sm"
              />
              <Button type="submit" disabled={saving}>
                {saving ? "保存中…" : "保存到数据面"}
              </Button>
              {saveState.message ? (
                <p className={saveState.ok ? "text-muted-foreground text-sm" : "text-destructive text-sm"}>
                  {saveState.message}
                </p>
              ) : null}
            </form>
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="sandbox" className="mt-4">
        <Card>
          <CardHeader>
            <CardTitle>沙盒</CardTitle>
            <CardDescription>
              文本交给数据面 <code className="font-mono">POST /v1/inspect</code>
              ，不打上游、不在 Next 里做正则。
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form action={inspectFormAction} className="flex flex-col gap-3">
              <Label htmlFor="text">粘贴文本</Label>
              <Textarea id="text" name="text" placeholder="please coragate-block-me" className="min-h-32" />
              <Button type="submit" disabled={inspecting}>
                {inspecting ? "检测中…" : "调用数据面检测"}
              </Button>
            </form>
            {inspectState.inspect ? (
              <div className="mt-4 flex items-center gap-2">
                {inspectState.inspect.engine_error ? (
                  <Badge variant="destructive">引擎错误</Badge>
                ) : inspectState.inspect.hit ? (
                  <Badge variant="destructive">命中 {inspectState.inspect.rule_id}</Badge>
                ) : (
                  <Badge variant="secondary">未命中</Badge>
                )}
                {inspectState.inspect.engine_error ? (
                  <span className="text-muted-foreground text-sm">
                    {inspectState.inspect.engine_error}
                  </span>
                ) : null}
              </div>
            ) : inspectState.message ? (
              <p className="text-destructive mt-3 text-sm">{inspectState.message}</p>
            ) : null}
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="hits" className="mt-4">
        <Card>
          <CardHeader>
            <CardTitle>命中列表</CardTitle>
            <CardDescription>
              只读展示数据面审计 envelope（经存储接口），不是 SQLite 直查。
            </CardDescription>
          </CardHeader>
          <CardContent>
            {hits.length === 0 ? (
              <p className="text-muted-foreground text-sm">暂无已 flush 的审计记录。</p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>时间</TableHead>
                    <TableHead>规则</TableHead>
                    <TableHead>结果</TableHead>
                    <TableHead>策略</TableHead>
                    <TableHead>哈希</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {hits.map((h, i) => (
                    <TableRow key={`${h.prompt_hash}-${h.time}-${i}`}>
                      <TableCell className="whitespace-nowrap">{h.time}</TableCell>
                      <TableCell>{h.rule_id}</TableCell>
                      <TableCell>{h.outcome ?? "—"}</TableCell>
                      <TableCell>{h.policy_mode}</TableCell>
                      <TableCell className="max-w-40 truncate font-mono text-xs">
                        {h.prompt_hash}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>
  );
}
