"use client";

import { useActionState } from "react";
import { CircleAlert } from "lucide-react";
import { useSearchParams } from "next/navigation";
import { useTranslations } from "next-intl";
import { inspectAction, saveRulesAction, type ActionState } from "@/app/actions";
import { AppSidebar } from "@/components/app-sidebar";
import { RulesEditor } from "@/components/rules-editor";
import { SettingsPanel } from "@/components/settings-panel";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Separator } from "@/components/ui/separator";
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import type { Locale } from "@/i18n/locale";
import type { AuditItem, PluginInfo, RuleSnapshot } from "@/lib/dataplane";
import { parseView, type ViewId } from "@/lib/view";

const idle: ActionState = { ok: true, message: "" };

export function Console({
  locale,
  view: initialView,
  rules,
  hits,
  plugins,
  dataplaneError,
}: {
  locale: Locale;
  view: ViewId;
  rules: RuleSnapshot;
  hits: AuditItem[];
  plugins: PluginInfo[];
  dataplaneError?: string;
}) {
  const searchParams = useSearchParams();
  const view = parseView(searchParams.get("view") ?? initialView);
  const tRules = useTranslations("rules");
  const tSandbox = useTranslations("sandbox");
  const tHits = useTranslations("hits");
  const tSettings = useTranslations("settings");
  const tDp = useTranslations("dataplane");
  const [saveState, saveAction, saving] = useActionState(saveRulesAction, idle);
  const [inspectState, inspectFormAction, inspecting] = useActionState(
    inspectAction,
    idle
  );

  const title =
    view === "sandbox"
      ? tSandbox("title")
      : view === "hits"
        ? tHits("title")
        : view === "settings"
          ? tSettings("title")
          : tRules("title");

  return (
    <SidebarProvider>
      <AppSidebar hitsCount={hits.length} view={view} />
      <SidebarInset>
        <header className="flex h-16 shrink-0 items-center gap-2 border-b px-4 transition-[height] duration-200 ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-12">
          <SidebarTrigger />
          <Separator className="h-4" orientation="vertical" />
          <h1 className="font-medium text-sm">{title}</h1>
        </header>
        <div className="flex flex-1 flex-col gap-6 overflow-auto p-6">
          {dataplaneError ? (
            <Alert variant="destructive">
              <CircleAlert aria-hidden />
              <AlertTitle>{tDp("title")}</AlertTitle>
              <AlertDescription>
                {tDp("unreachable", { error: dataplaneError })}
              </AlertDescription>
            </Alert>
          ) : null}

          {view === "rules" ? (
            <div className="flex max-w-2xl flex-col gap-4">
              <p className="text-muted-foreground text-sm">{tRules("description")}</p>
              <form action={saveAction}>
                <FieldGroup>
                  <RulesEditor
                    initial={rules.rules ?? []}
                    plugins={plugins}
                    schemaVersion={rules.schema_version || 1}
                  />
                  <Button type="submit" isLoading={saving}>
                    {tRules("save")}
                  </Button>
                  {saveState.message ? (
                    <p
                      className={
                        saveState.ok
                          ? "text-muted-foreground text-sm"
                          : "text-destructive text-sm"
                      }
                    >
                      {saveState.message}
                    </p>
                  ) : null}
                </FieldGroup>
              </form>
            </div>
          ) : null}

          {view === "sandbox" ? (
            <div className="flex max-w-2xl flex-col gap-4">
              <p className="text-muted-foreground text-sm">
                {tSandbox.rich("description", {
                  endpoint: (chunks) => (
                    <code className="font-mono">{chunks}</code>
                  ),
                })}
              </p>
              <form action={inspectFormAction}>
                <FieldGroup>
                  <Field>
                    <FieldLabel>{tSandbox("paste")}</FieldLabel>
                    <Textarea
                      className="min-h-32"
                      name="text"
                      placeholder="please coragate-block-me"
                    />
                  </Field>
                  <Button type="submit" isLoading={inspecting}>
                    {tSandbox("run")}
                  </Button>
                </FieldGroup>
              </form>
              {inspectState.inspect ? (
                <div className="flex flex-col gap-3">
                  <div className="flex items-center gap-2">
                    {inspectState.inspect.engine_error ? (
                      <Badge variant="destructive">{tSandbox("engineError")}</Badge>
                    ) : inspectState.inspect.hit ? (
                      <Badge variant="destructive">
                        {tSandbox("hit", { id: inspectState.inspect.rule_id ?? "" })}
                      </Badge>
                    ) : (
                      <Badge variant="secondary">{tSandbox("miss")}</Badge>
                    )}
                    {inspectState.inspect.engine_error ? (
                      <span className="text-muted-foreground text-sm">
                        {inspectState.inspect.engine_error}
                      </span>
                    ) : null}
                  </div>
                  {inspectState.inspect.matches &&
                  inspectState.inspect.matches.length > 0 ? (
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{tHits("rule")}</TableHead>
                          <TableHead>{tHits("entity")}</TableHead>
                          <TableHead>{tHits("action")}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {inspectState.inspect.matches.map((m, i) => (
                          <TableRow key={`${m.rule_id}-${i}`}>
                            <TableCell>{m.rule_id}</TableCell>
                            <TableCell>{m.entity_type ?? "—"}</TableCell>
                            <TableCell>{m.action ?? "—"}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  ) : null}
                </div>
              ) : inspectState.message ? (
                <p className="text-destructive text-sm">{inspectState.message}</p>
              ) : null}
            </div>
          ) : null}

          {view === "hits" ? (
            <div className="flex flex-col gap-4">
              <p className="max-w-2xl text-muted-foreground text-sm">
                {tHits("description")}
              </p>
              {hits.length === 0 ? (
                <p className="text-muted-foreground text-sm">{tHits("empty")}</p>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{tHits("time")}</TableHead>
                      <TableHead>{tHits("rule")}</TableHead>
                      <TableHead>{tHits("entity")}</TableHead>
                      <TableHead>{tHits("action")}</TableHead>
                      <TableHead>{tHits("outcome")}</TableHead>
                      <TableHead>{tHits("policy")}</TableHead>
                      <TableHead>{tHits("hash")}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {hits.map((h, i) => (
                      <TableRow key={`${h.prompt_hash}-${h.time}-${i}`}>
                        <TableCell>{h.time}</TableCell>
                        <TableCell>{h.rule_id}</TableCell>
                        <TableCell>{h.entity_type ?? "—"}</TableCell>
                        <TableCell>{h.rule_action ?? "—"}</TableCell>
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
            </div>
          ) : null}

          {view === "settings" ? <SettingsPanel locale={locale} /> : null}
        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}
