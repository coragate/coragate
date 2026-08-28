"use client";

import { useTranslations } from "next-intl";
import { LanguageSwitcher } from "@/components/language-switcher";
import {
  Card,
  CardContent,
  CardHeader,
} from "@/components/ui/card";
import type { Locale } from "@/i18n/locale";

export function SettingsPanel({ locale }: { locale: Locale }) {
  const t = useTranslations("settings");
  const tUser = useTranslations("user");

  return (
    <div className="flex max-w-2xl flex-col gap-6">
      <p className="text-muted-foreground text-sm">{t("description")}</p>
      <Card>
        <CardHeader description={t("languageHint")} title={t("language")} />
        <CardContent>
          <LanguageSwitcher locale={locale} />
        </CardContent>
      </Card>
      <Card>
        <CardHeader description={tUser("placeholder")} title={tUser("account")} />
        <CardContent className="flex flex-col gap-1">
          <p className="text-sm">{tUser("name")}</p>
          <p className="text-muted-foreground text-sm">{tUser("email")}</p>
        </CardContent>
      </Card>
    </div>
  );
}
