import { Layers } from "lucide-react";

import { ConsolePage } from "@/components/layout/console-page";
import { EmptyState } from "@/components/feedback/empty-state";

export default function StrategyTemplatesPage() {
  return (
    <ConsolePage
      title="策略模板"
      description="沉淀可复用的策略片段与参数契约 · 策略核心函数保持纯 Step()。"
    >
      <EmptyState
        icon={Layers}
        title="模板目录占位"
        description="视图模型 StrategyTemplateView 位于 src/types/console.ts；接口适配请在 src/api/adapters 聚合。"
      />
    </ConsolePage>
  );
}
