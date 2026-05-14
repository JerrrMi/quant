"use client";

import type { ReactNode } from "react";

import { ErrorState } from "@/components/feedback/error-state";
import { LoadingState } from "@/components/feedback/loading-state";

type AsyncBoundaryProps = {
  loading?: boolean;
  error?: Error | null;
  onRetry?: () => void;
  children: ReactNode;
};

export function AsyncBoundary({
  loading,
  error,
  onRetry,
  children,
}: AsyncBoundaryProps) {
  if (loading) {
    return <LoadingState />;
  }

  if (error) {
    return (
      <ErrorState
        description={error.message}
        onRetry={onRetry}
      />
    );
  }

  return children;
}
