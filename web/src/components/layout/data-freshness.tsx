"use client";

import { RefreshCw } from "lucide-react";

import { Button } from "@/components/ui/button";
import { formatDateTime } from "@/lib/format-trading";

type DataFreshnessProps = {
  lastUpdated: Date | null;
  onRefresh?: () => void;
  refreshing?: boolean;
  /** 例如「每 4s 自动刷新」 */
  hint?: string;
};

export function DataFreshness({
  lastUpdated,
  onRefresh,
  refreshing,
  hint,
}: DataFreshnessProps) {
  return (
    <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
      <span>
        最后更新：
        {lastUpdated ? (
          <time
            dateTime={lastUpdated.toISOString()}
            className="ml-1 font-medium text-foreground"
          >
            {formatDateTime(lastUpdated.toISOString())}
          </time>
        ) : (
          <span className="ml-1">—</span>
        )}
      </span>
      {hint ? <span className="text-muted-foreground/80">· {hint}</span> : null}
      {onRefresh ? (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-7 gap-1 px-2 text-xs"
          disabled={refreshing}
          onClick={() => void onRefresh()}
        >
          <RefreshCw
            className={`h-3.5 w-3.5 ${refreshing ? "animate-spin" : ""}`}
          />
          刷新
        </Button>
      ) : null}
    </div>
  );
}
