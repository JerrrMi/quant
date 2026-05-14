export function formatUsd(value: number, opts?: { digits?: number }): string {
  const digits = opts?.digits ?? 2;
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  }).format(value);
}

export function formatSignedUsd(value: number, opts?: { digits?: number }): string {
  if (value > 0) return `+${formatUsd(value, opts)}`;
  if (value < 0) return `−${formatUsd(Math.abs(value), opts)}`;
  return formatUsd(0, opts);
}

export function formatRatioAsPercent(value: number, opts?: { digits?: number }): string {
  const digits = opts?.digits ?? 1;
  return `${(value * 100).toFixed(digits)}%`;
}

export function formatCompactNumber(value: number): string {
  return new Intl.NumberFormat("en-US", {
    notation: "compact",
    maximumFractionDigits: 2,
  }).format(value);
}

export function formatDateTime(iso: string | null | undefined): string {
  if (!iso) return "—";
  try {
    return new Date(iso).toLocaleString();
  } catch {
    return iso;
  }
}

export function formatRelativeShort(iso: string | null | undefined): string {
  if (!iso) return "—";
  try {
    const t = new Date(iso).getTime();
    const diff = Date.now() - t;
    if (diff < 60_000) return `${Math.max(1, Math.floor(diff / 1000))}s 前`;
    if (diff < 3600_000) return `${Math.floor(diff / 60_000)}m 前`;
    if (diff < 86400_000) return `${Math.floor(diff / 3600_000)}h 前`;
    return `${Math.floor(diff / 86400_000)}d 前`;
  } catch {
    return iso;
  }
}
