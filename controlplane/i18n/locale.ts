/** 面板 locale：cookie 为真源，不进 URL 路径。 */

export const LOCALES = ["en", "zh-CN"] as const;

export type Locale = (typeof LOCALES)[number];

export const DEFAULT_LOCALE: Locale = "en";

/** 稳定 cookie 名（SPEC-controlplane-i18n）。 */
export const LOCALE_COOKIE = "coragate_locale";

export function isLocale(value: string | undefined | null): value is Locale {
  return value === "en" || value === "zh-CN";
}

/**
 * 从 Accept-Language 协商 en / zh-CN。
 * zh* → zh-CN；en* → en；按 q 值从高到低。
 */
export function negotiateAcceptLanguage(
  header: string | undefined | null
): Locale | undefined {
  if (!header?.trim()) {
    return undefined;
  }
  const parts = header.split(",").map((raw) => {
    const [tagPart, ...params] = raw.trim().split(";");
    const tag = tagPart.trim().toLowerCase();
    let q = 1;
    for (const p of params) {
      const [k, v] = p.split("=").map((s) => s.trim());
      if (k === "q" && v) {
        const n = Number(v);
        if (!Number.isNaN(n)) {
          q = n;
        }
      }
    }
    return { tag, q };
  });
  parts.sort((a, b) => b.q - a.q);
  for (const { tag, q } of parts) {
    if (!tag || q <= 0) {
      continue;
    }
    if (tag === "zh-cn" || tag.startsWith("zh-") || tag === "zh") {
      return "zh-CN";
    }
    if (tag === "en" || tag.startsWith("en-")) {
      return "en";
    }
  }
  return undefined;
}

/** cookie 优先；未知 cookie 忽略；再 Accept-Language；否则 en。 */
export function resolveLocale(
  cookie: string | undefined | null,
  acceptLanguage: string | undefined | null
): Locale {
  if (isLocale(cookie)) {
    return cookie;
  }
  return negotiateAcceptLanguage(acceptLanguage) ?? DEFAULT_LOCALE;
}
