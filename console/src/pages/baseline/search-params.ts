/** Legacy tab query values still accepted for redirects. */
export const LEGACY_BASELINE_TABS = ["composition", "over-time"] as const;

export type LegacyBaselineTab = (typeof LEGACY_BASELINE_TABS)[number];

export type BaselineSearchParams = {
  /** @deprecated Prefer /insights/months; kept for redirects. */
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
