"use client";

import { useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  NativeSelect,
  NativeSelectOption,
} from "@/components/ui/native-select";
import type { RuleRow } from "@/lib/dataplane";

const PII_TYPES = ["email", "phone_cn", "id_card_cn", "bank_card"] as const;

type DraftRule = {
  id: string;
  plugin: "keyword" | "pii";
  type: string;
  action: "block" | "redact";
  pattern: string;
};

function fromRow(r: RuleRow): DraftRule {
  const plugin = r.plugin === "pii" ? "pii" : "keyword";
  return {
    id: r.id,
    plugin,
    type: r.type ?? "email",
    action: r.action === "redact" || r.action === "block"
      ? r.action
      : plugin === "pii"
        ? "redact"
        : "block",
    pattern: r.pattern ?? "",
  };
}

function toRow(d: DraftRule): RuleRow {
  const row: RuleRow = {
    id: d.id.trim(),
    plugin: d.plugin,
    action: d.action,
  };
  if (d.plugin === "pii") {
    row.type = d.type;
  } else {
    row.pattern = d.pattern;
  }
  return row;
}

function emptyKeyword(): DraftRule {
  return {
    id: "demo-keyword",
    plugin: "keyword",
    type: "email",
    action: "block",
    pattern: "(?i)coragate-block-me",
  };
}

export function RulesEditor({
  initial,
  schemaVersion,
}: {
  initial: RuleRow[];
  schemaVersion: number;
}) {
  const [rows, setRows] = useState<DraftRule[]>(() =>
    initial.length ? initial.map(fromRow) : [emptyKeyword()]
  );

  const snapshot = useMemo(
    () =>
      JSON.stringify(
        {
          schema_version: schemaVersion || 1,
          rules: rows.map(toRow),
        },
        null,
        2
      ),
    [rows, schemaVersion]
  );

  function update(i: number, patch: Partial<DraftRule>) {
    setRows((prev) => prev.map((r, j) => (j === i ? { ...r, ...patch } : r)));
  }

  return (
    <FieldGroup>
      {/* 提交给数据面的快照；匹配仍在 Go 插件里做 */}
      <input type="hidden" name="snapshot" value={snapshot} />
      {rows.map((row, i) => (
        <div
          key={i}
          className="flex flex-col gap-3 rounded-xl border p-3"
        >
          <Field>
            <FieldLabel>规则 ID</FieldLabel>
            <Input
              type="text"
              value={row.id}
              onChange={(e) => update(i, { id: e.target.value })}
            />
          </Field>
          <div className="flex flex-wrap gap-3">
            <Field>
              <FieldLabel>插件</FieldLabel>
              <NativeSelect
                value={row.plugin}
                onChange={(e) => {
                  const plugin = e.target.value === "pii" ? "pii" : "keyword";
                  update(i, {
                    plugin,
                    action: plugin === "pii" ? "redact" : "block",
                  });
                }}
              >
                <NativeSelectOption value="keyword">keyword</NativeSelectOption>
                <NativeSelectOption value="pii">pii</NativeSelectOption>
              </NativeSelect>
            </Field>
            <Field>
              <FieldLabel>动作</FieldLabel>
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
            {row.plugin === "pii" ? (
              <Field>
                <FieldLabel>实体类型</FieldLabel>
                <NativeSelect
                  value={row.type}
                  onChange={(e) => update(i, { type: e.target.value })}
                >
                  {PII_TYPES.map((t) => (
                    <NativeSelectOption key={t} value={t}>
                      {t}
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
              </Field>
            ) : (
              <Field className="min-w-48 flex-1">
                <FieldLabel>模式（正则）</FieldLabel>
                <Input
                  type="text"
                  value={row.pattern}
                  onChange={(e) => update(i, { pattern: e.target.value })}
                  className="font-mono"
                />
              </Field>
            )}
          </div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => setRows((prev) => prev.filter((_, j) => j !== i))}
            disabled={rows.length <= 1}
          >
            删除此条
          </Button>
        </div>
      ))}
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
              action: "redact",
              pattern: "",
            },
          ])
        }
      >
        添加 PII 规则
      </Button>
    </FieldGroup>
  );
}
