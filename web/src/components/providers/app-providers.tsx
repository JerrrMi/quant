"use client";

import { ThemeProvider } from "next-themes";
import type { ReactNode } from "react";
import { Toaster } from "sonner";

import { ConfirmProvider } from "@/components/feedback/confirm-provider";
import { AuthProvider } from "@/features/auth/auth-context";
import { TooltipProvider } from "@/components/ui/tooltip";

export function AppProviders({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider attribute="class" defaultTheme="system" enableSystem>
      <TooltipProvider delayDuration={200}>
        <ConfirmProvider>
          <AuthProvider>{children}</AuthProvider>
          <Toaster richColors closeButton position="top-center" />
        </ConfirmProvider>
      </TooltipProvider>
    </ThemeProvider>
  );
}
