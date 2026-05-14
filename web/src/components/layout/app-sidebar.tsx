"use client";

import { ChevronLeft } from "lucide-react";

import { SidebarLinks } from "@/components/layout/sidebar-links";
import { Button } from "@/components/ui/button";
import { publicEnv } from "@/lib/env";
import { cn } from "@/lib/utils";

type AppSidebarProps = {
  collapsed: boolean;
  onToggleCollapsed: () => void;
};

export function AppSidebar({ collapsed, onToggleCollapsed }: AppSidebarProps) {
  return (
    <aside
      className={cn(
        "relative hidden shrink-0 flex-col border-r border-border bg-card/70 backdrop-blur-sm transition-[width] duration-200 md:flex",
        collapsed ? "w-[72px]" : "w-56",
      )}
    >
      <div className="flex h-14 items-center gap-2 border-b border-border px-3">
        <div className="flex h-9 w-9 items-center justify-center rounded-md bg-primary text-xs font-semibold text-primary-foreground">
          AS
        </div>
        {!collapsed ? (
          <div className="min-w-0 leading-tight">
            <div className="truncate text-sm font-semibold">{publicEnv.appName}</div>
            <div className="text-[11px] text-muted-foreground">Trading Console</div>
          </div>
        ) : null}
      </div>

      <SidebarLinks collapsed={collapsed} />

      <div className="mt-auto border-t border-border p-2">
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="w-full"
          aria-label={collapsed ? "展开侧边栏" : "折叠侧边栏"}
          onClick={onToggleCollapsed}
        >
          <ChevronLeft
            className={cn("h-4 w-4 transition-transform", collapsed && "rotate-180")}
          />
        </Button>
      </div>
    </aside>
  );
}
