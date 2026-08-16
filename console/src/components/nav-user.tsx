import type React from "react";
import { useNavigate } from "@tanstack/react-router";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ChevronsUpDownIcon, LogOutIcon, UserRoundIcon } from "lucide-react";
import { useTranslation } from "react-i18next";

import { postLogoutMutation } from "@/api/@tanstack/react-query.gen";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar";
import { useMe } from "@/hooks/use-me";
import { initialsFromName, primaryDisplayName } from "@/lib/user-display";

export const NavUser: React.FC = () => {
  const { t } = useTranslation();
  const { isMobile } = useSidebar();
  const me = useMe();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const logout = useMutation({
    ...postLogoutMutation(),
    onSettled: async () => {
      queryClient.clear();
      await navigate({ to: "/login" });
    },
  });

  const user = me.data?.user;
  if (!user) {
    return null;
  }

  const name = primaryDisplayName({
    givenName: user.given_name,
    displayName: user.display_name,
    email: user.email,
  });
  const email = user.email ?? "";
  const initials = initialsFromName(
    user.given_name?.trim() || user.display_name || name,
  );

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <SidebarMenuButton
                size="lg"
                className="data-open:bg-sidebar-accent data-open:text-sidebar-accent-foreground"
                tooltip={name}
              />
            }
          >
            <Avatar size="sm" className="rounded-lg">
              {user.picture_url ? (
                <AvatarImage src={user.picture_url} alt={name} />
              ) : null}
              <AvatarFallback className="rounded-lg">{initials}</AvatarFallback>
            </Avatar>
            <div className="grid flex-1 text-left text-sm leading-tight">
              <span className="truncate font-medium">{name}</span>
              {email ? (
                <span className="truncate text-xs text-muted-foreground">
                  {email}
                </span>
              ) : null}
            </div>
            <ChevronsUpDownIcon className="ml-auto size-4" />
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className="min-w-56 rounded-lg"
            side={isMobile ? "bottom" : "top"}
            align="end"
            sideOffset={4}
          >
            <DropdownMenuItem
              className="gap-2"
              onClick={() => {
                void navigate({ to: "/account" });
              }}
            >
              <UserRoundIcon className="size-4" />
              {t("nav.account")}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              disabled={logout.isPending}
              className="gap-2"
              onClick={() => {
                logout.mutate({});
              }}
            >
              <LogOutIcon className="size-4" />
              {t("nav.logOut")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
};
