"use client";

import { useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import clsx from "clsx";
import {
  ArrowLeftRight,
  LayoutDashboard,
  LogOut,
  Menu,
  PieChart,
  Tags,
  Wallet,
  X,
} from "lucide-react";
import { useAuth } from "@/lib/auth-context";

const NAV = [
  { href: "/", label: "Tổng quan", icon: LayoutDashboard },
  { href: "/transactions", label: "Giao dịch", icon: ArrowLeftRight },
  { href: "/reports", label: "Báo cáo", icon: PieChart },
  { href: "/categories", label: "Danh mục", icon: Tags },
  { href: "/accounts", label: "Nguồn tiền", icon: Wallet },
];

export function AppShell({ children }: { children: React.ReactNode }) {
  const [menuOpen, setMenuOpen] = useState(false);

  return (
    <div className="flex min-h-screen">
      {/* Trên máy tính thanh điều hướng luôn hiện; trên điện thoại nó
          trượt ra khi bấm nút menu. */}
      <Sidebar open={menuOpen} onNavigate={() => setMenuOpen(false)} />

      <div className="flex min-w-0 flex-1 flex-col">
        <TopBar onToggleMenu={() => setMenuOpen((v) => !v)} menuOpen={menuOpen} />
        <main className="mx-auto w-full max-w-6xl flex-1 p-4 sm:p-6">{children}</main>
      </div>
    </div>
  );
}

function Sidebar({ open, onNavigate }: { open: boolean; onNavigate: () => void }) {
  const pathname = usePathname();

  return (
    <aside
      className={clsx(
        "w-60 shrink-0 border-r border-line bg-surface p-4",
        "max-md:fixed max-md:inset-y-0 max-md:z-40 max-md:transition-transform",
        open ? "max-md:translate-x-0" : "max-md:-translate-x-full",
      )}
    >
      <div className="mb-6 flex items-center gap-2 px-2">
        <span className="grid h-8 w-8 place-items-center rounded-lg bg-brand text-sm font-bold text-brand-fg">
          F
        </span>
        <span className="font-semibold">FinTrack</span>
      </div>

      <nav className="space-y-1">
        {NAV.map(({ href, label, icon: Icon }) => {
          // So sánh bằng nhau tuyệt đối cho "/" vì mọi đường dẫn đều bắt
          // đầu bằng "/", dùng startsWith sẽ khiến Tổng quan luôn sáng.
          const active = href === "/" ? pathname === "/" : pathname.startsWith(href);
          return (
            <Link
              key={href}
              href={href}
              onClick={onNavigate}
              aria-current={active ? "page" : undefined}
              className={clsx(
                "flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors",
                active
                  ? "bg-brand-soft font-medium text-brand-on-soft"
                  : "text-fg-muted hover:bg-surface-2 hover:text-fg",
              )}
            >
              <Icon size={18} />
              {label}
            </Link>
          );
        })}
      </nav>
    </aside>
  );
}

function TopBar({ onToggleMenu, menuOpen }: { onToggleMenu: () => void; menuOpen: boolean }) {
  const { user, logout } = useAuth();

  return (
    <header className="sticky top-0 z-30 flex h-14 items-center gap-3 border-b border-line bg-bg/80 px-4 backdrop-blur sm:px-6">
      <button
        type="button"
        onClick={onToggleMenu}
        aria-label={menuOpen ? "Đóng menu" : "Mở menu"}
        aria-expanded={menuOpen}
        className="rounded-md p-1.5 text-fg-muted hover:bg-surface-2 hover:text-fg md:hidden"
      >
        {menuOpen ? <X size={18} /> : <Menu size={18} />}
      </button>

      <div className="ml-auto flex items-center gap-3">
        <span className="hidden text-sm text-fg-muted sm:block">{user?.full_name}</span>
        <button
          type="button"
          onClick={() => logout()}
          className="flex items-center gap-2 rounded-lg px-2 py-1.5 text-sm text-fg-muted hover:bg-surface-2 hover:text-fg"
        >
          <LogOut size={16} />
          <span className="hidden sm:inline">Đăng xuất</span>
        </button>
      </div>
    </header>
  );
}
