import { useMatches } from "@tanstack/react-router";
import type React from "react";
import { useTranslation } from "react-i18next";

export const PageTitle: React.FC = () => {
  const { t } = useTranslation();
  const matches = useMatches();
  const titleKey = [...matches]
    .reverse()
    .find((match) => match.staticData?.titleKey)?.staticData?.titleKey;

  if (!titleKey) {
    return null;
  }

  return <h1 className="text-base font-semibold">{t(titleKey)}</h1>;
};
