import { FlaskConical } from "lucide-react";

import { ConsolePage } from "@/components/layout/console-page";
import { EmptyState } from "@/components/feedback/empty-state";

export default function BacktestsPage() {
  return (
    <ConsolePage
      title="Backtests"
      description="回放引擎结果与工作台入口 · 历史数据装载逻辑在后端 Backtest 模块。"
    >
      <EmptyState
        icon={FlaskConical}
        title="回测记录占位"
        description="视图模型 BacktestRunView；耗时任务请在适配层拆分分页与导出。"
      />
    </ConsolePage>
  );
}
