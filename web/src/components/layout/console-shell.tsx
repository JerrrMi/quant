"use client";

import { useState, type ReactNode } from "react";

import { AppSidebar } from "@/components/layout/app-sidebar";
import { CommandStreamDrawer } from "@/features/commands/command-stream-drawer";
import { SidebarLinks } from "@/components/layout/sidebar-links";
import { TopBar } from "@/components/layout/top-bar";
import { publicEnv } from "@/lib/env";
import { useSidebarCollapsed } from "@/hooks/use-sidebar-collapsed";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";

export function ConsoleShell({ children }: { children: ReactNode }) {
  const [mobileOpen, setMobileOpen] = useState(false);
  const [commandOpen, setCommandOpen] = useState(false);
  const { collapsed, toggle } = useSidebarCollapsed();

  return (
    <div className="flex min-h-screen bg-background">
      <AppSidebar collapsed={collapsed} onToggleCollapsed={toggle} />

      <div className="flex min-h-screen flex-1 flex-col">
        <TopBar
          onOpenSidebar={() => setMobileOpen(true)}
          onOpenCommandStream={() => setCommandOpen(true)}
        />
        <main className="flex-1 bg-muted/25">{children}</main>
      </div>

      <CommandStreamDrawer open={commandOpen} onOpenChange={setCommandOpen} />

      <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
        <SheetContent side="left" className="flex w-[260px] flex-col gap-0 p-0">
          <SheetHeader className="sr-only">
            <SheetTitle>主导航</SheetTitle>
            <SheetDescription>{publicEnv.appName} 导航菜单</SheetDescription>
          </SheetHeader>
          <div className="flex h-14 items-center gap-2 border-b px-4">
            <div className="flex h-9 w-9 items-center justify-center rounded-md bg-primary text-xs font-semibold text-primary-foreground">
              AS
            </div>
            <div className="leading-tight">
              <div className="text-sm font-semibold">{publicEnv.appName}</div>
              <div className="text-[11px] text-muted-foreground">Trading Console</div>
            </div>
          </div>
          <SidebarLinks onNavigate={() => setMobileOpen(false)} />
        </SheetContent>
      </Sheet>
    </div>
  );
}
