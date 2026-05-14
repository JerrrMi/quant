"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

import { ApiError } from "@/api/errors";
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

import { useAuth } from "./auth-context";

const loginSchema = z.object({
  email: z.string().email({ message: "请输入有效的邮箱地址" }),
  password: z.string().min(6, { message: "密码至少 6 位" }),
});

type LoginValues = z.infer<typeof loginSchema>;

export function LoginForm() {
  const { login } = useAuth();
  const form = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      email: "",
      password: "",
    },
  });

  const onSubmit = form.handleSubmit(async (values) => {
    try {
      await login(values.email, values.password);
      toast.success("登录成功");
    } catch (error) {
      const message =
        error instanceof ApiError ? error.message : "登录失败，请稍后重试";
      toast.error(message);
    }
  });

  return (
    <Card className="w-full max-w-md border-border/80 shadow-md">
      <CardHeader className="space-y-2">
        <CardTitle className="text-xl font-semibold">登录控制台</CardTitle>
        <CardDescription>
          使用控制台账号登录；密钥仅保存在 Agent 侧，不在浏览器持久化交易所凭证。
        </CardDescription>
        {publicEnv.devMockAuth ? (
          <p className="rounded-md border border-dashed border-amber-500/40 bg-amber-500/10 px-3 py-2 text-xs text-amber-900 dark:text-amber-100">
            开发 Mock 模式已开启（NEXT_PUBLIC_DEV_MOCK_AUTH）：后端未就绪时可跳过会话校验。
          </p>
        ) : null}
      </CardHeader>
      <CardContent>
        <form className="space-y-4" onSubmit={onSubmit}>
          <div className="space-y-2">
            <Label htmlFor="email">邮箱</Label>
            <Input
              id="email"
              type="email"
              autoComplete="username"
              {...form.register("email")}
            />
            {form.formState.errors.email ? (
              <p className="text-xs text-destructive">
                {form.formState.errors.email.message}
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
          <Button
            className="w-full"
            type="submit"
            disabled={form.formState.isSubmitting}
          >
            {form.formState.isSubmitting ? "登录中..." : "登录"}
          </Button>
        </form>
      </CardContent>
      <CardFooter className="flex-col items-start gap-2 text-xs text-muted-foreground">
        <p>首次接入后端时，请将 SaaS API 基础路径写入 NEXT_PUBLIC_API_BASE_URL。</p>
      </CardFooter>
    </Card>
  );
}
