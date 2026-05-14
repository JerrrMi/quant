import { Wallet } from "lucide-react";

import { ConsolePage } from "@/components/layout/console-page";
import { EmptyState } from "@/components/feedback/empty-state";

export default function AccountsPage() {
  return (
    <ConsolePage
      title="Accounts"
      description="账户聚合与资金视图 · 不包含交易所密钥字段。"
    >
      <EmptyState
        icon={Wallet}
        title="账户总览占位"
        description="映射 AccountSummaryView；风险限额与保证金口径在后端统一。"
      />
    </ConsolePage>
  );
}
