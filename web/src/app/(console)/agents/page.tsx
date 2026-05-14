import { Bot } from "lucide-react";

import { ConsolePage } from "@/components/layout/console-page";
import { EmptyState } from "@/components/feedback/empty-state";

export default function AgentsPage() {
  return (
    <ConsolePage
      title="Agents"
      description="编排执行节点 · 交易所密钥仅在 Agent 进程持有。"
    >
      <EmptyState
        icon={Bot}
        title="尚未接入 Agents 列表"
        description="后端就绪后通过适配层映射 AgentSummaryView，并在此渲染队列与健康检查。"
      />
    </ConsolePage>
  );
}
