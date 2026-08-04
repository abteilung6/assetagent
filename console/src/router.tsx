import {
  createRootRoute,
  createRoute,
  createRouter,
  redirect,
  type RouterHistory,
} from "@tanstack/react-router";

import { AppLayout } from "@/app/layout";
import BaselinePage from "@/pages/baseline/page";
import BaselineMonthPage from "@/pages/baseline/month/page";
import ChatPage from "@/pages/chat/page";
import ImportsPage from "@/pages/imports/page";
import PlanPage from "@/pages/plan/page";
import ReviewPage from "@/pages/review/page";
import ReviewDetailPage from "@/pages/reviews/detail";
import ReviewsPage from "@/pages/reviews/page";
import TransactionsPage from "@/pages/transactions/page";
import { parseBaselineSearchParams } from "@/pages/baseline/search-params";
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
    title: "Baseline",
  },
  validateSearch: (search) =>
    parseBaselineSearchParams(search as Record<string, unknown>),
});

const baselineMonthRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/baseline/months/$yyyyMm",
  component: BaselineMonthPage,
  staticData: {
    title: "Baseline",
  },
  validateSearch: (search) =>
    parseBaselineSearchParams(search as Record<string, unknown>),
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
  baselineMonthRoute,
  baselineRoute,
  chatRoute,
  importsRoute,
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
  baselineMonthRoute,
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
