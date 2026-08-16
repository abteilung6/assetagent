import i18n from "i18next";
import { initReactI18next } from "react-i18next";

import de from "@/i18n/locales/de.json";
import en from "@/i18n/locales/en.json";
import { DEFAULT_LOCALE, resolveInitialLocale, type Locale } from "@/i18n/locale";

void i18n.use(initReactI18next).init({
  resources: {
    de: { translation: de },
    en: { translation: en },
  },
  lng: resolveInitialLocale(),
  fallbackLng: DEFAULT_LOCALE,
  interpolation: { escapeValue: false },
  returnNull: false,
});

export async function changeAppLocale(locale: Locale): Promise<void> {
  if (i18n.language === locale) {
    return;
  }
  await i18n.changeLanguage(locale);
}

export default i18n;
