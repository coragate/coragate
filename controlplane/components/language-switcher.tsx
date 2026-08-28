"use client";

import { useTransition } from "react";
import { useTranslations } from "next-intl";
import { setLocaleAction } from "@/app/actions";
import { Field, FieldLabel } from "@/components/ui/field";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { isLocale, LOCALES, type Locale } from "@/i18n/locale";

export function LanguageSwitcher({ locale }: { locale: Locale }) {
  const t = useTranslations("locale");
  const [pending, start] = useTransition();

  return (
    <Field>
      <FieldLabel className="sr-only">{t("label")}</FieldLabel>
      <ToggleGroup
        className="w-full max-w-xs"
        disabled={pending}
        multiple={false}
        onValueChange={(details) => {
          const next = details.value[0];
          if (!isLocale(next) || next === locale) {
            return;
          }
          start(async () => {
            const fd = new FormData();
            fd.set("locale", next);
            await setLocaleAction(fd);
            window.location.reload();
          });
        }}
        size="sm"
        value={[locale]}
        variant="outline"
      >
        {LOCALES.map((id) => (
          <ToggleGroupItem className="flex-1" key={id} value={id}>
            {id === "zh-CN" ? t("zhCN") : t("en")}
          </ToggleGroupItem>
        ))}
      </ToggleGroup>
    </Field>
  );
}
