"use client";

import { useRouter } from "next/navigation";
import { ChevronsUpDown, LogOut, Settings, UserRound } from "lucide-react";
import { useTranslations } from "next-intl";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  Menu,
  MenuContent,
  MenuGroup,
  MenuItem,
  MenuSeparator,
  MenuTrigger,
} from "@/components/ui/menu";
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { viewHref } from "@/lib/view";

export function NavUser() {
  const t = useTranslations("user");
  const router = useRouter();

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <Menu
          onSelect={(details) => {
            if (details.value === "settings") {
              router.push(viewHref("settings"));
            }
          }}
          positioning={{ placement: "top-start" }}
        >
          <MenuTrigger asChild>
            <SidebarMenuButton size="lg" type="button">
              <Avatar>
                <AvatarFallback>{t("initials")}</AvatarFallback>
              </Avatar>
              <div className="grid min-w-0 flex-1 text-start leading-tight">
                <span className="truncate font-medium">{t("name")}</span>
                <span className="truncate text-sidebar-foreground/70 text-xs">
                  {t("email")}
                </span>
              </div>
              <ChevronsUpDown aria-hidden className="ms-auto size-4 group-data-[collapsible=icon]:hidden" />
            </SidebarMenuButton>
          </MenuTrigger>
          <MenuContent className="w-56">
            <MenuGroup>
              <div className="flex items-center gap-2 px-2.5 py-1.5">
                <Avatar>
                  <AvatarFallback>{t("initials")}</AvatarFallback>
                </Avatar>
                <div className="grid min-w-0 leading-tight">
                  <span className="truncate font-medium text-sm">{t("name")}</span>
                  <span className="truncate text-muted-foreground text-xs">
                    {t("email")}
                  </span>
                </div>
              </div>
            </MenuGroup>
            <MenuSeparator />
            <MenuGroup>
              <MenuItem
                onClick={() => router.push(viewHref("settings"))}
                value="settings"
              >
                <Settings aria-hidden />
                {t("settings")}
              </MenuItem>
              <MenuItem disabled value="account">
                <UserRound aria-hidden />
                {t("account")}
              </MenuItem>
            </MenuGroup>
            <MenuSeparator />
            <MenuGroup>
              <MenuItem disabled value="sign-out">
                <LogOut aria-hidden />
                {t("signOut")}
              </MenuItem>
            </MenuGroup>
          </MenuContent>
        </Menu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}
