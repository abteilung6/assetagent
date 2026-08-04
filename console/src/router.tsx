import {
  createRootRoute,
  createRoute,
  createRouter,
  redirect,
  type RouterHistory,
} from "@tanstack/react-router";

import { AppLayout } from "@/app/layout";
import BaselinePage from "@/pages/baseline/page";
import BaselineIncomePage from "@/pages/baseline/income/page";
import BaselinePerformancePage from "@/pages/baseline/performance/page";
import ChatPage from "@/pages/chat/page";
import ImportsPage from "@/pages/imports/page";
import InsightsMonthPage from "@/pages/insights/month/page";
import InsightsMonthsPage from "@/pages/insights/months/page";
import PlanPage from "@/pages/plan/page";
import ReviewPage from "@/pages/review/page";
import ReviewDetailPage from "@/pages/reviews/detail";
import ReviewsPage from "@/pages/reviews/page";
import TransactionsPage from "@/pages/transactions/page";
import { parseBaselineSearchParams } from "@/pages/baseline/search-params";
import { parseInsightsMonthSearchParams } from "@/pages/insights/search-params";
import { parseChatSearchParams } from "@/pages/chat/search-params";
import { parseReviewSearchParams } from "@/pages/review/search-params";
import { parseTransactionSearchParams } from "@/pages/transactions/search-params";

const rootRoute = createRootRoute({
  component: AppLayout,
});

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  beforeLoad: () => {
    throw redirect({ to: "/chat" });
  },
});

const chatRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/chat",
  component: ChatPage,
  staticData: {
    title: "Chat",
  },
  validateSearch: (search) =>
    parseChatSearchParams(search as Record<string, unknown>),
});

const baselineRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/baseline",
  component: BaselinePage,
  staticData: {
    title: "Cashflow",
  },
  validateSearch: (search) =>
    parseBaselineSearchParams(search as Record<string, unknown>),
  beforeLoad: ({ search }) => {
    if (search.tab === "over-time") {
      throw redirect({ to: "/insights/months" });
    }
  },
});

const baselinePerformanceRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/baseline/performance",
  component: BaselinePerformancePage,
  staticData: {
    title: "Performance",
  },
});

const baselineIncomeRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/baseline/income",
  component: BaselineIncomePage,
  staticData: {
    title: "Income",
  },
});

const insightsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/insights",
  beforeLoad: () => {
    throw redirect({ to: "/insights/months" });
  },
});

const insightsMonthsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/insights/months",
  component: InsightsMonthsPage,
  staticData: {
    title: "Months",
  },
});

const insightsMonthRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/insights/months/$yyyyMm",
  component: InsightsMonthPage,
  staticData: {
    title: "Months",
  },
  validateSearch: (search) =>
    parseInsightsMonthSearchParams(search as Record<string, unknown>),
});

/** Legacy Baseline IA paths → Insights. */
const legacyBaselineHistoryRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/baseline/history",
  beforeLoad: () => {
    throw redirect({ to: "/insights/months" });
  },
});

const legacyBaselineMonthRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/baseline/months/$yyyyMm",
  beforeLoad: ({ params, search }) => {
    throw redirect({
      to: "/insights/months/$yyyyMm",
      params: { yyyyMm: params.yyyyMm },
      search,
    });
  },
  validateSearch: (search) =>
    parseInsightsMonthSearchParams(search as Record<string, unknown>),
});

const importsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/imports",
  component: ImportsPage,
  staticData: {
    title: "Import",
  },
});

const reviewRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/review",
  component: ReviewPage,
  staticData: {
    title: "Needs review",
  },
  validateSearch: (search) =>
    parseReviewSearchParams(search as Record<string, unknown>),
});

const reviewsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/reviews",
  component: ReviewsPage,
  staticData: {
    title: "Money Reviews",
  },
});

const reviewDetailRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/reviews/$id",
  component: ReviewDetailPage,
  staticData: {
    title: "Money Review",
  },
});

const planRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/plan",
  component: PlanPage,
  staticData: {
    title: "Plan",
  },
});

const transactionsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/transactions",
  component: TransactionsPage,
  staticData: {
    title: "Transactions",
  },
  validateSearch: (search) =>
    parseTransactionSearchParams(search as Record<string, unknown>),
});

export {
  baselineIncomeRoute,
  baselinePerformanceRoute,
  baselineRoute,
  chatRoute,
  importsRoute,
  insightsMonthRoute,
  insightsMonthsRoute,
  planRoute,
  reviewDetailRoute,
  reviewRoute,
  reviewsRoute,
  transactionsRoute,
};

const routeTree = rootRoute.addChildren([
  indexRoute,
  chatRoute,
  baselineRoute,
  baselineIncomeRoute,
  baselinePerformanceRoute,
  insightsRoute,
  insightsMonthsRoute,
  insightsMonthRoute,
  legacyBaselineHistoryRoute,
  legacyBaselineMonthRoute,
  planRoute,
  transactionsRoute,
  importsRoute,
  reviewRoute,
  reviewsRoute,
  reviewDetailRoute,
]);

export function createAppRouter(history?: RouterHistory) {
  return createRouter({
    routeTree,
    history,
  });
}

export const router = createAppRouter();

declare module "@tanstack/react-router" {
  interface Register {
    router: ReturnType<typeof createAppRouter>;
  }

  interface StaticDataRouteOption {
    title?: string;
  }
}
