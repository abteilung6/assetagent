import type React from "react";
import { Outlet } from "@tanstack/react-router";

import { AppSidebar } from "@/app/sidebar";
import { PageTitle } from "@/app/page-title";
import { Separator } from "@/components/ui/separator";
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar";
import { TooltipProvider } from "@/components/ui/tooltip";
import { HealthStatus } from "@/components/health-status";
import { ThemeToggle } from "@/components/theme-toggle";

export const AppLayout: React.FC = () => {
  return (
    <TooltipProvider>
      <SidebarProvider className="h-svh max-h-svh overflow-hidden">
        <AppSidebar />
        <SidebarInset className="min-h-0 overflow-hidden">
          <header className="flex h-16 shrink-0 items-center gap-2 border-b transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-12">
            <div className="flex min-w-0 flex-1 items-center gap-2 px-4">
              <SidebarTrigger className="-ml-1" />
              <Separator
                orientation="vertical"
                className="mr-2 data-vertical:h-4 data-vertical:self-auto"
              />
              <PageTitle />
            </div>
            <div className="ml-auto flex items-center gap-4 px-4">
              <HealthStatus />
              <ThemeToggle />
            </div>
          </header>
          <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden p-4">
            <Outlet />
          </div>
        </SidebarInset>
      </SidebarProvider>
    </TooltipProvider>
  );
};
