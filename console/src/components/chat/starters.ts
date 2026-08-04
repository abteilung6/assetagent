import type { ChatPageContext } from "@/api/types.gen";

export type ChatStarter = {
  id: string;
  label: string;
  prompt: string;
};

export type FollowUpChip = {
  id: string;
  label: string;
  prompt: string;
};

/** Last complete calendar month relative to `now` (local). */
export function lastCompleteCalendarMonth(now = new Date()): {
  yyyyMm: string;
  from: string;
  to: string;
  label: string;
} {
  const firstOfThisMonth = new Date(now.getFullYear(), now.getMonth(), 1);
  const lastMonthEnd = new Date(firstOfThisMonth.getTime() - 1);
  const yyyy = lastMonthEnd.getFullYear();
  const mm = String(lastMonthEnd.getMonth() + 1).padStart(2, "0");
  const lastDay = lastMonthEnd.getDate();
  const yyyyMm = `${yyyy}-${mm}`;
  return {
    yyyyMm,
    from: `${yyyyMm}-01`,
    to: `${yyyyMm}-${String(lastDay).padStart(2, "0")}`,
    label: lastMonthEnd.toLocaleDateString(undefined, {
      month: "long",
      year: "numeric",
    }),
  };
}

export function defaultStarters(now = new Date()): ChatStarter[] {
  const last = lastCompleteCalendarMonth(now);
  return [
    {
      id: "typical-month",
      label: "What’s my typical month?",
      prompt: "What’s my typical month?",
    },
    {
      id: "last-month-spend",
      label: `How much did I spend in ${last.label}?`,
      prompt: `How much did I spend in ${last.label}? Use booking dates ${last.from} to ${last.to}.`,
    },
    {
      id: "unusual",
      label: "What looks unusual recently?",
      prompt: "What looks unusual recently?",
    },
    {
      id: "needs-review",
      label: "What should I clear in Needs review?",
      prompt: "What should I clear in Needs review?",
    },
  ];
}

export type StarterPromotionInput = {
  needsReviewTotal: number;
  baselineStatus?: "draft" | "confirmed" | null;
};

/** Reorder starters based on household state (A5). */
export function promoteStarters(
  starters: ChatStarter[],
  input: StarterPromotionInput,
): ChatStarter[] {
  const byId = new Map(starters.map((s) => [s.id, s]));
  const ordered: ChatStarter[] = [];
  const used = new Set<string>();

  const push = (id: string) => {
    const starter = byId.get(id);
    if (starter && !used.has(id)) {
      ordered.push(starter);
      used.add(id);
    }
  };

  if (input.needsReviewTotal > 0) {
    push("needs-review");
  }
  if (input.baselineStatus === "draft") {
    push("typical-month");
  }

  for (const starter of starters) {
    push(starter.id);
  }
  return ordered;
}

/** Follow-ups after an assistant answer, keyed by last tool name (A5). */
export function followUpsForTool(toolName: string | undefined): FollowUpChip[] {
  switch (toolName) {
    case "get_cashflow_v2":
    case "get_month_cashflow":
      return [
        {
          id: "top-paid",
          label: "Who did I pay most?",
          prompt: "Who did I pay the most in that period?",
        },
        {
          id: "unusual-after-cashflow",
          label: "Anything unusual?",
          prompt: "Anything unusual in that period?",
        },
      ];
    case "get_baseline":
      return [
        {
          id: "explain-free",
          label: "Explain free cashflow",
          prompt: "Explain how my sustainable free cashflow is calculated.",
        },
        {
          id: "last-month-vs-baseline",
          label: "Compare last month",
          prompt: "How did last calendar month compare to my typical month?",
        },
      ];
    case "get_needs_review_summary":
    case "suggest_review_categories":
      return [
        {
          id: "category-help",
          label: "Suggest categories",
          prompt: "What category suggestions do I have in Needs review?",
        },
        {
          id: "unusual-after-review",
          label: "Anything unusual?",
          prompt: "What looks unusual recently?",
        },
      ];
    case "get_anomalies":
      return [
        {
          id: "one-offs",
          label: "One-off impact?",
          prompt: "How much did one-off expenses affect that period?",
        },
        {
          id: "categories-after-anomaly",
          label: "Top categories?",
          prompt: "What were the top spending categories in that period?",
        },
      ];
    case "get_forecast":
      return [
        {
          id: "runway",
          label: "Explain runway",
          prompt: "Explain my 90-day runway in plain language.",
        },
        {
          id: "baseline-after-forecast",
          label: "Typical month?",
          prompt: "What’s my typical month?",
        },
      ];
    default:
      return [
        {
          id: "typical-follow",
          label: "Typical month?",
          prompt: "What’s my typical month?",
        },
        {
          id: "review-follow",
          label: "Needs review?",
          prompt: "What should I clear in Needs review?",
        },
      ];
  }
}

export type ChatHandoff = {
  prompt: string;
  context?: ChatPageContext;
};

export function buildChatHandoff(handoff: ChatHandoff): {
  to: "/chat";
  search: { prompt: string; context?: string };
} {
  return {
    to: "/chat",
    search: {
      prompt: handoff.prompt,
      ...(handoff.context
        ? { context: JSON.stringify(handoff.context) }
        : {}),
    },
  };
}

export function parseChatContextParam(raw: unknown): ChatPageContext | undefined {
  if (typeof raw !== "string" || !raw.trim()) {
    return undefined;
  }
  try {
    const parsed = JSON.parse(raw) as ChatPageContext;
    if (!parsed || typeof parsed !== "object") {
      return undefined;
    }
    return parsed;
  } catch {
    return undefined;
  }
}
