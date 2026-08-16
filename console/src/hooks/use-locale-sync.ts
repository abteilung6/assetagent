import { useEffect } from "react";

import { changeAppLocale } from "@/i18n";
import { isLocale, persistLocale } from "@/i18n/locale";
import { useMe } from "@/hooks/use-me";

/** Keep i18n + localStorage aligned with the authenticated preferred_locale. */
export function useLocaleSync(): void {
  const me = useMe();
  const preferred = me.data?.user.preferred_locale;

  useEffect(() => {
    if (!isLocale(preferred)) {
      return;
    }
    persistLocale(preferred);
    void changeAppLocale(preferred);
  }, [preferred]);
}
