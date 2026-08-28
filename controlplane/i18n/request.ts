import { cookies, headers } from "next/headers";
import { getRequestConfig } from "next-intl/server";
import { LOCALE_COOKIE, resolveLocale } from "./locale";

export default getRequestConfig(async () => {
  const jar = await cookies();
  const hdrs = await headers();
  const locale = resolveLocale(
    jar.get(LOCALE_COOKIE)?.value,
    hdrs.get("accept-language")
  );
  return {
    locale,
    messages: (await import(`../messages/${locale}.json`)).default,
  };
});
