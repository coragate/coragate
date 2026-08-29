"use client";

import { useMemo, useState } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  NativeSelect,
  NativeSelectOption,
} from "@/components/ui/native-select";
import type { PluginInfo, RuleRow } from "@/lib/dataplane";
import {
  catalogOrFallback,
  defaultActionFor,
  emptyKeyword,
  fromRow,
  isKnownPlugin,
  pluginInfo,
  toRow,
  type DraftRule,
} from "@/lib/rules-draft";

export function RulesEditor({
  initial,
  schemaVersion,
  plugins,
}: {
  initial: RuleRow[];
  schemaVersion: number;
  plugins?: PluginInfo[];
}) {
  const t = useTranslations("rules");
  const catalog = catalogOrFallback(plugins);
  const [rows, setRows] = useState<DraftRule[]>(() =>
    initial.length
      ? initial.map((r) => fromRow(r, catalog))
      : [emptyKeyword(catalog)]
  );

  const snapshot = useMemo(
    () =>
      JSON.stringify(
        {
          schema_version: schemaVersion || 1,
          rules: rows.map((r) => toRow(r, catalog)),
        },
        null,
        2
      ),
    [rows, schemaVersion, catalog]
  );

  function update(i: number, patch: Partial<DraftRule>) {
    setRows((prev) => prev.map((r, j) => (j === i ? { ...r, ...patch } : r)));
  }

  const hasPii = catalog.some((p) => p.id === "pii");
  const hasInjection = catalog.some((p) => p.id === "injection");

  return (
    <FieldGroup>
      {/* 提交给数据面的快照；匹配仍在 Go 插件里做 */}
      <input type="hidden" name="snapshot" value={snapshot} />
      {rows.map((row, i) => {
        const info = pluginInfo(row.plugin, catalog);
        const known = isKnownPlugin(row.plugin, catalog);
        const types = info?.entity_types ?? [];
        return (
          <div
            key={i}
            className="flex flex-col gap-3 rounded-xl border p-3"
          >
            <Field>
              <FieldLabel>{t("ruleId")}</FieldLabel>
              <Input
                type="text"
                value={row.id}
                onChange={(e) => update(i, { id: e.target.value })}
              />
            </Field>
            {!known ? (
              <p className="text-destructive text-sm">{t("unknownPlugin")}</p>
            ) : null}
            <div className="flex flex-wrap gap-3">
              <Field>
                <FieldLabel>{t("plugin")}</FieldLabel>
                <NativeSelect
                  value={row.plugin}
                  onChange={(e) => {
                    const plugin = e.target.value;
                    if (!isKnownPlugin(plugin, catalog)) {
                      return;
                    }
                    const next = pluginInfo(plugin, catalog);
                    update(i, {
                      plugin,
                      action: defaultActionFor(plugin, catalog),
                      type: next?.entity_types?.[0] ?? "",
                    });
                  }}
                >
                  {!known ? (
                    <NativeSelectOption value={row.plugin}>
                      {row.plugin}
                    </NativeSelectOption>
                  ) : null}
                  {catalog.map((p) => (
                    <NativeSelectOption key={p.id} value={p.id}>
                      {p.id}
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
              </Field>
              <Field>
                <FieldLabel>{t("action")}</FieldLabel>
                <NativeSelect
                  value={row.action}
                  onChange={(e) =>
                    update(i, {
                      action: e.target.value === "redact" ? "redact" : "block",
                    })
                  }
                >
                  <NativeSelectOption value="block">block</NativeSelectOption>
                  <NativeSelectOption value="redact">redact</NativeSelectOption>
                </NativeSelect>
              </Field>
              {types.length > 0 ? (
                <Field>
                  <FieldLabel>{t("entityType")}</FieldLabel>
                  <NativeSelect
                    value={row.type || types[0]}
                    disabled={types.length === 1}
                    onChange={(e) => update(i, { type: e.target.value })}
                  >
                    {types.map((typ) => (
                      <NativeSelectOption key={typ} value={typ}>
                        {typ}
                      </NativeSelectOption>
                    ))}
                  </NativeSelect>
                </Field>
              ) : info?.needs_pattern ? (
                <Field className="min-w-48 flex-1">
                  <FieldLabel>{t("pattern")}</FieldLabel>
                  <Input
                    type="text"
                    value={row.pattern}
                    onChange={(e) => update(i, { pattern: e.target.value })}
                    className="font-mono"
                  />
                </Field>
              ) : null}
            </div>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setRows((prev) => prev.filter((_, j) => j !== i))}
              disabled={rows.length <= 1}
            >
              {t("remove")}
            </Button>
          </div>
        );
      })}
      <div className="flex flex-wrap gap-2">
        {hasPii ? (
          <Button
            type="button"
            variant="outline"
            onClick={() =>
              setRows((prev) => [
                ...prev,
                {
                  id: `pii-email-${prev.length + 1}`,
                  plugin: "pii",
                  type: "email",
                  action: defaultActionFor("pii", catalog),
                  pattern: "",
                },
              ])
            }
          >
            {t("addPii")}
          </Button>
        ) : null}
        {hasInjection ? (
          <Button
            type="button"
            variant="outline"
            onClick={() =>
              setRows((prev) => [
                ...prev,
                {
                  id: `injection-prompt-${prev.length + 1}`,
                  plugin: "injection",
                  type: "prompt_injection",
                  action: defaultActionFor("injection", catalog),
                  pattern: "",
                },
              ])
            }
          >
            {t("addInjection")}
          </Button>
        ) : null}
      </div>
    </FieldGroup>
  );
}
