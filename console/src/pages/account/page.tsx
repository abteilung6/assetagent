import type React from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";

import {
  getMeQueryKey,
  patchMeMutation,
} from "@/api/@tanstack/react-query.gen";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { useMe } from "@/hooks/use-me";
import { changeAppLocale } from "@/i18n";
import { isLocale, persistLocale, type Locale } from "@/i18n/locale";
import { initialsFromName, primaryDisplayName } from "@/lib/user-display";

const LOCALE_OPTIONS: { value: Locale; labelKey: "account.languageDe" | "account.languageEn" }[] =
  [
    { value: "de", labelKey: "account.languageDe" },
    { value: "en", labelKey: "account.languageEn" },
  ];

const AccountPage: React.FC = () => {
  const { t, i18n } = useTranslation();
  const me = useMe();
  const queryClient = useQueryClient();
  const patch = useMutation({
    ...patchMeMutation(),
    onSuccess: async (data) => {
      queryClient.setQueryData(getMeQueryKey(), data);
      const locale = data.user.preferred_locale;
      if (isLocale(locale)) {
        persistLocale(locale);
        await changeAppLocale(locale);
      }
    },
  });

  const user = me.data?.user;
  if (!user) {
    return null;
  }

  const name = primaryDisplayName({
    givenName: user.given_name,
    displayName: user.display_name,
    email: user.email,
  });
  const fullName = user.display_name.trim() || name;
  const initials = initialsFromName(fullName);
  const activeLocale: Locale = isLocale(user.preferred_locale)
    ? user.preferred_locale
    : isLocale(i18n.language)
      ? i18n.language
      : "de";

  const selectLocale = (locale: Locale) => {
    if (locale === activeLocale || patch.isPending) {
      return;
    }
    const previous = activeLocale;
    persistLocale(locale);
    void changeAppLocale(locale);
    patch.mutate(
      { body: { preferred_locale: locale } },
      {
        onError: () => {
          persistLocale(previous);
          void changeAppLocale(previous);
        },
      },
    );
  };

  return (
    <div className="mx-auto flex w-full max-w-lg flex-col gap-8">
      <section className="space-y-4">
        <h2 className="text-sm font-medium text-muted-foreground">
          {t("account.profileHeading")}
        </h2>
        <div className="flex items-center gap-4">
          <Avatar size="lg" className="rounded-xl">
            {user.picture_url ? (
              <AvatarImage src={user.picture_url} alt={fullName} />
            ) : null}
            <AvatarFallback className="rounded-xl text-base">
              {initials}
            </AvatarFallback>
          </Avatar>
          <div className="min-w-0 space-y-1">
            <p className="truncate text-base font-medium">{fullName}</p>
            {user.email ? (
              <p className="truncate text-sm text-muted-foreground">
                {user.email}
              </p>
            ) : null}
            <p className="text-xs text-muted-foreground">
              {t("account.signedInWithGoogle")}
            </p>
          </div>
        </div>
      </section>

      <section className="space-y-3">
        <div className="space-y-1">
          <h2 className="text-sm font-medium text-muted-foreground">
            {t("account.languageHeading")}
          </h2>
          <p className="text-sm text-muted-foreground">
            {t("account.languageDescription")}
          </p>
        </div>
        <div
          className="flex gap-2"
          role="group"
          aria-label={t("account.languageHeading")}
        >
          {LOCALE_OPTIONS.map((option) => {
            const selected = option.value === activeLocale;
            return (
              <Button
                key={option.value}
                type="button"
                variant={selected ? "default" : "outline"}
                size="sm"
                disabled={patch.isPending}
                aria-pressed={selected}
                onClick={() => {
                  selectLocale(option.value);
                }}
              >
                {t(option.labelKey)}
              </Button>
            );
          })}
        </div>
        {patch.isPending ? (
          <p className="text-xs text-muted-foreground">{t("account.saving")}</p>
        ) : null}
        {patch.isError ? (
          <p className="text-xs text-destructive">{t("account.saveFailed")}</p>
        ) : null}
      </section>
    </div>
  );
};

export default AccountPage;
