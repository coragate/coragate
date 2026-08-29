import type { PluginInfo, RuleRow } from "./dataplane";
import { FALLBACK_PLUGINS } from "./dataplane";

export type DraftRule = {
  id: string;
  plugin: string;
  type: string;
  action: "block" | "redact";
  pattern: string;
};

export function catalogOrFallback(plugins?: PluginInfo[]): PluginInfo[] {
  return plugins && plugins.length > 0 ? plugins : FALLBACK_PLUGINS;
}

/** 未知 plugin 返回 null，禁止静默当成 keyword。 */
export function parsePlugin(
  v: string | undefined,
  catalog: PluginInfo[]
): string | null {
  if (!v) {
    return null;
  }
  return catalog.some((p) => p.id === v) ? v : null;
}

export function resolvePlugin(
  v: string | undefined,
  catalog: PluginInfo[]
): string {
  const known = parsePlugin(v, catalog);
  if (known) {
    return known;
  }
  if (v) {
    return v;
  }
  return catalog[0]?.id ?? "keyword";
}

export function isKnownPlugin(id: string, catalog: PluginInfo[]): boolean {
  return catalog.some((p) => p.id === id);
}

export function pluginInfo(
  id: string,
  catalog: PluginInfo[]
): PluginInfo | undefined {
  return catalog.find((p) => p.id === id);
}

export function defaultActionFor(
  pluginId: string,
  catalog: PluginInfo[]
): "block" | "redact" {
  return pluginInfo(pluginId, catalog)?.default_action === "redact"
    ? "redact"
    : "block";
}

export function fromRow(r: RuleRow, catalog: PluginInfo[]): DraftRule {
  const plugin = resolvePlugin(r.plugin, catalog);
  const info = pluginInfo(plugin, catalog);
  const types = info?.entity_types ?? [];
  const action: "block" | "redact" =
    r.action === "redact" || r.action === "block"
      ? r.action
      : defaultActionFor(plugin, catalog);
  return {
    id: r.id,
    plugin,
    type: r.type ?? types[0] ?? "",
    action,
    pattern: r.pattern ?? "",
  };
}

export function toRow(d: DraftRule, catalog: PluginInfo[]): RuleRow {
  const info = pluginInfo(d.plugin, catalog);
  const row: RuleRow = {
    id: d.id.trim(),
    plugin: d.plugin,
    action: d.action,
  };
  if (info?.needs_pattern) {
    row.pattern = d.pattern;
  } else if (info?.entity_types && info.entity_types.length > 0) {
    row.type = d.type || info.entity_types[0];
  }
  return row;
}

export function emptyKeyword(catalog: PluginInfo[]): DraftRule {
  const plugin = parsePlugin("keyword", catalog) ?? catalog[0]?.id ?? "keyword";
  return {
    id: "demo-keyword",
    plugin,
    type: "",
    action: defaultActionFor(plugin, catalog),
    pattern: "(?i)coragate-block-me",
  };
}
