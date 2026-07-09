import { render, type RenderOptions } from "@testing-library/react";
import { createMemoryHistory, RouterProvider } from "@tanstack/react-router";

import { ThemeProvider } from "@/components/theme-provider";
import { createAppRouter } from "@/router";

type RenderWithRouterOptions = Omit<RenderOptions, "wrapper">;

export function renderWithRouter(
  initialRoute = "/transactions",
  options?: RenderWithRouterOptions,
) {
  const history = createMemoryHistory({ initialEntries: [initialRoute] });
  const router = createAppRouter(history);

  return render(
    <ThemeProvider>
      <RouterProvider router={router} />
    </ThemeProvider>,
    options,
  );
}
