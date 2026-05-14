"use client";

import { cva } from "class-variance-authority";

import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import type { SystemStatusKind } from "@/types/console";

const styles = cva("", {
  variants: {
    status: {
      online:
        "border-emerald-500/40 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
      offline: "border-border bg-muted text-muted-foreground",
      running:
        "border-sky-500/40 bg-sky-500/10 text-sky-800 dark:text-sky-200",
      paused:
        "border-amber-500/40 bg-amber-500/10 text-amber-900 dark:text-amber-100",
      failed: "border-destructive/40 bg-destructive/10 text-destructive",
      idle: "border-border bg-secondary text-secondary-foreground",
    },
  },
});

const labels: Record<SystemStatusKind, string> = {
  online: "Online",
  offline: "Offline",
  running: "Running",
  paused: "Paused",
  failed: "Failed",
  idle: "Idle",
};

type StatusTagProps = {
  status: SystemStatusKind;
  label?: string;
  className?: string;
};

export function StatusTag({ status, label, className }: StatusTagProps) {
  const text = label ?? labels[status];
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Badge
          variant="outline"
          className={cn(
            "rounded-full px-2 py-0 text-[11px] font-semibold uppercase tracking-wide",
            styles({ status }),
            className,
          )}
        >
          {text}
        </Badge>
      </TooltipTrigger>
      <TooltipContent>{text}</TooltipContent>
    </Tooltip>
  );
}
