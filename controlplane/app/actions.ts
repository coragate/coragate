"use server";

import { revalidatePath } from "next/cache";
import { cookies } from "next/headers";
import { getTranslations } from "next-intl/server";
import { inspectText, putRules, type InspectResult } from "@/lib/dataplane";
import { isLocale, LOCALE_COOKIE } from "@/i18n/locale";

export type ActionState = {
  ok: boolean;
  message: string;
  inspect?: InspectResult;
};

export async function setLocaleAction(formData: FormData) {
  const next = String(formData.get("locale") ?? "");
  if (!isLocale(next)) {
    return;
  }
  const jar = await cookies();
  jar.set(LOCALE_COOKIE, next, {
    path: "/",
    maxAge: 60 * 60 * 24 * 365,
    sameSite: "lax",
  });
  // 同一路径刷新；不 redirect 到 /en
  revalidatePath("/", "layout");
}

export async function saveRulesAction(
  _prev: ActionState,
  formData: FormData
): Promise<ActionState> {
  const t = await getTranslations("actions");
  const raw = String(formData.get("snapshot") ?? "");
  let snap: {
    schema_version: number;
    rules: {
      id: string;
      plugin?: string;
      pattern?: string;
      type?: string;
      action?: string;
    }[];
  };
  try {
    snap = JSON.parse(raw) as typeof snap;
  } catch {
    return { ok: false, message: t("parseError") };
  }
  try {
    await putRules({
      schema_version: snap.schema_version || 1,
      rules: (snap.rules ?? []).map((r) => ({
        id: r.id,
        plugin: r.plugin,
        pattern: r.pattern,
        type: r.type,
        action: r.action,
      })),
    });
    revalidatePath("/");
    return { ok: true, message: t("saved") };
  } catch (e) {
    return {
      ok: false,
      message: e instanceof Error ? e.message : t("saveFailed"),
    };
  }
}

export async function inspectAction(
  _prev: ActionState,
  formData: FormData
): Promise<ActionState> {
  const t = await getTranslations("actions");
  const text = String(formData.get("text") ?? "");
  if (!text.trim()) {
    return { ok: false, message: t("needText") };
  }
  try {
    const inspect = await inspectText(text);
    return {
      ok: true,
      message: inspect.engine_error ? t("engineDown") : t("inspected"),
      inspect,
    };
  } catch (e) {
    return {
      ok: false,
      message: e instanceof Error ? e.message : t("inspectFailed"),
    };
  }
}
