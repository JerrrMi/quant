import { ScrollText } from "lucide-react";

import { ConsolePage } from "@/components/layout/console-page";
import { EmptyState } from "@/components/feedback/empty-state";

export default function LogsPage() {
  return (
    <ConsolePage
      title="Logs"
      description="跨 Agents / SaaS / 风控管道的结构化日志占位。"
    >
      <EmptyState
        icon={ScrollText}
        title="日志流占位"
        description="接入后将映射 LogEntryView，可通过 SSE 订阅增量。"
      />
    </ConsolePage>
  );
}
