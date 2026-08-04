export const INSIGHTS_MONTH_TABS = ["overview", "activity"] as const;

export type InsightsMonthTab = (typeof INSIGHTS_MONTH_TABS)[number];

export type InsightsMonthSearchParams = {
  tab?: InsightsMonthTab;
};

export function parseInsightsMonthSearchParams(
  search: Record<string, unknown>,
): InsightsMonthSearchParams {
  const tab = parseInsightsMonthTab(search.tab);
  return tab ? { tab } : {};
}

export function parseInsightsMonthTab(
  value: unknown,
): InsightsMonthTab | undefined {
  if (value === "overview" || value === "activity") {
    return value;
  }
  return undefined;
}
