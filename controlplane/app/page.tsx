import { Suspense } from "react";
import { getLocale, getTranslations } from "next-intl/server";
import { Console } from "@/components/console";
import type { Locale } from "@/i18n/locale";
import {
  FALLBACK_PLUGINS,
  fetchAudit,
  fetchPlugins,
  fetchRules,
  type AuditItem,
  type PluginInfo,
  type RuleSnapshot,
} from "@/lib/dataplane";
import { parseView } from "@/lib/view";

export const dynamic = "force-dynamic";

export default async function Home({
  searchParams,
}: {
  searchParams: Promise<{ view?: string }>;
}) {
  const params = await searchParams;
  const view = parseView(params.view);
  const locale = (await getLocale()) as Locale;
  let rules: RuleSnapshot = { schema_version: 1, rules: [] };
  let hits: AuditItem[] = [];
  let plugins: PluginInfo[] = FALLBACK_PLUGINS;
  let dataplaneError: string | undefined;
  try {
    rules = await fetchRules();
    hits = await fetchAudit(50);
    plugins = await fetchPlugins();
  } catch (e) {
    dataplaneError = e instanceof Error ? e.message : (await getTranslations("actions"))("unknownError");
    plugins = await fetchPlugins();
  }

  return (
    <Suspense>
      <Console
        dataplaneError={dataplaneError}
        hits={hits}
        locale={locale}
        plugins={plugins}
        rules={rules}
        view={view}
      />
    </Suspense>
  );
}
