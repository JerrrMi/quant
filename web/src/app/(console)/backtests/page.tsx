"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  FlaskConical,
  GitCompare,
  Loader2,
  Pause,
  Play,
  RefreshCw,
  Square,
} from "lucide-react";
import { toast } from "sonner";

import {
  backtestJobAction,
  createBacktestJob,
  fetchBacktestJobs,
} from "@/api/backtests";
import {
  fetchInstanceDetail,
  fetchInstances,
} from "@/api/instances";
import { fetchTemplates } from "@/api/templates";
import { useConfirm } from "@/components/feedback/confirm-provider";
import { ConsolePage } from "@/components/layout/console-page";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { formatDateTime } from "@/lib/format-trading";
import type {
  BacktestJobListItemDTO,
  BacktestJobStatus,
  BacktestRequestDTO,
} from "@/types/backtests";
import type { StrategyInstanceRowDTO } from "@/types/strategies";
import type { StrategyTemplateDTO } from "@/types/strategies";

const DEFAULT_FORM: BacktestRequestDTO = {
  source_kind: "template",
  template_id: 0,
  instance_id: 0,
  symbol: "BTCUSDT",
  market_kind: "futures",
  data_provider: "file",
  data_path: "./data/bars",
  window_start: "",
  window_end: "",
  warmup_bars: 50,
  bar_stride: 1,
  initial_quote: "100000",
  currency: "USDT",
  maker_bps: 2,
  taker_bps: 5,
  use_taker_fees: true,
  slippage_bps: 1,
  funding_bps_per_day: 0,
  lppl_enabled: false,
  lppl_bubble_metric_01: 0,
  lppl_job_id: "",
  failure_rate: 0,
  rng_seed: 42,
};

function statusBadgeClass(s: BacktestJobStatus): string {
  switch (s) {
    case "pending":
      return "border-amber-500/40 bg-amber-500/10 text-amber-900 dark:text-amber-100";
    case "running":
      return "border-sky-500/40 bg-sky-500/10 text-sky-900 dark:text-sky-100";
    case "finished":
      return "border-emerald-500/40 bg-emerald-500/10 text-emerald-900 dark:text-emerald-100";
    case "failed":
      return "border-destructive/40 bg-destructive/10 text-destructive";
    case "cancelled":
      return "border-muted-foreground/40 bg-muted text-muted-foreground";
    default:
      return "border-border bg-secondary";
  }
}

function statusLabel(s: BacktestJobStatus): string {
  const map: Record<BacktestJobStatus, string> = {
    pending: "排队中",
    running: "运行中",
    finished: "已完成",
    failed: "失败",
    cancelled: "已取消",
  };
  return map[s] ?? s;
}

function FieldHint({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <p className={`text-xs text-muted-foreground ${className ?? ""}`}>
      {children}
    </p>
  );
}

