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
import { useTranslation } from "react-i18next";

import { NavUser } from "@/components/nav-user";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
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
  const { t } = useTranslation();
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
  const categoriesActive = pathActive(pathname, "/insights/categories", true);
  const insightsActive =
    pathActive(pathname, "/insights") || monthsActive || categoriesActive;

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
                <span className="truncate font-semibold">
                  {t("common.brand")}
                </span>
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
                tooltip={t("nav.chat")}
                isActive={pathActive(pathname, "/chat", true)}
              >
                <MessageSquareIcon />
                <span>{t("nav.chat")}</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
            <SidebarMenuItem>
              <SidebarMenuButton
                tooltip={t("nav.baseline")}
                isActive={baselineActive}
              >
                <GaugeIcon />
                <span>{t("nav.baseline")}</span>
              </SidebarMenuButton>
              <SidebarMenuSub>
                <SidebarMenuSubItem>
                  <SidebarMenuSubButton
                    render={<Link to="/baseline" />}
                    isActive={cashflowActive}
                  >
                    {t("nav.cashflow")}
                  </SidebarMenuSubButton>
                </SidebarMenuSubItem>
                <SidebarMenuSubItem>
                  <SidebarMenuSubButton
                    render={<Link to="/baseline/income" />}
                    isActive={incomeActive}
                  >
                    {t("nav.income")}
                  </SidebarMenuSubButton>
                </SidebarMenuSubItem>
                <SidebarMenuSubItem>
                  <SidebarMenuSubButton
                    render={<Link to="/baseline/expenses" />}
                    isActive={expensesActive}
                  >
                    {t("nav.expenses")}
                  </SidebarMenuSubButton>
                </SidebarMenuSubItem>
                <SidebarMenuSubItem>
                  <SidebarMenuSubButton
                    render={<Link to="/baseline/tracking" />}
                    isActive={trackingActive}
                  >
                    {t("nav.tracking")}
                  </SidebarMenuSubButton>
                </SidebarMenuSubItem>
              </SidebarMenuSub>
            </SidebarMenuItem>
            <SidebarMenuItem>
              <SidebarMenuButton
                tooltip={t("nav.insights")}
                isActive={insightsActive}
              >
                <LineChartIcon />
                <span>{t("nav.insights")}</span>
              </SidebarMenuButton>
              <SidebarMenuSub>
                <SidebarMenuSubItem>
                  <SidebarMenuSubButton
                    render={<Link to="/insights/months" />}
                    isActive={monthsActive}
                  >
                    {t("nav.months")}
                  </SidebarMenuSubButton>
                </SidebarMenuSubItem>
                <SidebarMenuSubItem>
                  <SidebarMenuSubButton
                    render={<Link to="/insights/categories" />}
                    isActive={categoriesActive}
                  >
                    {t("nav.categories")}
                  </SidebarMenuSubButton>
                </SidebarMenuSubItem>
              </SidebarMenuSub>
            </SidebarMenuItem>
            <SidebarMenuItem>
              <SidebarMenuButton
                render={<Link to="/reviews" />}
                tooltip={t("nav.reviews")}
                isActive={pathActive(pathname, "/reviews")}
              >
                <ClipboardListIcon />
                <span>{t("nav.reviews")}</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
            <SidebarMenuItem>
              <SidebarMenuButton
                render={<Link to="/plan" />}
                tooltip={t("nav.plan")}
                isActive={pathActive(pathname, "/plan", true)}
              >
                <TrendingUpIcon />
                <span>{t("nav.plan")}</span>
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
                tooltip={t("nav.transactions")}
                isActive={pathActive(pathname, "/transactions", true)}
              >
                <Table2Icon />
                <span>{t("nav.transactions")}</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
            <SidebarMenuItem>
              <SidebarMenuButton
                render={<Link to="/imports" />}
                tooltip={t("nav.imports")}
                isActive={pathActive(pathname, "/imports", true)}
              >
                <UploadIcon />
                <span>{t("nav.imports")}</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
            <SidebarMenuItem>
              <SidebarMenuButton
                render={<Link to="/review" />}
                tooltip={t("nav.needsReview")}
                isActive={pathActive(pathname, "/review", true)}
              >
                <InboxIcon />
                <span>{t("nav.needsReview")}</span>
              </SidebarMenuButton>
              {pendingCount > 0 ? (
                <SidebarMenuBadge>{pendingCount}</SidebarMenuBadge>
              ) : null}
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter>
        <NavUser />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
};
