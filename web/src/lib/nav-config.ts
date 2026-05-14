import {
  Bot,
  Boxes,
  FlaskConical,
  LayoutDashboard,
  Layers,
  LineChart,
  ListOrdered,
  ScrollText,
  Wallet,
  type LucideIcon,
} from "lucide-react";

export type NavItem =
  | {
      kind: "link";
      title: string;
      href: string;
      icon: LucideIcon;
    }
  | {
      kind: "group";
      title: string;
      icon: LucideIcon;
      children: { title: string; href: string; icon: LucideIcon }[];
    };

export const mainNav: NavItem[] = [
  {
    kind: "link",
    title: "Dashboard",
    href: "/dashboard",
    icon: LayoutDashboard,
  },
  {
    kind: "link",
    title: "Agents",
    href: "/agents",
    icon: Bot,
  },
  {
    kind: "group",
    title: "Strategies",
    icon: LineChart,
    children: [
      {
        title: "Templates",
        href: "/strategies/templates",
        icon: Layers,
      },
      {
        title: "Instances",
        href: "/strategies/instances",
        icon: Boxes,
      },
    ],
  },
  {
    kind: "link",
    title: "Backtests",
    href: "/backtests",
    icon: FlaskConical,
  },
  {
    kind: "link",
    title: "Accounts",
    href: "/accounts",
    icon: Wallet,
  },
  {
    kind: "link",
    title: "命令流",
    href: "/commands",
    icon: ListOrdered,
  },
  {
    kind: "link",
    title: "Logs",
    href: "/logs",
    icon: ScrollText,
  },
];

export const breadcrumbLabels: Record<string, string> = {
  dashboard: "Dashboard",
  agents: "Agents",
  strategies: "Strategies",
  templates: "Templates",
  instances: "Instances",
  backtests: "Backtests",
  accounts: "Accounts",
  commands: "命令流",
  logs: "Logs",
  login: "Login",
};