export default function BacktestsPage() {
  const confirm = useConfirm();
  const [templates, setTemplates] = useState<StrategyTemplateDTO[]>([]);
  const [instances, setInstances] = useState<StrategyInstanceRowDTO[]>([]);
  const [jobs, setJobs] = useState<BacktestJobListItemDTO[]>([]);
  const [loadErr, setLoadErr] = useState<string | null>(null);
  const [jobsLoading, setJobsLoading] = useState(true);
  const [form, setForm] = useState<BacktestRequestDTO>(() => ({
    ...DEFAULT_FORM,
  }));
  const [submitting, setSubmitting] = useState(false);
  const [comparePick, setComparePick] = useState<number[]>([]);

  const selectedTpl = useMemo(
    () => templates.find((t) => t.id === form.template_id),
    [templates, form.template_id],
  );

  const loadJobs = useCallback(async () => {
    try {
      const res = await fetchBacktestJobs();
      setJobs(res.jobs);
      setLoadErr(null);
    } catch (e) {
      setLoadErr(e instanceof Error ? e.message : "加载任务失败");
    } finally {
      setJobsLoading(false);
    }
  }, []);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      (async () => {
        try {
          const [tplRes, instRes] = await Promise.all([
            fetchTemplates(),
            fetchInstances(),
          ]);
          setTemplates(tplRes.templates);
          setInstances(instRes.instances);
        } catch (e) {
          setLoadErr(e instanceof Error ? e.message : "加载模板/实例失败");
        }
      })();
    }, 0);
    return () => window.clearTimeout(timer);
  }, []);

  useEffect(() => {
    if (templates.length === 0) return;
    const timer = window.setTimeout(() => {
      setForm((f) => {
        if (f.source_kind !== "template" || f.template_id !== 0) return f;
        return { ...f, template_id: templates[0].id };
      });
    }, 0);
    return () => window.clearTimeout(timer);
  }, [templates]);

  useEffect(() => {
    const timer = window.setTimeout(() => void loadJobs(), 0);
    return () => window.clearTimeout(timer);
  }, [loadJobs]);

  const hasRunning = jobs.some((j) => j.status === "running");

  useEffect(() => {
    if (!hasRunning) return;
    const t = window.setInterval(() => void loadJobs(), 2500);
    return () => window.clearInterval(t);
  }, [hasRunning, loadJobs]);

  useEffect(() => {
    if (form.source_kind !== "instance" || !form.instance_id) return;
    let cancelled = false;
    (async () => {
      try {
        const d = await fetchInstanceDetail(form.instance_id);
        if (cancelled) return;
        const params =
          typeof d.instance_params_json === "object" &&
          d.instance_params_json !== null
            ? (d.instance_params_json as Record<string, unknown>)
            : {};
        const cap =
          typeof params.capital_quote === "string"
            ? params.capital_quote
            : typeof params.capital_quote === "number"
              ? String(params.capital_quote)
              : null;
        setForm((f) => ({
          ...f,
          template_id: d.template_id,
          symbol: d.symbol || f.symbol,
          market_kind:
            d.market_kind === "spot" || d.market_kind === "futures"
              ? d.market_kind
              : f.market_kind,
          initial_quote: cap ?? f.initial_quote,
        }));
      } catch {
        /* 实例缺失时保持表单 */
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [form.source_kind, form.instance_id]);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (form.source_kind === "template" && !form.template_id) {
      toast.error("请选择策略模板");
      return;
    }
    if (form.source_kind === "instance" && !form.instance_id) {
      toast.error("请选择策略实例");
      return;
    }
    const tpl =
      form.source_kind === "template"
        ? templates.find((t) => t.id === form.template_id)
        : templates.find((t) => t.id === form.template_id);
    if (tpl && !tpl.allow_backtest) {
      toast.error("所选模板不允许回测");
      return;
    }
    if (selectedTpl && selectedTpl.markets.length > 0) {
      if (!selectedTpl.markets.includes(form.market_kind)) {
        toast.error("市场类型与模板不匹配");
        return;
      }
    }

    const ok = await confirm({
      title: "开始回测？",
      description:
        "将使用当前表单参数在服务端异步执行回放；运行期间可暂停或终止任务。",
      confirmLabel: "开始",
    });
    if (!ok) return;

    const body: BacktestRequestDTO = {
      ...form,
      external_features: form.lppl_enabled ? { lppl: true } : undefined,
    };

    setSubmitting(true);
    try {
      const res = await createBacktestJob(body);
      toast.success(`任务 #${res.id} 已创建`);
      await loadJobs();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "创建失败");
    } finally {
      setSubmitting(false);
    }
  }

  async function doPause(job: BacktestJobListItemDTO) {
    const confirmed = await confirm({
      title: "暂停回测？",
      description: `任务 #${job.id} 将收到取消信号；未完成的部分不会写入最终结果（状态记为已取消）。`,
      confirmLabel: "暂停",
      destructive: true,
    });
    if (!confirmed) return;
    try {
      await backtestJobAction(job.id, "pause");
      toast.message("已请求暂停");
      await loadJobs();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "操作失败");
    }
  }

  async function doTerminate(job: BacktestJobListItemDTO) {
    const confirmed = await confirm({
      title: "终止回测？",
      description: `终止任务 #${job.id}，等价于取消正在运行的回放。`,
      confirmLabel: "终止",
      destructive: true,
    });
    if (!confirmed) return;
    try {
      await backtestJobAction(job.id, "terminate");
      toast.message("已请求终止");
      await loadJobs();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "操作失败");
    }
  }

  async function doRerun(job: BacktestJobListItemDTO) {
    const confirmed = await confirm({
      title: "重新运行？",
      description: `将使用任务 #${job.id} 保存的参数快照创建新任务。`,
      confirmLabel: "重新运行",
    });
    if (!confirmed) return;
    try {
      const res = await backtestJobAction(job.id, "rerun");
      if ("id" in res && typeof res.id === "number") {
        toast.success(`新任务 #${res.id} 已创建`);
      }
      await loadJobs();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "操作失败");
    }
  }

  function toggleCompare(id: number) {
    setComparePick((prev) => {
      if (prev.includes(id)) return prev.filter((x) => x !== id);
      if (prev.length >= 2) return [prev[1], id];
      return [...prev, id];
    });
  }

  return (
    <ConsolePage
      title="Backtests"
      description="离线回放与 KPI 汇总：参数语义与策略实例创建表单一致（标的、市场、资金、成本模型）。"
      actions={
        <Button variant="outline" size="sm" asChild>
          <Link href="/strategies/instances/new">创建策略实例</Link>
        </Button>
      }
    >
      {hasRunning ? (
        <div className="mb-4 flex items-center gap-2 rounded-lg border border-sky-500/30 bg-sky-500/5 px-4 py-3 text-sm text-sky-900 dark:text-sky-100">
          <Loader2 className="h-4 w-4 animate-spin shrink-0" />
          <span>
            有回测任务正在运行，页面会自动刷新列表与进度；详情页提供更细日志。
          </span>
        </div>
      ) : null}

      {loadErr ? (
        <Card className="mb-6 border-destructive/40">
          <CardHeader>
            <CardTitle className="text-base text-destructive">加载问题</CardTitle>
            <CardDescription>{loadErr}</CardDescription>
          </CardHeader>
        </Card>
      ) : null}

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.1fr)]">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <FlaskConical className="h-5 w-5" />
              新建回测
            </CardTitle>
            <CardDescription>
              选择模板或实例作为参数来源；模板{" "}
              <code className="text-xs">template_defaults</code> 会合并进模型超参。
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={onSubmit} className="space-y-5">
              <div className="space-y-2">
                <Label>参数来源</Label>
                <div className="flex flex-wrap gap-4">
                  <label className="flex cursor-pointer items-center gap-2 text-sm">
                    <input
                      type="radio"
                      name="sk"
                      checked={form.source_kind === "template"}
                      onChange={() =>
                        setForm((f) => ({ ...f, source_kind: "template" }))
                      }
                    />
                    策略模板
                  </label>
                  <label className="flex cursor-pointer items-center gap-2 text-sm">
                    <input
                      type="radio"
                      name="sk"
                      checked={form.source_kind === "instance"}
                      onChange={() =>
                        setForm((f) => ({ ...f, source_kind: "instance" }))
                      }
                    />
                    策略实例（对齐实例上的标的与资金快照）
                  </label>
                </div>
                <FieldHint>
                  选实例时会在右侧自动拉取实例详情，用于预填标的与资金等字段。
                </FieldHint>
              </div>

              {form.source_kind === "template" ? (
                <div className="space-y-2">
                  <Label htmlFor="tpl">策略模板</Label>
                  <select
                    id="tpl"
                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                    value={form.template_id || ""}
                    onChange={(e) =>
                      setForm((f) => ({
                        ...f,
                        template_id: Number(e.target.value),
                      }))
                    }
                  >
                    <option value="">选择模板…</option>
                    {templates.map((t) => (
                      <option key={t.id} value={t.id}>
                        {t.name}
                        {!t.allow_backtest ? "（未开放回测）" : ""}
                      </option>
                    ))}
                  </select>
                  <FieldHint>
                    仅展示目录模板；若模板禁止回测，提交时会被后端拒绝。
                  </FieldHint>
                </div>
              ) : (
                <div className="space-y-2">
                  <Label htmlFor="inst">策略实例</Label>
                  <select
                    id="inst"
                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                    value={form.instance_id || ""}
                    onChange={(e) =>
                      setForm((f) => ({
                        ...f,
                        instance_id: Number(e.target.value),
                      }))
                    }
                  >
                    <option value="">选择实例…</option>
                    {instances.map((i) => (
                      <option key={i.id} value={i.id}>
                        #{i.id} {i.display_name} · {i.symbol}
                      </option>
                    ))}
                  </select>
                </div>
              )}

              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="sym">标的</Label>
                  <Input
                    id="sym"
                    value={form.symbol}
                    onChange={(e) =>
                      setForm((f) => ({
                        ...f,
                        symbol: e.target.value.toUpperCase(),
                      }))
                    }
                  />
                  <FieldHint>与历史数据文件名一致，如 BTCUSDT。</FieldHint>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="mk">市场类型</Label>
                  <select
                    id="mk"
                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                    value={form.market_kind}
                    onChange={(e) =>
                      setForm((f) => ({
                        ...f,
                        market_kind: e.target.value as "spot" | "futures",
                      }))
                    }
                  >
                    <option value="futures">futures（合约）</option>
                    <option value="spot">spot（现货）</option>
                  </select>
                  <FieldHint>
                    与模板 markets 字段对齐；影响展示与后续扩展，不参与 Step 纯函数。
                  </FieldHint>
                </div>
              </div>

              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="dp">历史数据源类型</Label>
                  <Input
                    id="dp"
                    value={form.data_provider}
                    onChange={(e) =>
                      setForm((f) => ({ ...f, data_provider: e.target.value }))
                    }
                  />
                  <FieldHint>
                    与后端{" "}
                    <code className="text-xs">BacktestConfig.data.provider</code>{" "}
                    一致；当前实现以 file 为主。
                  </FieldHint>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="path">数据路径</Label>
                  <Input
                    id="path"
                    value={form.data_path}
                    onChange={(e) =>
                      setForm((f) => ({ ...f, data_path: e.target.value }))
                    }
                  />
                  <FieldHint>
                    CSV 根目录，内含 {"{SYMBOL}.csv"}，见 configs/backtest.yaml。
                  </FieldHint>
                </div>
              </div>

              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="ws">时间区间起点</Label>
                  <Input
                    id="ws"
                    placeholder="留空表示从头"
                    value={form.window_start}
                    onChange={(e) =>
                      setForm((f) => ({ ...f, window_start: e.target.value }))
                    }
                  />
                  <FieldHint>RFC3339 或后端支持的日期串；空=不截断起点。</FieldHint>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="we">时间区间终点</Label>
                  <Input
                    id="we"
                    placeholder="留空表示到末"
                    value={form.window_end}
                    onChange={(e) =>
                      setForm((f) => ({ ...f, window_end: e.target.value }))
                    }
                  />
                  <FieldHint>半开区间 [start,end) 由回放模块解析。</FieldHint>
                </div>
              </div>

              <div className="grid gap-4 sm:grid-cols-3">
                <div className="space-y-2">
                  <Label htmlFor="cap">初始资金</Label>
                  <Input
                    id="cap"
                    value={form.initial_quote}
                    onChange={(e) =>
                      setForm((f) => ({
                        ...f,
                        initial_quote: e.target.value,
                      }))
                    }
                  />
                  <FieldHint>报价货币数量字符串，写入 capital.initial_quote。</FieldHint>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="cur">币种</Label>
                  <Input
                    id="cur"
                    value={form.currency}
                    onChange={(e) =>
                      setForm((f) => ({ ...f, currency: e.target.value }))
                    }
                  />
                  <FieldHint>如 USDT，仅记账字段。</FieldHint>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="warm">预热 K 线数</Label>
                  <Input
                    id="warm"
                    type="number"
                    min={0}
                    value={form.warmup_bars}
                    onChange={(e) =>
                      setForm((f) => ({
                        ...f,
                        warmup_bars: Number(e.target.value),
                      }))
                    }
                  />
                  <FieldHint>正式统计前额外行走的根数，需 ≥ 特征窗口。</FieldHint>
                </div>
              </div>

              <div className="grid gap-4 sm:grid-cols-3">
                <div className="space-y-2">
                  <Label htmlFor="stride">回放步进（每 N 根 K 线）</Label>
                  <Input
                    id="stride"
                    type="number"
                    min={1}
                    value={form.bar_stride}
                    onChange={(e) =>
                      setForm((f) => ({
                        ...f,
                        bar_stride: Math.max(1, Number(e.target.value) || 1),
                      }))
                    }
                  />
                  <FieldHint>
                    对齐{" "}
                    <code className="text-xs">replay.bar_stride</code>：对载入后的序列降采样。
                  </FieldHint>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="maker">Maker 手续费（bps）</Label>
                  <Input
                    id="maker"
                    type="number"
                    min={0}
                    step="0.01"
                    value={form.maker_bps}
                    onChange={(e) =>
                      setForm((f) => ({
                        ...f,
                        maker_bps: Number(e.target.value),
                      }))
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="taker">Taker 手续费（bps）</Label>
                  <Input
                    id="taker"
                    type="number"
                    min={0}
                    step="0.01"
                    value={form.taker_bps}
                    onChange={(e) =>
                      setForm((f) => ({
                        ...f,
                        taker_bps: Number(e.target.value),
                      }))
                    }
                  />
                </div>
              </div>

              <div className="flex flex-wrap items-center gap-6">
                <div className="flex items-center gap-2">
                  <Switch
                    id="use-taker"
                    checked={form.use_taker_fees}
                    onCheckedChange={(v: boolean) =>
                      setForm((f) => ({ ...f, use_taker_fees: v }))
                    }
                  />
                  <Label htmlFor="use-taker" className="font-normal">
                    使用 Taker 费率
                  </Label>
                </div>
                <FieldHint className="basis-full sm:basis-auto sm:inline">
                  关闭则按 Maker 计费（对齐 simulation.use_taker_fees）。
                </FieldHint>
              </div>

              <div className="grid gap-4 sm:grid-cols-3">
                <div className="space-y-2">
                  <Label htmlFor="slp">滑点（bps）</Label>
                  <Input
                    id="slp"
                    type="number"
                    min={0}
                    value={form.slippage_bps}
                    onChange={(e) =>
                      setForm((f) => ({
                        ...f,
                        slippage_bps: Number(e.target.value),
                      }))
                    }
                  />
                  <FieldHint>成交价的额外不利偏移，简化为基点。</FieldHint>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="fund">资金费率（bps/日）</Label>
                  <Input
                    id="fund"
                    type="number"
                    min={0}
                    step="0.01"
                    value={form.funding_bps_per_day}
                    onChange={(e) =>
                      setForm((f) => ({
                        ...f,
                        funding_bps_per_day: Number(e.target.value),
                      }))
                    }
                  />
                  <FieldHint>按持仓名义在 bar 区间上摊销，对齐合约资金费草稿模型。</FieldHint>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="fail">命令失败率（仿真）</Label>
                  <Input
                    id="fail"
                    type="number"
                    min={0}
                    max={1}
                    step="0.01"
                    value={form.failure_rate}
                    onChange={(e) =>
                      setForm((f) => ({
                        ...f,
                        failure_rate: Number(e.target.value),
                      }))
                    }
                  />
                  <FieldHint>[0,1] 之间，用于近似拒单/网络失败。</FieldHint>
                </div>
              </div>

              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="seed">RNG 种子</Label>
                  <Input
                    id="seed"
                    type="number"
                    value={form.rng_seed}
                    onChange={(e) =>
                      setForm((f) => ({
                        ...f,
                        rng_seed: Number(e.target.value),
                      }))
                    }
                  />
                  <FieldHint>
                    控制仿真随机性；0 表示由服务端选择非确定性种子。
                  </FieldHint>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="lppl_id">LPPL job_id（可选）</Label>
                  <Input
                    id="lppl_id"
                    value={form.lppl_job_id}
                    onChange={(e) =>
                      setForm((f) => ({ ...f, lppl_job_id: e.target.value }))
                    }
                  />
                  <FieldHint>传入特征注入模块的 job 标识占位。</FieldHint>
                </div>
              </div>

              <div className="flex flex-col gap-3 rounded-md border p-4">
                <div className="flex items-center justify-between gap-4">
                  <div>
                    <div className="font-medium">外部特征 · LPPL</div>
                    <FieldHint>
                      打开后写入{" "}
                      <code className="text-xs">lppl.enabled</code> 与{" "}
                      <code className="text-xs">external_features.lppl</code>。
                    </FieldHint>
                  </div>
                  <Switch
                    checked={form.lppl_enabled}
                    onCheckedChange={(v: boolean) =>
                      setForm((f) => ({ ...f, lppl_enabled: v }))
                    }
                  />
                </div>
                {form.lppl_enabled ? (
                  <div className="space-y-2">
                    <Label htmlFor="bub">泡沫度量（0–1）</Label>
                    <Input
                      id="bub"
                      type="number"
                      min={0}
                      max={1}
                      step="0.01"
                      value={form.lppl_bubble_metric_01}
                      onChange={(e) =>
                        setForm((f) => ({
                          ...f,
                          lppl_bubble_metric_01: Number(e.target.value),
                        }))
                      }
                    />
                  </div>
                ) : null}
              </div>

              <div className="flex flex-wrap gap-2">
                <Button type="submit" disabled={submitting}>
                  {submitting ? (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  ) : (
                    <Play className="mr-2 h-4 w-4" />
                  )}
                  开始回测
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setForm({ ...DEFAULT_FORM })}
                >
                  重置默认
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row flex-wrap items-start justify-between gap-2">
            <div>
              <CardTitle className="text-lg">回测任务</CardTitle>
              <CardDescription>
                状态与后端字段一一对应；运行中自动轮询。可选择两项进行对比。
              </CardDescription>
            </div>
            <div className="flex flex-wrap gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={comparePick.length !== 2}
                asChild={comparePick.length === 2}
              >
                {comparePick.length === 2 ? (
                  <Link
                    href={`/backtests/compare?left=${comparePick[0]}&right=${comparePick[1]}`}
                  >
                    <GitCompare className="mr-2 h-4 w-4" />
                    对比选中
                  </Link>
                ) : (
                  <span>
                    <GitCompare className="mr-2 h-4 w-4" />
                    对比选中
                  </span>
                )}
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => void loadJobs()}
                disabled={jobsLoading}
              >
                <RefreshCw className="mr-2 h-4 w-4" />
                刷新
              </Button>
            </div>
          </CardHeader>
          <CardContent className="overflow-x-auto">
            {comparePick.length > 0 ? (
              <p className="mb-2 text-xs text-muted-foreground">
                已选任务 ID：{comparePick.join("、")}
              </p>
            ) : null}
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[52px]">比</TableHead>
                  <TableHead>ID</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>模板</TableHead>
                  <TableHead>标的</TableHead>
                  <TableHead>区间</TableHead>
                  <TableHead>创建</TableHead>
                  <TableHead>时长</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {jobs.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={9} className="text-center text-sm text-muted-foreground">
                      {jobsLoading ? "加载中…" : "暂无任务"}
                    </TableCell>
                  </TableRow>
                ) : (
                  jobs.map((j) => (
                    <TableRow key={j.id}>
                      <TableCell>
                        <input
                          type="checkbox"
                          checked={comparePick.includes(j.id)}
                          onChange={() => toggleCompare(j.id)}
                          aria-label={`对比任务 ${j.id}`}
                        />
                      </TableCell>
                      <TableCell>
                        <Link
                          href={`/backtests/${j.id}`}
                          className="font-medium text-primary underline-offset-4 hover:underline"
                        >
                          #{j.id}
                        </Link>
                      </TableCell>
                      <TableCell>
                        <span
                          className={`inline-flex rounded-full border px-2 py-0.5 text-[11px] font-medium ${statusBadgeClass(j.status)}`}
                        >
                          {statusLabel(j.status)}
                        </span>
                        {j.status === "running" && j.progress ? (
                          <div className="mt-1 text-[10px] text-muted-foreground">
                            {j.progress.done}/{j.progress.total} (
                            {(j.progress.pct_01 * 100).toFixed(0)}
                            %)
                          </div>
                        ) : null}
                      </TableCell>
                      <TableCell className="max-w-[140px] truncate text-xs">
                        {j.template_name || "—"}
                      </TableCell>
                      <TableCell className="text-xs">{j.symbol}</TableCell>
                      <TableCell className="max-w-[120px] truncate text-xs">
                        {j.window_start || "—"} → {j.window_end || "—"}
                      </TableCell>
                      <TableCell className="whitespace-nowrap text-xs">
                        {formatDateTime(j.created_at)}
                      </TableCell>
                      <TableCell className="text-xs">
                        {j.duration_ms != null
                          ? `${(j.duration_ms / 1000).toFixed(1)}s`
                          : "—"}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex flex-wrap justify-end gap-1">
                          <Button size="sm" variant="outline" asChild>
                            <Link href={`/backtests/${j.id}`}>详情</Link>
                          </Button>
                          <Button
                            size="sm"
                            variant="ghost"
                            disabled={j.status !== "running"}
                            onClick={() => void doPause(j)}
                            title="暂停"
                          >
                            <Pause className="h-4 w-4" />
                          </Button>
                          <Button
                            size="sm"
                            variant="ghost"
                            disabled={j.status !== "running"}
                            onClick={() => void doTerminate(j)}
                            title="终止"
                          >
                            <Square className="h-4 w-4" />
                          </Button>
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => void doRerun(j)}
                            title="重新运行"
                          >
                            <RefreshCw className="h-4 w-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </div>
    </ConsolePage>
  );
}
