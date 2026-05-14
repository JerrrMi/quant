"use client";

import Link from "next/link";
import { LogOut, Menu, ListOrdered, UserRound } from "lucide-react";

import { useConfirm } from "@/components/feedback/confirm-provider";
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
  onOpenCommandStream?: () => void;
};

export function TopBar({ onOpenSidebar, onOpenCommandStream }: TopBarProps) {
  const { user, logout } = useAuth();
  const confirm = useConfirm();
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
          <div className="hidden items-center gap-2 rounded-full border border-border bg-muted/60 px-3 py-1 text-xs font-medium text-muted-foreground sm:flex">
            <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" aria-hidden />
            会话有效
          </div>
        </TooltipTrigger>
        <TooltipContent>
          会话由 Cookie / Bearer（若下发）共同决定；详情见登录与 API 封装。
        </TooltipContent>
      </Tooltip>

      <ThemeSwitcher />

      {onOpenCommandStream ? (
        <Button
          type="button"
          variant="secondary"
          size="sm"
          className="hidden gap-1 sm:inline-flex"
          onClick={() => onOpenCommandStream()}
        >
          <ListOrdered className="h-4 w-4" />
          命令流
        </Button>
      ) : null}
      <Button type="button" variant="ghost" size="sm" className="hidden sm:inline-flex" asChild>
        <Link href="/commands" className="text-xs text-muted-foreground">
          命令页
        </Link>
      </Button>

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
              void (async () => {
                const ok = await confirm({
                  title: "确认退出登录？",
                  description: "将清除本地凭据并回到登录页（httpOnly 会话由服务端注销）。",
                  confirmLabel: "退出",
                  cancelLabel: "取消",
                  destructive: true,
                });
                if (ok) await logout();
              })();
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
