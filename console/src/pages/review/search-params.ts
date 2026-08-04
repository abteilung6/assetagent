export const REVIEW_TABS = ["categories", "recurring", "transfers"] as const;

export type ReviewTab = (typeof REVIEW_TABS)[number];

export type ReviewSearchParams = {
  tab?: ReviewTab;
};

export function parseReviewSearchParams(
  search: Record<string, unknown>,
): ReviewSearchParams {
  const tab = parseReviewTab(search.tab);
  return tab ? { tab } : {};
}

export function parseReviewTab(value: unknown): ReviewTab | undefined {
  if (
    value === "categories" ||
    value === "recurring" ||
    value === "transfers"
  ) {
    return value;
  }
  return undefined;
}
