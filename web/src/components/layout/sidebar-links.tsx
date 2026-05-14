"use client";

import type { ReactElement } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";

import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { mainNav } from "@/lib/nav-config";
import { cn } from "@/lib/utils";

type SidebarLinksProps = {
  collapsed?: boolean;
  onNavigate?: () => void;
};

export function SidebarLinks({ collapsed, onNavigate }: SidebarLinksProps) {
  const pathname = usePathname();

  const wrapCollapsed = (label: string, node: ReactElement) =>
    collapsed ? (
      <Tooltip>
        <TooltipTrigger asChild>{node}</TooltipTrigger>
        <TooltipContent side="right">{label}</TooltipContent>
      </Tooltip>
    ) : (
      node
    );

  return (
    <nav className="flex flex-1 flex-col gap-2 px-3 pb-6 pt-4">
      {mainNav.map((item) => {
        if (item.kind === "link") {
          const Icon = item.icon;
          const active =
            pathname === item.href ||
            (item.href !== "/dashboard" && pathname.startsWith(`${item.href}/`));

          const link = (
            <Link
              href={item.href}
              onClick={onNavigate}
              className={cn(
                "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground",
                active && "bg-accent text-accent-foreground",
                collapsed && "justify-center px-2",
              )}
            >
              <Icon className="h-4 w-4 shrink-0 text-muted-foreground" />
              {!collapsed ? <span>{item.title}</span> : null}
            </Link>
          );

          return (
            <div key={item.href}>
              {collapsed ? wrapCollapsed(item.title, link) : link}
            </div>
          );
        }

        const Icon = item.icon;

        return (
          <div key={item.title} className="space-y-1">
            <div
              className={cn(
                "flex items-center gap-3 px-3 py-2 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground",
                collapsed && "justify-center px-0",
              )}
            >
              <Icon className="h-4 w-4" />
              {!collapsed ? <span>{item.title}</span> : null}
            </div>
            <div className={cn("space-y-1", !collapsed && "pl-3")}>
              {item.children.map((child) => {
                const ChildIcon = child.icon;
                const active =
                  pathname === child.href || pathname.startsWith(`${child.href}/`);
                const link = (
                  <Link
                    href={child.href}
                    onClick={onNavigate}
                    className={cn(
                      "flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors hover:bg-accent hover:text-accent-foreground",
                      active && "bg-accent text-accent-foreground",
                      collapsed && "justify-center px-2",
                    )}
                  >
                    <ChildIcon className="h-4 w-4 shrink-0 text-muted-foreground" />
                    {!collapsed ? <span>{child.title}</span> : null}
                  </Link>
                );

                return (
                  <div key={child.href}>
                    {collapsed ? wrapCollapsed(child.title, link) : link}
                  </div>
                );
              })}
            </div>
          </div>
        );
      })}
    </nav>
  );
}
