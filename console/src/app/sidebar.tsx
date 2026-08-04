import type React from "react";
import { Link, useRouterState } from "@tanstack/react-router";
import {
  ClipboardListIcon,
  GaugeIcon,
  InboxIcon,
  LineChartIcon,
  MessageSquareIcon,
  Table2Icon,
  TrendingUpIcon,
  UploadIcon,
  WalletIcon,
} from "lucide-react";

import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarRail,
} from "@/components/ui/sidebar";
import { useClassificationQueue } from "@/hooks/use-classification-queue";
import { useUncertainRecurring } from "@/hooks/use-recurring-uncertain";
import { useTransferCandidates } from "@/hooks/use-transfer-candidates";
import { defaultTransactionSearchParams } from "@/pages/transactions/search-params";

function pathActive(pathname: string, target: string, exact = false): boolean {
  if (exact) {
    return pathname === target;
  }
  return pathname === target || pathname.startsWith(`${target}/`);
}

export const AppSidebar: React.FC = () => {
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const candidatesQuery = useTransferCandidates();
  const classificationQuery = useClassificationQueue();
  const recurringQuery = useUncertainRecurring();
  const pendingCount =
    (candidatesQuery.data?.data.length ?? 0) +
    (classificationQuery.data?.data.length ?? 0) +
    (recurringQuery.data?.data.length ?? 0);

  const cashflowActive = pathActive(pathname, "/baseline", true);
  const incomeActive = pathActive(pathname, "/baseline/income", true);
  const expensesActive = pathActive(pathname, "/baseline/expenses", true);
  const trackingActive =
    pathActive(pathname, "/baseline/tracking", true) ||
    pathActive(pathname, "/baseline/performance", true);
  const baselineActive =
    cashflowActive || incomeActive || expensesActive || trackingActive;
  const monthsActive =
    pathActive(pathname, "/insights/months") ||
    pathActive(pathname, "/baseline/history") ||
    pathActive(pathname, "/baseline/months");
  const insightsActive = pathActive(pathname, "/insights") || monthsActive;

  return (
    <Sidebar collapsible="icon">
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton size="lg">
              <div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground">
                <WalletIcon className="size-4" />
              </div>
              <div className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-semibold">assetagent</span>
              </div>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton
                render={<Link to="/chat" />}
                tooltip="Chat"
                isActive={pathActive(pathname, "/chat", true)}
              >
                <MessageSquareIcon />
                <span>Chat</span>
              </SidebarMenuButton>
            </SidebarMenuItem>

            <SidebarMenuItem>
              <SidebarMenuButton
                render={<Link to="/baseline" />}
                tooltip="Baseline"
                isActive={baselineActive}
              >
                <GaugeIcon />
                <span>Baseline</span>
              </SidebarMenuButton>
              <SidebarMenuSub>
                <SidebarMenuSubItem>
                  <SidebarMenuSubButton
                    render={<Link to="/baseline/income" />}
                    isActive={incomeActive}
                  >
                    Income
                  </SidebarMenuSubButton>
                </SidebarMenuSubItem>
                <SidebarMenuSubItem>
                  <SidebarMenuSubButton
                    render={<Link to="/baseline/expenses" />}
                    isActive={expensesActive}
                  >
                    Expenses
                  </SidebarMenuSubButton>
                </SidebarMenuSubItem>
                <SidebarMenuSubItem>
                  <SidebarMenuSubButton
                    render={<Link to="/baseline" />}
                    isActive={cashflowActive}
                  >
                    Cashflow
                  </SidebarMenuSubButton>
                </SidebarMenuSubItem>
                <SidebarMenuSubItem>
                  <SidebarMenuSubButton
                    render={<Link to="/baseline/tracking" />}
                    isActive={trackingActive}
                  >
                    Tracking
                  </SidebarMenuSubButton>
                </SidebarMenuSubItem>
              </SidebarMenuSub>
            </SidebarMenuItem>

            <SidebarMenuItem>
              <SidebarMenuButton
                render={<Link to="/insights/months" />}
                tooltip="Insights"
                isActive={insightsActive}
              >
                <LineChartIcon />
                <span>Insights</span>
              </SidebarMenuButton>
              <SidebarMenuSub>
                <SidebarMenuSubItem>
                  <SidebarMenuSubButton
                    render={<Link to="/insights/months" />}
                    isActive={monthsActive}
                  >
                    Months
                  </SidebarMenuSubButton>
                </SidebarMenuSubItem>
              </SidebarMenuSub>
            </SidebarMenuItem>

            <SidebarMenuItem>
              <SidebarMenuButton
                render={<Link to="/reviews" />}
                tooltip="Money Reviews"
                isActive={pathActive(pathname, "/reviews")}
              >
                <ClipboardListIcon />
                <span>Money Reviews</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
            <SidebarMenuItem>
              <SidebarMenuButton
                render={<Link to="/plan" />}
                tooltip="Plan"
                isActive={pathActive(pathname, "/plan", true)}
              >
                <TrendingUpIcon />
                <span>Plan</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
            <SidebarMenuItem>
              <SidebarMenuButton
                render={
                  <Link
                    to="/transactions"
                    search={defaultTransactionSearchParams}
                  />
                }
                tooltip="Transactions"
                isActive={pathActive(pathname, "/transactions", true)}
              >
                <Table2Icon />
                <span>Transactions</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
            <SidebarMenuItem>
              <SidebarMenuButton
                render={<Link to="/imports" />}
                tooltip="Import"
                isActive={pathActive(pathname, "/imports", true)}
              >
                <UploadIcon />
                <span>Import</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
            <SidebarMenuItem>
              <SidebarMenuButton
                render={<Link to="/review" />}
                tooltip="Needs review"
                isActive={pathActive(pathname, "/review", true)}
              >
                <InboxIcon />
                <span>Needs review</span>
              </SidebarMenuButton>
              {pendingCount > 0 ? (
                <SidebarMenuBadge>{pendingCount}</SidebarMenuBadge>
              ) : null}
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroup>
      </SidebarContent>
      <SidebarRail />
    </Sidebar>
  );
};
