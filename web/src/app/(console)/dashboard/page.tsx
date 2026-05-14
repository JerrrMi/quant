import { ConsolePage } from "@/components/layout/console-page";
import { DashboardOverview } from "@/features/dashboard/dashboard-overview";

export default function DashboardPage() {
  return (
    <ConsolePage
      title="Dashboard"
      description="系统总览：SaaS / Agent / 策略实例与连接健康。数据由 BFF 或 Mock 轮询刷新。"
    >
      <DashboardOverview />
    </ConsolePage>
  );
}
