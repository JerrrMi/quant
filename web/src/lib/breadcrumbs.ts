import { breadcrumbLabels } from "@/lib/nav-config";

export type Crumb = { label: string; href?: string };

export function buildCrumbs(pathname: string): Crumb[] {
  const segments = pathname.split("/").filter(Boolean);
  const crumbs: Crumb[] = [];

  const showDashboardRoot =
    segments.length === 0 || segments[0] !== "dashboard";

  if (showDashboardRoot) {
    crumbs.push({ label: "Dashboard", href: "/dashboard" });
  }

  let accumulator = "";
  segments.forEach((segment, index) => {
    accumulator += `/${segment}`;
    const isLast = index === segments.length - 1;
    const label = breadcrumbLabels[segment] ?? segment;
    crumbs.push({
      label,
      href: isLast ? undefined : accumulator,
    });
  });

  return crumbs;
}
