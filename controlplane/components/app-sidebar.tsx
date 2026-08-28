"use client";

import { useRouter } from "next/navigation";
import { FlaskConical, List, Shield } from "lucide-react";
import { useTranslations } from "next-intl";
import { NavUser } from "@/components/nav-user";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
  useSidebar,
} from "@/components/ui/sidebar";
import { type ViewId, viewHref } from "@/lib/view";

export function AppSidebar({
  view,
  hitsCount,
}: {
  view: ViewId;
  hitsCount: number;
}) {
  const tShell = useTranslations("shell");
  const tNav = useTranslations("tabs");
  const router = useRouter();
  const { state } = useSidebar();

  const items: { id: ViewId; icon: typeof Shield; label: string }[] = [
    { id: "rules", icon: Shield, label: tNav("rules") },
    { id: "sandbox", icon: FlaskConical, label: tNav("sandbox") },
    { id: "hits", icon: List, label: tNav("hits") },
  ];

  return (
    <Sidebar collapsible="icon">
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton className="pointer-events-none" size="lg" type="button">
              <div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary font-medium text-sidebar-primary-foreground text-xs">
                CG
              </div>
              <div className="grid min-w-0 flex-1 text-start leading-tight">
                <span className="truncate font-medium">{tShell("product")}</span>
                <span className="truncate text-sidebar-foreground/70 text-xs">
                  {tShell("workspace")}
                </span>
              </div>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>{tShell("navGroup")}</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {items.map((item) => {
                const Icon = item.icon;
                return (
                  <SidebarMenuItem key={item.id}>
                    {/* Ark Button 的 asChild 会拦掉 <a> 默认导航，侧栏用 router.push */}
                    <SidebarMenuButton
                      isActive={view === item.id}
                      onClick={() => router.push(viewHref(item.id))}
                      tooltip={state === "collapsed" ? item.label : undefined}
                      type="button"
                    >
                      <Icon aria-hidden className="size-4" />
                      <span>{item.label}</span>
                    </SidebarMenuButton>
                    {item.id === "hits" && hitsCount > 0 ? (
                      <SidebarMenuBadge>{hitsCount}</SidebarMenuBadge>
                    ) : null}
                  </SidebarMenuItem>
                );
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter>
        <NavUser />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}
