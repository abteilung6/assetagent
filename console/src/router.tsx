import {
  createRootRoute,
  createRoute,
  createRouter,
  redirect,
  type RouterHistory,
} from "@tanstack/react-router";

import { AppLayout } from "@/app/layout";
import ChatPage from "@/pages/chat/page";
import ImportsPage from "@/pages/imports/page";
import TransactionsPage from "@/pages/transactions/page";
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
});

const importsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/imports",
  component: ImportsPage,
  staticData: {
    title: "Import",
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

export { chatRoute, importsRoute, transactionsRoute };

const routeTree = rootRoute.addChildren([
  indexRoute,
  chatRoute,
  importsRoute,
  transactionsRoute,
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
