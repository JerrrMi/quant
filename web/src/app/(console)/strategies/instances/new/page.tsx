"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { z } from "zod";
import { ArrowLeft, ArrowRight, CheckCircle2 } from "lucide-react";

import { fetchTemplates } from "@/api/templates";
import { apiFetch } from "@/api/client";
import { API_PATHS } from "@/lib/api-paths";
import { createInstance } from "@/api/instances";
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
import type { AgentsListDTO } from "@/types/agents";
import type { StrategyTemplateDTO } from "@/types/strategies";

const STEPS = [
  "选择模板",
  "标的与市场",
  "资金与风控",
  "运行模式",
  "绑定 Agent",
  "确认创建",
] as const;

const payloadSchema = z.object({
  display_name: z
    .string()
    .min(2, "实例名至少 2 个字符")
    .max(128, "实例名过长"),
  strategy_id: z.number().int().positive("请选择模板"),
  symbol: z
    .string()
    .trim()
    .regex(/^[A-Za-z0-9]{3,40}$/, "标的格式应为字母数字组合，如 BTCUSDT"),
  market_kind: z.enum(["spot", "futures"]),
  capital_quote: z.string().min(1, "请填写分配资金"),
  max_leverage: z.coerce
    .number()
    .min(1, "杠杆 ≥ 1")
    .max(125, "杠杆上限 125"),
  max_position_fraction: z.coerce
    .number()
    .min(0.01, "仓位上限比例过小")
    .max(1, "仓位上限比例不能超过 1"),
  max_daily_loss_quote: z.string().min(1, "请填写单日最大亏损上限"),
  risk_summary: z.string().optional(),
  run_mode: z.enum(["backtest", "paper", "live"]),
  agent_key: z.string().min(2, "请选择或填写 Agent 标识"),
});

