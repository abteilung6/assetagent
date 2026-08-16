import type React from "react";

import { useLocaleSync } from "@/hooks/use-locale-sync";

/** Applies authenticated preferred_locale to i18n + localStorage. */
export const LocaleSync: React.FC = () => {
  useLocaleSync();
  return null;
};
