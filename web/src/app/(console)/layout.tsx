import type { ReactNode } from "react";

import { ConsoleAuthGate } from "@/components/auth/console-auth-gate";
import { ConsoleShell } from "@/components/layout/console-shell";

export default function ConsoleRootLayout({
  children,
}: {
  children: ReactNode;
}) {
  return (
    <ConsoleShell>
      <ConsoleAuthGate>{children}</ConsoleAuthGate>
    </ConsoleShell>
  );
}
