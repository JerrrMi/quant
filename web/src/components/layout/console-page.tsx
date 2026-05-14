import type { ReactNode } from "react";

import { PageHeader } from "@/components/layout/page-header";

type ConsolePageProps = {
  title: string;
  description?: string;
  actions?: ReactNode;
  children: ReactNode;
};

export function ConsolePage({
  title,
  description,
  actions,
  children,
}: ConsolePageProps) {
  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-6 px-4 py-6 lg:px-8 lg:py-8">
      <PageHeader title={title} description={description} actions={actions} />
      <section className="space-y-4">{children}</section>
    </div>
  );
}
