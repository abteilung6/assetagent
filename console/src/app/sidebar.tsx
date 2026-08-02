import type React from "react";
import { Link } from "@tanstack/react-router";
import {
  InboxIcon,
  MessageSquareIcon,
  Table2Icon,
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
  SidebarRail,
} from "@/components/ui/sidebar";
import { useClassificationQueue } from "@/hooks/use-classification-queue";
import { useUncertainRecurring } from "@/hooks/use-recurring-uncertain";
import { useTransferCandidates } from "@/hooks/use-transfer-candidates";
import { defaultTransactionSearchParams } from "@/pages/transactions/search-params";

export const AppSidebar: React.FC = () => {
  const candidatesQuery = useTransferCandidates();
  const classificationQuery = useClassificationQueue();
  const recurringQuery = useUncertainRecurring();
  const pendingCount =
    (candidatesQuery.data?.data.length ?? 0) +
    (classificationQuery.data?.data.length ?? 0) +
    (recurringQuery.data?.data.length ?? 0);

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
              >
                <MessageSquareIcon />
                <span>Chat</span>
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
              >
                <Table2Icon />
                <span>Transactions</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
            <SidebarMenuItem>
              <SidebarMenuButton
                render={<Link to="/imports" />}
                tooltip="Import"
              >
                <UploadIcon />
                <span>Import</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
            <SidebarMenuItem>
              <SidebarMenuButton
                render={<Link to="/review" />}
                tooltip="Needs review"
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
