import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { RouterProvider } from "@tanstack/react-router";

import { AppProviders } from "@/app/providers";
import { LocaleSync } from "@/components/locale-sync";
import { ThemeProvider } from "@/components/theme-provider";
import { router } from "@/router";
import "@/i18n";
import "@/lib/api-client";
import "./index.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ThemeProvider>
      <AppProviders>
        <LocaleSync />
        <RouterProvider router={router} />
      </AppProviders>
    </ThemeProvider>
  </StrictMode>,
);
