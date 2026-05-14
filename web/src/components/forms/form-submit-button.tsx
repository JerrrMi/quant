"use client";

import type { ButtonHTMLAttributes } from "react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

type FormSubmitButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  loading?: boolean;
  loadingLabel?: string;
};

export function FormSubmitButton({
  loading,
  loadingLabel = "提交中…",
  children,
  className,
  disabled,
  ...props
}: FormSubmitButtonProps) {
  return (
    <Button
      type="submit"
      className={cn(className)}
      disabled={disabled || loading}
      {...props}
    >
      {loading ? loadingLabel : children}
    </Button>
  );
}
