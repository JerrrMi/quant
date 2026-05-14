"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { ArrowLeft } from "lucide-react";
import { useParams } from "next/navigation";

import { fetchTemplateDetail } from "@/api/templates";
import { ConsolePage } from "@/components/layout/console-page";
import { ErrorState } from "@/components/feedback/error-state";
import { LoadingState } from "@/components/feedback/loading-state";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import type { StrategyTemplateDetailDTO } from "@/types/strategies";

export default function StrategyTemplateDetailPage() {
  const params = useParams();
  const id = String(params.id ?? "");
  const [data, setData] = useState<StrategyTemplateDetailDTO | null>(null);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const d = await fetchTemplateDetail(id);
        if (!cancelled) {
          setData(d);
          setErr(null);
        }
      } catch (e) {
        if (!cancelled) {
          setErr(e instanceof Error ? e.message : "加载失败");
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [id]);

  return (
    <ConsolePage
      title={data?.name ?? "模板详情"}
      description="模板参数来自后端 config_json（默认快照）；策略逻辑仍在服务端策略包，前端不做计算。"
      actions={
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" size="sm" asChild>
            <Link href="/strategies/templates">
              <ArrowLeft className="mr-2 h-4 w-4" />
              返回列表
            </Link>
          </Button>
          {data ? (
            <Button size="sm" asChild>
              <Link href={`/strategies/instances/new?template_id=${data.id}`}>
                创建实例
              </Link>
            </Button>
          ) : null}
        </div>
      }
    >
      {err ? <ErrorState description={err} /> : null}
      {!data && !err ? <LoadingState label="加载模板…" /> : null}
      {data ? (
        <div className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>元数据</CardTitle>
              <CardDescription>目录展示字段（IsCatalog）</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              <div className="flex flex-wrap gap-2">
                <span className="text-muted-foreground">类型</span>
                <code className="rounded bg-muted px-1.5 py-0.5 text-xs">
                  {data.kind}
                </code>
              </div>
              <div>
                <div className="text-muted-foreground">适用市场</div>
                <div className="mt-1 flex flex-wrap gap-1">
                  {data.markets.map((m) => (
                    <Badge key={m} variant="secondary">
                      {m}
                    </Badge>
                  ))}
                </div>
              </div>
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <div className="text-muted-foreground">可实盘</div>
                  <div>{data.allow_live ? "是" : "否"}</div>
                </div>
                <div>
                  <div className="text-muted-foreground">可回测</div>
                  <div>{data.allow_backtest ? "是" : "否"}</div>
                </div>
              </div>
              <div>
                <div className="text-muted-foreground">描述</div>
                <p className="mt-1 leading-relaxed">{data.description || "—"}</p>
              </div>
              <div className="text-xs text-muted-foreground">
                更新时间 {new Date(data.updated_at).toLocaleString()}
              </div>
            </CardContent>
          </Card>

          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle>模板参数（config_json）</CardTitle>
              <CardDescription>
                与运行实例的 ParamsJSON 区分：此处为模板默认/契约说明 JSON，由后端保管。
              </CardDescription>
            </CardHeader>
            <CardContent>
              <pre className="max-h-[420px] overflow-auto rounded-md border border-border bg-muted/40 p-4 text-xs leading-relaxed">
                {JSON.stringify(data.config_json, null, 2)}
              </pre>
            </CardContent>
          </Card>
        </div>
      ) : null}
    </ConsolePage>
  );
}
