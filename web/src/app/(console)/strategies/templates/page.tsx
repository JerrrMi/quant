"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { Layers } from "lucide-react";

import { fetchTemplates } from "@/api/templates";
import { ConsolePage } from "@/components/layout/console-page";
import { EmptyState } from "@/components/feedback/empty-state";
import { ErrorState } from "@/components/feedback/error-state";
import { LoadingState } from "@/components/feedback/loading-state";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import type { StrategyTemplateDTO } from "@/types/strategies";

const POLL_MS = 5000;

export default function StrategyTemplatesPage() {
  const [rows, setRows] = useState<StrategyTemplateDTO[] | null>(null);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const data = await fetchTemplates();
        if (!cancelled) {
          setRows(data.templates);
          setErr(null);
        }
      } catch (e) {
        if (!cancelled) {
          setErr(e instanceof Error ? e.message : "加载失败");
        }
      }
    }
    const boot = setTimeout(() => void load(), 0);
    const t = setInterval(load, POLL_MS);
    return () => {
      cancelled = true;
      clearTimeout(boot);
      clearInterval(t);
    };
  }, []);

  return (
    <ConsolePage
      title="策略模板"
      description="模板来自后端目录库（IsCatalog）；新增模板写入数据库后此处会自动列出，无需前端改代码。"
    >
      {err ? (
        <ErrorState
          title="无法连接控制台 API"
          description={`${err} · 请确认 SaaS 已启动，且服务端已将 SAAS_CONSOLE_ORIGIN（默认 http://127.0.0.1:8080）指向控制面监听地址。`}
        />
      ) : null}

      {rows === null && !err ? <LoadingState label="加载模板目录…" /> : null}

      {rows && rows.length === 0 ? (
        <EmptyState
          icon={Layers}
          title="暂无目录模板"
          description="启动 SaaS 后会自动写入内置「最小做空山寨币」骨架模板；也可在库中插入新的 catalog 策略行。"
        />
      ) : null}

      {rows && rows.length > 0 ? (
        <div className="rounded-lg border border-border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>名称</TableHead>
                <TableHead>类型</TableHead>
                <TableHead>市场</TableHead>
                <TableHead>描述</TableHead>
                <TableHead>实盘</TableHead>
                <TableHead>回测</TableHead>
                <TableHead>更新时间</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((t) => (
                <TableRow key={t.id}>
                  <TableCell className="font-medium">{t.name}</TableCell>
                  <TableCell>
                    <code className="rounded bg-muted px-1.5 py-0.5 text-xs">
                      {t.kind}
                    </code>
                  </TableCell>
                  <TableCell className="space-x-1">
                    {t.markets.length === 0 ? (
                      <span className="text-muted-foreground">—</span>
                    ) : (
                      t.markets.map((m) => (
                        <Badge key={m} variant="secondary" className="text-[10px]">
                          {m}
                        </Badge>
                      ))
                    )}
                  </TableCell>
                  <TableCell className="max-w-[260px] text-xs leading-snug text-muted-foreground">
                    <span className="line-clamp-3">{t.description || "—"}</span>
                  </TableCell>
                  <TableCell>{t.allow_live ? "是" : "否"}</TableCell>
                  <TableCell>{t.allow_backtest ? "是" : "否"}</TableCell>
                  <TableCell className="text-muted-foreground text-xs">
                    {new Date(t.updated_at).toLocaleString()}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button variant="outline" size="sm" asChild>
                      <Link href={`/strategies/templates/${t.id}`}>详情</Link>
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          <p className="border-t border-border px-4 py-2 text-[11px] text-muted-foreground">
            完整模板 JSON 见详情页。
          </p>
        </div>
      ) : null}
    </ConsolePage>
  );
}
