"use client";

import { useEffect, type ReactNode } from "react";
import { useRouter } from "next/navigation";

import { Skeleton } from "@/components/ui/skeleton";
import { useAuth } from "@/features/auth/auth-context";
import { publicEnv } from "@/lib/env";

export function ConsoleAuthGate({ children }: { children: ReactNode }) {
  const { user, loading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (loading || publicEnv.devMockAuth) return;
    if (!user) {
      router.replace("/login");
    }
  }, [user, loading, router]);

  if (loading) {
    return (
      <div className="space-y-4 p-6">
        <Skeleton className="h-8 w-56" />
        <Skeleton className="h-32 w-full max-w-3xl" />
        <Skeleton className="h-64 w-full max-w-5xl" />
      </div>
    );
  }

  if (!user && !publicEnv.devMockAuth) {
    return null;
  }

  return <>{children}</>;
}
