import { QueryClientProvider } from "@tanstack/react-query";
import { render, type RenderOptions, type RenderResult } from "@testing-library/react";
import { createMemoryHistory, RouterProvider } from "@tanstack/react-router";
import type React from "react";
import { isValidElement, useState } from "react";

import { LocaleSync } from "@/components/locale-sync";
import { ThemeProvider } from "@/components/theme-provider";
import { createAppRouter } from "@/router";
import "@/i18n";

import { createTestQueryClient } from "./query-client";

const TestProviders: React.FC<React.PropsWithChildren> = ({ children }) => {
  const [queryClient] = useState(() => createTestQueryClient());

  return (
    <ThemeProvider>
      <QueryClientProvider client={queryClient}>
        <LocaleSync />
        {children}
      </QueryClientProvider>
    </ThemeProvider>
  );
};

type TestRenderOptions = Omit<RenderOptions, "wrapper"> & {
  /** Mount the app router at this path. Omit when rendering a component directly. */
  route?: string;
};

export function testRender(
  ui: React.ReactElement,
  options?: Omit<RenderOptions, "wrapper">,
): RenderResult;

export function testRender(options?: TestRenderOptions): RenderResult;

export function testRender(
  uiOrOptions?: React.ReactElement | TestRenderOptions,
  maybeOptions?: Omit<RenderOptions, "wrapper">,
): RenderResult {
  if (isValidElement(uiOrOptions)) {
    return render(uiOrOptions, {
      wrapper: TestProviders,
      ...maybeOptions,
    });
  }

  const { route = "/transactions", ...renderOptions } = uiOrOptions ?? {};
  const history = createMemoryHistory({ initialEntries: [route] });
  const router = createAppRouter(history);

  return render(
    <TestProviders>
      <RouterProvider router={router} />
    </TestProviders>,
    renderOptions,
  );
}
