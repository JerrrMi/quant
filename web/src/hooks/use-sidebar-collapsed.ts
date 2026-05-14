"use client";

import { useCallback, useEffect, useState } from "react";

const STORAGE_KEY = "altshort.sidebar";

export function useSidebarCollapsed() {
  const [collapsed, setCollapsed] = useState(false);

  useEffect(() => {
    const frame = window.requestAnimationFrame(() => {
      const stored = window.localStorage.getItem(STORAGE_KEY);
      if (stored === "collapsed") {
        setCollapsed(true);
      }
    });
    return () => window.cancelAnimationFrame(frame);
  }, []);

  const toggle = useCallback(() => {
    setCollapsed((prev) => {
      const next = !prev;
      window.localStorage.setItem(STORAGE_KEY, next ? "collapsed" : "expanded");
      return next;
    });
  }, []);

  return { collapsed, toggle };
}
