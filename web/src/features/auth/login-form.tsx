"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

import { ApiError, formatApiErrorMessage } from "@/api/errors";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { publicEnv } from "@/lib/env";
import {
  getRememberMePlaceholder,
  setRememberMePlaceholder,
} from "@/lib/session-store";
import { cn } from "@/lib/utils";

import { useAuth } from "./auth-context";

const loginSchema = z.object({
  account: z.string().min(2, { message: "账号至少 2 个字符" }),
  password: z.string().min(6, { message: "密码至少 6 位" }),
});

type LoginValues = z.infer<typeof loginSchema>;

export function LoginForm() {
  const { login } = useAuth();
  const [rememberPlaceholder, setRememberPlaceholder] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const form = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      account: "",
      password: "",
    },
  });

  useEffect(() => {
    void Promise.resolve().then(() => {
      setRememberPlaceholder(getRememberMePlaceholder());
    });
  }, []);

  const onSubmit = form.handleSubmit(async (values) => {
    setSubmitError(null);
    try {
      await login(values.account, values.password);
      toast.success("登录成功");
    } catch (error) {
      const message =
        error instanceof ApiError
          ? formatApiErrorMessage(error.body, error.message)
          : "登录失败，请稍后重试";
      setSubmitError(message);
      toast.error(message);
    }
  });

  return (
    <Card className="w-full max-w-md border-border/80 shadow-md">
      <CardHeader className="space-y-2">
        <CardTitle className="text-xl font-semibold">登录控制台</CardTitle>
        <CardDescription>
          使用控制台账号登录；交易所 API Key 仅保存在 Agent 进程，不会写入浏览器。
        </CardDescription>
        {publicEnv.devMockAuth ? (
          <p className="rounded-md border border-dashed border-amber-500/40 bg-amber-500/10 px-3 py-2 text-xs text-amber-900 dark:text-amber-100">
            开发 Mock 模式已开启（NEXT_PUBLIC_DEV_MOCK_AUTH）：将跳过后端会话 Cookie
            校验，便于离线调试导航。
          </p>
        ) : null}
      </CardHeader>
      <CardContent>
        <form className="space-y-4" onSubmit={onSubmit}>
          <div className="space-y-2">
            <Label htmlFor="account">账号</Label>
            <Input
              id="account"
              type="text"
              autoComplete="username"
              placeholder="邮箱或用户名"
              {...form.register("account")}
            />
            {form.formState.errors.account ? (
              <p className="text-xs text-destructive">
                {form.formState.errors.account.message}
              </p>
            ) : null}
          </div>
          <div className="space-y-2">
            <Label htmlFor="password">密码</Label>
            <Input
              id="password"
              type="password"
              autoComplete="current-password"
              {...form.register("password")}
            />
            {form.formState.errors.password ? (
              <p className="text-xs text-destructive">
                {form.formState.errors.password.message}
              </p>
            ) : null}
          </div>

          <label className="flex cursor-pointer items-center gap-2 text-sm text-muted-foreground">
            <input
              type="checkbox"
              className="h-3.5 w-3.5 rounded border border-input"
              checked={rememberPlaceholder}
              onChange={(e) => {
                const next = e.target.checked;
                setRememberPlaceholder(next);
                setRememberMePlaceholder(next);
              }}
            />
            <span>
              记住我{" "}
              <span className="text-xs opacity-80">
                （占位：后续与后端会话 / 刷新策略对齐）
              </span>
            </span>
          </label>

          {submitError ? (
            <div
              role="alert"
              className="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive"
            >
              {submitError}
            </div>
          ) : null}

          <Button
            className={cn("w-full")}
            type="submit"
            disabled={form.formState.isSubmitting}
          >
            {form.formState.isSubmitting ? "登录中…" : "登录"}
          </Button>
        </form>
      </CardContent>
      <CardFooter className="flex-col items-start gap-2 text-xs text-muted-foreground">
        <p>
          SaaS API 根路径通过环境变量{" "}
          <code className="rounded bg-muted px-1 py-0.5 text-[11px]">
            NEXT_PUBLIC_API_BASE_URL
          </code>{" "}
          配置；页面调用统一走{" "}
          <code className="rounded bg-muted px-1 py-0.5 text-[11px]">
            src/api/*.ts
          </code>{" "}
          封装。
        </p>
      </CardFooter>
    </Card>
  );
}
