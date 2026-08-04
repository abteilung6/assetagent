export type ChatSearchParams = {
  prompt?: string;
  context?: string;
};

export function parseChatSearchParams(
  search: Record<string, unknown>,
): ChatSearchParams {
  const prompt =
    typeof search.prompt === "string" && search.prompt.trim()
      ? search.prompt
      : undefined;
  const context =
    typeof search.context === "string" && search.context.trim()
      ? search.context
      : undefined;
  return {
    ...(prompt ? { prompt } : {}),
    ...(context ? { context } : {}),
  };
}
