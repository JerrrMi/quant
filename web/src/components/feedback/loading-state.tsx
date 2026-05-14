import { Loader2 } from "lucide-react";

import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

type LoadingStateProps = {
  label?: string;
  variant?: "spinner" | "skeleton";
  lines?: number;
  className?: string;
};

export function LoadingState({
  label = "加载中…",
  variant = "spinner",
  lines = 3,
  className,
}: LoadingStateProps) {
  if (variant === "skeleton") {
    return (
      <Card className={cn("border-border/70", className)}>
        <CardContent className="space-y-3 pt-6">
          <Skeleton className="h-4 w-40" />
          {Array.from({ length: lines }).map((_, index) => (
            <Skeleton key={index} className="h-3 w-full" />
          ))}
        </CardContent>
      </Card>
    );
  }

  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-border px-6 py-10 text-sm text-muted-foreground",
        className,
      )}
    >
      <Loader2 className="h-6 w-6 animate-spin text-primary" aria-hidden />
      <span>{label}</span>
    </div>
  );
}
