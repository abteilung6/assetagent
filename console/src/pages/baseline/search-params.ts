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

/** Month detail pages no longer carry a parent tab. */
export type BaselineMonthSearchParams = Record<string, never>;

export function parseBaselineMonthSearchParams(
  _search: Record<string, unknown>,
): BaselineMonthSearchParams {
  return {};
}
