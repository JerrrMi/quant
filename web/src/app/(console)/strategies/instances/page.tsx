import { Boxes } from "lucide-react";

import { ConsolePage } from "@/components/layout/console-page";
import { EmptyState } from "@/components/feedback/empty-state";

export default function StrategyInstancesPage() {
  return (
    <ConsolePage
      title="策略实例"
      description="运行中的参数化实例 · 展示符号、风控别名与调度分区。"
    >
      <EmptyState
        icon={Boxes}
        title="实例列表占位"
        description="对齐 StrategyInstanceView；实例状态标签统一使用 StatusTag 映射。"
      />
    </ConsolePage>
  );
}
