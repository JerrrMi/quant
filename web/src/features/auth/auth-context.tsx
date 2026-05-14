"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";

import { getSession, login as loginApi, logout as logoutApi } from "@/api/auth";
import type { AuthUserDTO } from "@/types/auth";

type AuthContextValue = {
  user: AuthUserDTO | null;
  loading: boolean;
  refresh: () => Promise<void>;
  login: (account: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const router = useRouter();
  const [user, setUser] = useState<AuthUserDTO | null>(null);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const session = await getSession();
      setUser(session?.user ?? null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const frame = window.requestAnimationFrame(() => {
      void refresh();
    });
    return () => window.cancelAnimationFrame(frame);
  }, [refresh]);

  useEffect(() => {
    const handler = () => {
      toast.message("会话已失效，请重新登录");
      setUser(null);
      router.replace("/login");
    };

    window.addEventListener("altshort:session-expired", handler);
    return () => window.removeEventListener("altshort:session-expired", handler);
  }, [router]);

  const login = useCallback(
    async (account: string, password: string) => {
      await loginApi({ account, password });
      await refresh();
      router.replace("/dashboard");
    },
    [refresh, router],
  );

  const logout = useCallback(async () => {
    try {
      await logoutApi();
    } finally {
      setUser(null);
      router.replace("/login");
    }
  }, [router]);

  const value = useMemo(
    () => ({
      user,
      loading,
      refresh,
      login,
      logout,
    }),
    [user, loading, refresh, login, logout],
  );

  return (
    <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return ctx;
}
