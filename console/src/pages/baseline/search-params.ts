export const BASELINE_TABS = ["composition", "over-time"] as const;

export type BaselineTab = (typeof BASELINE_TABS)[number];

export type BaselineSearchParams = {
  tab: BaselineTab;
};

export const defaultBaselineSearchParams: BaselineSearchParams = {
  tab: "composition",
};

export function parseBaselineSearchParams(
  search: Record<string, unknown>,
): BaselineSearchParams {
  return {
    tab: parseBaselineTab(search.tab),
  };
}

export function parseBaselineTab(value: unknown): BaselineTab {
  if (value === "over-time" || value === "composition") {
    return value;
  }
  return "composition";
}
