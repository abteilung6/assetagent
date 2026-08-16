import {
  createRootRoute,
  createRoute,
  createRouter,
  redirect,
  type RouterHistory,
} from "@tanstack/react-router";

import { AppLayout } from "@/app/layout";
import BaselinePage from "@/pages/baseline/page";
import BaselineExpensesPage from "@/pages/baseline/expenses/page";
import BaselineIncomePage from "@/pages/baseline/income/page";
import BaselineTrackingPage from "@/pages/baseline/tracking/page";
import ChatPage from "@/pages/chat/page";
import ImportsPage from "@/pages/imports/page";
import InsightsCategoriesPage from "@/pages/insights/categories/page";
import InsightsMonthPage from "@/pages/insights/month/page";
import InsightsMonthsPage from "@/pages/insights/months/page";
import AccountPage from "@/pages/account/page";
import LoginPage from "@/pages/login/page";
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

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/login",
  component: LoginPage,
  staticData: {
    titleKey: "auth.loginTitle",
  },
});


const accountRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/account",
  component: AccountPage,
  staticData: {
    titleKey: "account.title",
  },
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
    titleKey: "nav.chat",
  },
  validateSearch: (search) =>
    parseChatSearchParams(search as Record<string, unknown>),
});

const baselineRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/baseline",
  component: BaselinePage,
  staticData: {
    titleKey: "nav.cashflow",
  },
  validateSearch: (search) =>
    parseBaselineSearchParams(search as Record<string, unknown>),
  beforeLoad: ({ search }) => {
    if (search.tab === "over-time") {
      throw redirect({ to: "/insights/months" });
    }
  },
});

const baselineTrackingRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/baseline/tracking",
  component: BaselineTrackingPage,
  staticData: {
    titleKey: "nav.tracking",
  },
});

const legacyBaselinePerformanceRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/baseline/performance",
  beforeLoad: () => {
    throw redirect({ to: "/baseline/tracking" });
  },
});

const baselineIncomeRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/baseline/income",
  component: BaselineIncomePage,
  staticData: {
    titleKey: "nav.income",
  },
});

const baselineExpensesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/baseline/expenses",
  component: BaselineExpensesPage,
  staticData: {
    titleKey: "nav.expenses",
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
    titleKey: "nav.months",
  },
});

const insightsCategoriesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/insights/categories",
  component: InsightsCategoriesPage,
  staticData: {
    titleKey: "nav.categories",
  },
});

const insightsMonthRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/insights/months/$yyyyMm",
  component: InsightsMonthPage,
  staticData: {
    titleKey: "nav.months",
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
    titleKey: "nav.imports",
  },
});

const reviewRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/review",
  component: ReviewPage,
  staticData: {
    titleKey: "nav.needsReview",
  },
  validateSearch: (search) =>
    parseReviewSearchParams(search as Record<string, unknown>),
});

const reviewsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/reviews",
  component: ReviewsPage,
  staticData: {
    titleKey: "nav.reviews",
  },
});

const reviewDetailRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/reviews/$id",
  component: ReviewDetailPage,
  staticData: {
    titleKey: "nav.reviewDetail",
  },
});

const planRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/plan",
  component: PlanPage,
  staticData: {
    titleKey: "nav.plan",
  },
});

const transactionsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/transactions",
  component: TransactionsPage,
  staticData: {
    titleKey: "nav.transactions",
  },
  validateSearch: (search) =>
    parseTransactionSearchParams(search as Record<string, unknown>),
});

export {
  baselineExpensesRoute,
  baselineIncomeRoute,
  baselineTrackingRoute,
  baselineRoute,
  chatRoute,
  importsRoute,
  insightsCategoriesRoute,
  insightsMonthRoute,
  insightsMonthsRoute,
  planRoute,
  reviewDetailRoute,
  reviewRoute,
  reviewsRoute,
  transactionsRoute,
};

const routeTree = rootRoute.addChildren([
  loginRoute,
    accountRoute,
  indexRoute,
  chatRoute,
  baselineRoute,
  baselineIncomeRoute,
  baselineExpensesRoute,
  baselineTrackingRoute,
  legacyBaselinePerformanceRoute,
  insightsRoute,
  insightsMonthsRoute,
  insightsCategoriesRoute,
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
    titleKey?: string;
  }
}
