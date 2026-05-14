import { ConsolePage } from "@/components/layout/console-page";
import { DashboardOverview } from "@/features/dashboard/dashboard-overview";

export default function DashboardPage() {
  return (
    <ConsolePage
      title="Dashboard"
      description="交易系统控制台入口 · 聚合 Agents、策略实例与风控信号的占位视图。"
    >
      <DashboardOverview />
    </ConsolePage>
  );
}
