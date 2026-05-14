"use client";

import { LogOut, Menu, UserRound, Wifi } from "lucide-react";

import { ThemeSwitcher } from "@/components/layout/theme-switcher";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useAuth } from "@/features/auth/auth-context";
import { publicEnv } from "@/lib/env";
import { cn } from "@/lib/utils";

const envLabel: Record<string, string> = {
  development: "开发",
  staging: "预发",
  production: "生产",
  local: "本地",
};

type TopBarProps = {
  onOpenSidebar: () => void;
};

export function TopBar({ onOpenSidebar }: TopBarProps) {
  const { user, logout } = useAuth();
  const envName = envLabel[publicEnv.deployEnv] ?? publicEnv.deployEnv;

  return (
    <header className="sticky top-0 z-40 flex h-14 items-center gap-3 border-b border-border bg-background/90 px-4 backdrop-blur-md">
      <Button
        type="button"
        variant="outline"
        size="icon"
        className="md:hidden"
        aria-label="打开导航"
        onClick={onOpenSidebar}
      >
        <Menu className="h-4 w-4" />
      </Button>

      <div className="flex min-w-0 flex-1 flex-wrap items-center gap-2">
        <span className="truncate text-sm font-medium text-muted-foreground">
          控制台总览
        </span>
        <Badge variant="secondary" className="text-[11px] font-semibold">
          {envName}
        </Badge>
      </div>

      <Tooltip>
        <TooltipTrigger asChild>
          <div className="hidden items-center gap-2 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-3 py-1 text-xs font-medium text-emerald-700 dark:text-emerald-300 sm:flex">
            <Wifi className="h-3.5 w-3.5" aria-hidden />
            已连接
          </div>
        </TooltipTrigger>
        <TooltipContent>控制台 API · WebSocket / SSE 接入点在适配层预留</TooltipContent>
      </Tooltip>

      <ThemeSwitcher />

      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm" className="gap-2">
            <UserRound className="h-4 w-4" />
            <span className="hidden max-w-[140px] truncate sm:inline">
              {user?.displayName ?? user?.email ?? "用户"}
            </span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-52">
          <DropdownMenuLabel className="font-normal">
            <div className="flex flex-col gap-1">
              <span className="text-sm font-medium leading-none">
                {user?.displayName ?? "控制台用户"}
              </span>
              <span className="text-xs text-muted-foreground">{user?.email}</span>
            </div>
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            className={cn("gap-2 text-destructive focus:text-destructive")}
            onSelect={(event) => {
              event.preventDefault();
              void logout();
            }}
          >
            <LogOut className="h-4 w-4" />
            退出登录
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </header>
  );
}
