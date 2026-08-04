/** Legacy tab query values still accepted for redirects. */
export const LEGACY_BASELINE_TABS = ["composition", "over-time"] as const;

export type LegacyBaselineTab = (typeof LEGACY_BASELINE_TABS)[number];

export type BaselineSearchParams = {
  /** @deprecated Prefer /baseline vs /baseline/history; kept for redirects. */
  tab?: LegacyBaselineTab;
};

export function parseBaselineSearchParams(
  search: Record<string, unknown>,
): BaselineSearchParams {
  const tab = parseLegacyBaselineTab(search.tab);
  return tab ? { tab } : {};
}

export function parseLegacyBaselineTab(
  value: unknown,
): LegacyBaselineTab | undefined {
  if (value === "composition" || value === "over-time") {
    return value;
  }
  return undefined;
}

export const BASELINE_MONTH_TABS = ["overview", "activity"] as const;

export type BaselineMonthTab = (typeof BASELINE_MONTH_TABS)[number];

export type BaselineMonthSearchParams = {
  tab?: BaselineMonthTab;
};

export function parseBaselineMonthSearchParams(
  search: Record<string, unknown>,
): BaselineMonthSearchParams {
  const tab = parseBaselineMonthTab(search.tab);
  return tab ? { tab } : {};
}

export function parseBaselineMonthTab(
  value: unknown,
): BaselineMonthTab | undefined {
  if (value === "overview" || value === "activity") {
    return value;
  }
  return undefined;
}