export default function NewStrategyInstancePage() {
  const sp = useSearchParams();
  const [step, setStep] = useState(0);
  const [templates, setTemplates] = useState<StrategyTemplateDTO[]>([]);
  const [agents, setAgents] = useState<AgentsListDTO["agents"]>([]);
  const [loadErr, setLoadErr] = useState<string | null>(null);

  const [form, setForm] = useState({
    display_name: "",
    strategy_id: 0,
    symbol: "",
    market_kind: "futures" as "spot" | "futures",
    capital_quote: "1000",
    max_leverage: 3,
    max_position_fraction: 0.2,
    max_daily_loss_quote: "200",
    risk_summary: "",
    run_mode: "paper" as "backtest" | "paper" | "live",
    agent_key: "",
    ackDanger: false as boolean,
  });

  useEffect(() => {
    const tid = sp.get("template_id");
    if (!tid) return;
    const n = parseInt(tid, 10);
    if (!Number.isNaN(n) && n > 0) {
      setForm((f) => ({ ...f, strategy_id: n }));
    }
  }, [sp]);

  const [fieldErr, setFieldErr] = useState<Record<string, string>>({});
  const [submitErr, setSubmitErr] = useState<string | null>(null);
  const [createdId, setCreatedId] = useState<number | null>(null);

  const selectedTpl = useMemo(
    () => templates.find((t) => t.id === form.strategy_id),
    [templates, form.strategy_id],
  );

  useEffect(() => {
    (async () => {
      try {
        const [tplRes, agentsRes] = await Promise.all([
          fetchTemplates(),
          apiFetch<AgentsListDTO>(API_PATHS.agents.list),
        ]);
        setTemplates(tplRes.templates);
        setAgents(agentsRes.agents);
        setLoadErr(null);
      } catch (e) {
        setLoadErr(e instanceof Error ? e.message : "加载失败");
      }
    })();
  }, []);

  function validateStep(i: number): boolean {
    const e: Record<string, string> = {};
    if (i === 0) {
      if (!form.strategy_id) e.strategy_id = "请选择一个模板";
      if (form.display_name.trim().length < 2) {
        e.display_name = "实例名至少 2 个字符";
      }
    }
    if (i === 1) {
      const sym = form.symbol.trim().toUpperCase();
      if (!/^[A-Z0-9]{3,40}$/.test(sym)) {
        e.symbol = "标的格式无效（示例：BTCUSDT）";
      }
      if (form.market_kind !== "spot" && form.market_kind !== "futures") {
        e.market_kind = "请选择市场类型";
      }
      if (selectedTpl && selectedTpl.markets.length > 0) {
        if (!selectedTpl.markets.includes(form.market_kind)) {
          e.market_kind = "该模板不支持所选市场";
        }
      }
    }
    if (i === 2) {
      if (!form.capital_quote.trim()) e.capital_quote = "必填";
      if (form.max_leverage < 1 || form.max_leverage > 125) {
        e.max_leverage = "杠杆需在 1–125 之间";
      }
      if (form.max_position_fraction <= 0 || form.max_position_fraction > 1) {
        e.max_position_fraction = "仓位上限需在 (0,1] 之间";
      }
      if (!form.max_daily_loss_quote.trim()) {
        e.max_daily_loss_quote = "必填";
      }
    }
    if (i === 3) {
      if (!selectedTpl) {
        e.run_mode = "模板未加载";
      } else {
        if (form.run_mode === "live" && !selectedTpl.allow_live) {
          e.run_mode = "该模板不允许实盘模式";
        }
        if (form.run_mode === "backtest" && !selectedTpl.allow_backtest) {
          e.run_mode = "该模板不允许回测模式";
        }
      }
    }
    if (i === 4) {
      if (!form.agent_key.trim()) e.agent_key = "请选择 Agent";
    }
    setFieldErr(e);
    return Object.keys(e).length === 0;
  }

  function next() {
    if (!validateStep(step)) return;
    setStep((s) => Math.min(s + 1, STEPS.length - 1));
  }

  function prev() {
    setStep((s) => Math.max(s - 1, 0));
  }

  async function submit() {
    setSubmitErr(null);
    const payload = {
      display_name: form.display_name.trim(),
      strategy_id: form.strategy_id,
      symbol: form.symbol.trim().toUpperCase(),
      market_kind: form.market_kind,
      capital_quote: form.capital_quote.trim(),
      max_leverage: form.max_leverage,
      max_position_fraction: form.max_position_fraction,
      max_daily_loss_quote: form.max_daily_loss_quote.trim(),
      risk_summary: form.risk_summary.trim(),
      run_mode: form.run_mode,
      agent_key: form.agent_key.trim(),
    };
    if (!form.ackDanger) {
      setFieldErr({ ackDanger: "请勾选确认后再提交" });
      return;
    }
    const parsed = payloadSchema.safeParse(payload);
    if (!parsed.success) {
      const fe: Record<string, string> = {};
      for (const issue of parsed.error.issues) {
        const k = issue.path[0];
        if (typeof k === "string") fe[k] = issue.message;
      }
      setFieldErr(fe);
      return;
    }
    try {
      const params = {
        capital_quote: parsed.data.capital_quote,
        max_leverage: parsed.data.max_leverage,
        max_position_fraction: parsed.data.max_position_fraction,
        max_daily_loss_quote: parsed.data.max_daily_loss_quote,
        risk_summary: parsed.data.risk_summary ?? "",
      };
      const res = await createInstance({
        display_name: parsed.data.display_name,
        strategy_id: parsed.data.strategy_id,
        symbol: parsed.data.symbol,
        market_kind: parsed.data.market_kind,
        run_mode: parsed.data.run_mode,
        agent_key: parsed.data.agent_key,
        params,
      });
      setCreatedId(res.id);
    } catch (err) {
      setSubmitErr(err instanceof Error ? err.message : "创建失败");
    }
  }

  return (
    <ConsolePage
      title="创建策略实例"
      description="分步向导：所有字段均带说明；提交写入后端数据库，可在实例列表实时刷新状态。"
      actions={
        <Button variant="outline" size="sm" asChild>
          <Link href="/strategies/instances">
            <ArrowLeft className="mr-2 h-4 w-4" />
            返回实例列表
          </Link>
        </Button>
      }
    >
      {loadErr ? (
        <Card className="border-destructive/40">
          <CardHeader>
            <CardTitle className="text-base text-destructive">
              依赖数据加载失败
            </CardTitle>
            <CardDescription>{loadErr}</CardDescription>
          </CardHeader>
        </Card>
      ) : null}

      {createdId !== null ? (
        <Card className="border-emerald-500/30 bg-emerald-500/5">
          <CardHeader className="flex flex-row items-start gap-3 space-y-0">
            <CheckCircle2 className="mt-0.5 h-6 w-6 text-emerald-600" />
            <div>
              <CardTitle className="text-lg">实例已创建</CardTitle>
              <CardDescription>
                编号 #{createdId}。实例默认处于「暂停」状态，请在实例详情页使用固定位置的「启动编排」按钮上线。
              </CardDescription>
            </div>
          </CardHeader>
          <CardContent className="flex flex-wrap gap-3">
            <Button asChild>
              <Link href={`/strategies/instances/${createdId}`}>前往实例详情</Link>
            </Button>
            <Button variant="outline" asChild>
              <Link href="/strategies/instances">返回列表</Link>
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-6 lg:grid-cols-[220px_1fr]">
          <Card className="h-fit">
            <CardHeader>
              <CardTitle className="text-base">步骤</CardTitle>
              <CardDescription>{STEPS[step]}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-2 text-sm">
              {STEPS.map((label, idx) => (
                <div
                  key={label}
                  className={`rounded-md px-2 py-1 ${
                    idx === step ? "bg-accent font-medium" : "text-muted-foreground"
                  }`}
                >
                  {idx + 1}. {label}
                </div>
              ))}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{STEPS[step]}</CardTitle>
              <CardDescription>
                {step === 0 &&
                  "模板列表来自后端 catalog；新增模板会自动出现在此。"}
                {step === 1 &&
                  "标的与市场决定调度读取的行情序列与下单路由（由执行端实现）。"}
                {step === 2 &&
                  "以下为实例级风控与经济约束快照，保存在 ParamsJSON，不影响策略纯函数。"}
                {step === 3 && "运行模式用于控制台分层治理；实盘需模板允许且 Agent 在线。"}
                {step === 4 &&
                  "AgentKey 必须与 Agent 进程握手时的 client_id 一致（公开标识，非 API Secret）。"}
                {step === 5 && "请核对摘要并勾选确认后再创建。"}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              {step === 0 ? (
                <div className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="display_name">实例显示名</Label>
                    <Input
                      id="display_name"
                      value={form.display_name}
                      onChange={(ev) =>
                        setForm((f) => ({ ...f, display_name: ev.target.value }))
                      }
                      placeholder="例如：PEPE 空单哨兵"
                    />
                    <p className="text-xs text-muted-foreground">
                      仅在控制台展示与审计索引使用，可与标的不同名。
                    </p>
                    {fieldErr.display_name ? (
                      <p className="text-xs text-destructive">{fieldErr.display_name}</p>
                    ) : null}
                  </div>
                  <div className="space-y-2">
                    <Label>模板</Label>
                    <div className="grid gap-2 md:grid-cols-2">
                      {templates.map((t) => (
                        <button
                          key={t.id}
                          type="button"
                          onClick={() =>
                            setForm((f) => ({ ...f, strategy_id: t.id }))
                          }
                          className={`rounded-lg border p-3 text-left text-sm transition-colors hover:bg-accent ${
                            form.strategy_id === t.id
                              ? "border-primary bg-accent"
                              : "border-border"
                          }`}
                        >
                          <div className="font-medium">{t.name}</div>
                          <div className="text-[11px] text-muted-foreground">
                            {t.kind} ·{" "}
                            {t.markets.length ? t.markets.join(", ") : "市场未标注"}
                          </div>
                        </button>
                      ))}
                    </div>
                    {fieldErr.strategy_id ? (
                      <p className="text-xs text-destructive">{fieldErr.strategy_id}</p>
                    ) : null}
                  </div>
                </div>
              ) : null}

              {step === 1 ? (
                <div className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="symbol">标的代码</Label>
                    <Input
                      id="symbol"
                      value={form.symbol}
                      onChange={(ev) =>
                        setForm((f) => ({
                          ...f,
                          symbol: ev.target.value.toUpperCase(),
                        }))
                      }
                      placeholder="例如：BTCUSDT"
                    />
                    <p className="text-xs text-muted-foreground">
                      使用交易所规范化符号；调度器按此读取行情窗口。
                    </p>
                    {fieldErr.symbol ? (
                      <p className="text-xs text-destructive">{fieldErr.symbol}</p>
                    ) : null}
                  </div>
                  <div className="space-y-2">
                    <Label>市场类型</Label>
                    <div className="flex gap-3">
                      {(["spot", "futures"] as const).map((m) => (
                        <button
                          key={m}
                          type="button"
                          className={`rounded-md border px-3 py-2 text-sm capitalize ${
                            form.market_kind === m
                              ? "border-primary bg-accent"
                              : "border-border"
                          }`}
                          onClick={() => setForm((f) => ({ ...f, market_kind: m }))}
                        >
                          {m}
                        </button>
                      ))}
                    </div>
                    <p className="text-xs text-muted-foreground">
                      必须与模板适用市场匹配；不匹配时提交会被后端拒绝。
                    </p>
                    {fieldErr.market_kind ? (
                      <p className="text-xs text-destructive">{fieldErr.market_kind}</p>
                    ) : null}
                  </div>
                </div>
              ) : null}

              {step === 2 ? (
                <div className="grid gap-4 md:grid-cols-2">
                  <div className="space-y-2">
                    <Label htmlFor="capital_quote">分配资金（报价货币）</Label>
                    <Input
                      id="capital_quote"
                      value={form.capital_quote}
                      onChange={(ev) =>
                        setForm((f) => ({ ...f, capital_quote: ev.target.value }))
                      }
                    />
                    <p className="text-xs text-muted-foreground">
                      字符串存储以避免精度丢失；执行端按账户真实余额校验。
                    </p>
                    {fieldErr.capital_quote ? (
                      <p className="text-xs text-destructive">{fieldErr.capital_quote}</p>
                    ) : null}
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="max_leverage">最大杠杆倍数</Label>
                    <Input
                      id="max_leverage"
                      type="number"
                      min={1}
                      max={125}
                      value={form.max_leverage}
                      onChange={(ev) =>
                        setForm((f) => ({
                          ...f,
                          max_leverage: Number(ev.target.value),
                        }))
                      }
                    />
                    <p className="text-xs text-muted-foreground">
                      控制台约束占位；真实杠杆以交易所设置为上限。
                    </p>
                    {fieldErr.max_leverage ? (
                      <p className="text-xs text-destructive">{fieldErr.max_leverage}</p>
                    ) : null}
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="max_position_fraction">最大仓位比例（0–1）</Label>
                    <Input
                      id="max_position_fraction"
                      type="number"
                      step="0.01"
                      min={0.01}
                      max={1}
                      value={form.max_position_fraction}
                      onChange={(ev) =>
                        setForm((f) => ({
                          ...f,
                          max_position_fraction: Number(ev.target.value),
                        }))
                      }
                    />
                    <p className="text-xs text-muted-foreground">
                      相对账户权益或分配资金的仓位上限（实例语义）。
                    </p>
                    {fieldErr.max_position_fraction ? (
                      <p className="text-xs text-destructive">
                        {fieldErr.max_position_fraction}
                      </p>
                    ) : null}
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="max_daily_loss_quote">单日最大亏损（报价货币）</Label>
                    <Input
                      id="max_daily_loss_quote"
                      value={form.max_daily_loss_quote}
                      onChange={(ev) =>
                        setForm((f) => ({
                          ...f,
                          max_daily_loss_quote: ev.target.value,
                        }))
                      }
                    />
                    {fieldErr.max_daily_loss_quote ? (
                      <p className="text-xs text-destructive">
                        {fieldErr.max_daily_loss_quote}
                      </p>
                    ) : null}
                  </div>
                  <div className="space-y-2 md:col-span-2">
                    <Label htmlFor="risk_summary">风险摘要（可选）</Label>
                    <Input
                      id="risk_summary"
                      value={form.risk_summary}
                      onChange={(ev) =>
                        setForm((f) => ({ ...f, risk_summary: ev.target.value }))
                      }
                      placeholder="一句话描述策略容忍度，展示在列表「风险」列"
                    />
                  </div>
                </div>
              ) : null}

              {step === 3 ? (
                <div className="space-y-3">
                  {(["backtest", "paper", "live"] as const).map((mode) => {
                    const disabled =
                      (mode === "live" && selectedTpl && !selectedTpl.allow_live) ||
                      (mode === "backtest" && selectedTpl && !selectedTpl.allow_backtest);
                    return (
                      <button
                        key={mode}
                        type="button"
                        disabled={disabled}
                        className={`flex w-full flex-col rounded-lg border px-4 py-3 text-left text-sm ${
                          form.run_mode === mode
                            ? "border-primary bg-accent"
                            : "border-border hover:bg-accent/60"
                        } ${disabled ? "cursor-not-allowed opacity-50" : ""}`}
                        onClick={() => setForm((f) => ({ ...f, run_mode: mode }))}
                      >
                        <span className="font-semibold capitalize">{mode}</span>
                        <span className="text-xs text-muted-foreground">
                          {mode === "backtest" && "离线回放 / 不产生实盘指令（模板需允许）"}
                          {mode === "paper" && "仿真撮合或影子账户；推荐默认演练"}
                          {mode === "live" && "真实下单路径 · 需模板允许且 Agent 连通"}
                        </span>
                      </button>
                    );
                  })}
                  {fieldErr.run_mode ? (
                    <p className="text-xs text-destructive">{fieldErr.run_mode}</p>
                  ) : null}
                </div>
              ) : null}

              {step === 4 ? (
                <div className="space-y-3">
                  <Label>选择 Agent</Label>
                  <div className="grid gap-2">
                    {agents.map((a) => (
                      <button
                        key={a.id}
                        type="button"
                        className={`rounded-lg border px-3 py-2 text-left text-sm ${
                          form.agent_key === a.id ? "border-primary bg-accent" : ""
                        }`}
                        onClick={() => setForm((f) => ({ ...f, agent_key: a.id }))}
                      >
                        <div className="font-medium">{a.displayName}</div>
                        <div className="text-[11px] text-muted-foreground">
                          {a.id} · {a.execMode} ·{" "}
                          {a.isOnline ? "在线" : "离线"}
                        </div>
                      </button>
                    ))}
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="agent_key_manual">或直接填写 AgentKey</Label>
                    <Input
                      id="agent_key_manual"
                      value={form.agent_key}
                      onChange={(ev) =>
                        setForm((f) => ({ ...f, agent_key: ev.target.value }))
                      }
                      placeholder="与 Agent YAML identity.client_id 一致"
                    />
                  </div>
                  {fieldErr.agent_key ? (
                    <p className="text-xs text-destructive">{fieldErr.agent_key}</p>
                  ) : null}
                </div>
              ) : null}

              {step === 5 ? (
                <div className="space-y-4 text-sm">
                  <div className="rounded-lg border border-border bg-muted/30 p-4 leading-relaxed">
                    <div>
                      <span className="text-muted-foreground">实例名：</span>
                      {form.display_name}
                    </div>
                    <div>
                      <span className="text-muted-foreground">模板：</span>
                      {selectedTpl?.name} (#{form.strategy_id})
                    </div>
                    <div>
                      <span className="text-muted-foreground">标的 / 市场：</span>
                      {form.symbol.toUpperCase()} · {form.market_kind}
                    </div>
                    <div>
                      <span className="text-muted-foreground">模式：</span>
                      {form.run_mode}
                    </div>
                    <div>
                      <span className="text-muted-foreground">AgentKey：</span>
                      {form.agent_key}
                    </div>
                  </div>
                  <label className="flex cursor-pointer items-start gap-2 text-sm">
                    <input
                      type="checkbox"
                      className="mt-1"
                      checked={form.ackDanger}
                      onChange={(ev) =>
                        setForm((f) => ({ ...f, ackDanger: ev.target.checked }))
                      }
                    />
                    <span>
                      我已阅读：实例创建后可通过「结束实例」软删除；实盘模式可能造成真实资金亏损。
                    </span>
                  </label>
                  {fieldErr.ackDanger ? (
                    <p className="text-xs text-destructive">{fieldErr.ackDanger}</p>
                  ) : null}
                  {submitErr ? (
                    <p className="text-xs text-destructive">{submitErr}</p>
                  ) : null}
                </div>
              ) : null}

              <div className="flex flex-wrap justify-between gap-3 border-t border-border pt-4">
                <Button
                  type="button"
                  variant="outline"
                  onClick={prev}
                  disabled={step === 0}
                >
                  上一步
                </Button>
                <div className="flex gap-2">
                  {step < STEPS.length - 1 ? (
                    <Button type="button" onClick={next}>
                      下一步
                      <ArrowRight className="ml-2 h-4 w-4" />
                    </Button>
                  ) : (
                    <Button type="button" onClick={() => void submit()}>
                      确认并创建
                    </Button>
                  )}
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </ConsolePage>
  );
}
