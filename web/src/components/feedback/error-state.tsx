import { AlertTriangle } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { cn } from "@/lib/utils";

type ErrorStateProps = {
  title?: string;
  description?: string;
  onRetry?: () => void;
  retryLabel?: string;
  className?: string;
};

export function ErrorState({
  title = "加载失败",
  description = "请稍后重试或联系管理员。",
  onRetry,
  retryLabel = "重试",
  className,
}: ErrorStateProps) {
  return (
    <Card className={cn("border-destructive/40", className)}>
      <CardHeader className="flex flex-row items-start gap-3 space-y-0">
        <div className="flex h-10 w-10 items-center justify-center rounded-md border border-destructive/40 bg-destructive/10">
          <AlertTriangle className="h-5 w-5 text-destructive" aria-hidden />
        </div>
        <div className="space-y-1">
          <CardTitle className="text-base">{title}</CardTitle>
          <CardDescription>{description}</CardDescription>
        </div>
      </CardHeader>
      {onRetry ? (
        <CardFooter>
          <Button type="button" variant="outline" onClick={onRetry}>
            {retryLabel}
          </Button>
        </CardFooter>
      ) : null}
    </Card>
  );
}
