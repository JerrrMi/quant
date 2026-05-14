"use client";

import Link from "next/link";

import type { ConsoleCommandDTO } from "@/types/commands";
import { formatDateTime } from "@/lib/format-trading";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

function fmt(ts: string) {
  return ts ? formatDateTime(ts) : "—";
}

export function CommandStreamTable({
  commands,
  dense,
}: {
  commands: ConsoleCommandDTO[];
  dense?: boolean;
}) {
  if (commands.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        暂无指令记录（调度产出命令后将出现在此）。
      </p>
    );
  }
  return (
    <div className="rounded-lg border border-border bg-card overflow-x-auto">
      <Table className={dense ? "text-xs" : "text-sm"}>
        <TableHeader>
          <TableRow>
            <TableHead>command_id</TableHead>
            <TableHead>instance</TableHead>
            <TableHead>symbol</TableHead>
            <TableHead>intent</TableHead>
            <TableHead>status</TableHead>
            <TableHead>下发</TableHead>
            <TableHead>ack</TableHead>
            <TableHead>状态更新</TableHead>
            <TableHead className="text-right">跳转</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {commands.map((c) => (
            <TableRow key={c.command_id}>
              <TableCell className="max-w-[120px] font-mono text-[11px]">
                {c.command_id}
              </TableCell>
              <TableCell className="font-mono text-[11px]">
                <Link
                  className="text-primary hover:underline"
                  href={`/strategies/instances/${c.instance_id}`}
                >
                  #{c.instance_id}
                </Link>
              </TableCell>
              <TableCell>
                <code>{c.symbol}</code>
              </TableCell>
              <TableCell className="max-w-[200px] truncate" title={c.intent}>
                {c.intent || `${c.kind}`}
              </TableCell>
              <TableCell>{c.status}</TableCell>
              <TableCell className="whitespace-nowrap text-muted-foreground">
                {fmt(c.dispatched_at || c.issued_at)}
              </TableCell>
              <TableCell className="whitespace-nowrap text-muted-foreground">
                {fmt(c.acked_at)}
              </TableCell>
              <TableCell className="whitespace-nowrap text-muted-foreground">
                {fmt(c.report_at)}
              </TableCell>
              <TableCell className="text-right">
                <div className="flex justify-end gap-2">
                  <Link
                    className="text-primary hover:underline"
                    href={`/commands?instance_id=${c.instance_id}`}
                  >
                    命令
                  </Link>
                  <Link
                    className="text-primary hover:underline"
                    href={`/logs?instance_id=${c.instance_id}`}
                  >
                    日志
                  </Link>
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
