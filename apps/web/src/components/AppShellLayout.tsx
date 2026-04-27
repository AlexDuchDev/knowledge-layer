"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useAuthSession } from "@/hooks/useAuthSession";
import { isDevPrincipalHeader } from "@/lib/api";
import { canAccessZoneSwitcher, filterNavByVisibility, NAV_GROUPS } from "@/lib/navigation";
import { RoleOnboardingBanner } from "@/components/RoleOnboardingBanner";
import { MobileNavDrawer } from "@/components/shell/MobileNavDrawer";
import { ZoneSwitcher } from "@/components/shell/ZoneSwitcher";

export function AppShellLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const { me, navVis, authErr, logout } = useAuthSession();
  const groups = filterNavByVisibility(NAV_GROUPS, navVis);

  return (
    <div className="flex min-h-screen bg-neutral-50">
      <aside className="hidden w-56 shrink-0 border-r border-neutral-200 bg-white md:block">
        <div className="flex h-14 items-center border-b border-neutral-200 px-4">
          <Link href="/" className="text-sm font-semibold text-neutral-900">
            Knowledge Layer
          </Link>
        </div>
        <nav className="space-y-6 p-3 text-sm" aria-label="Primary">
          {groups.map((g) => (
            <div key={g.title}>
              <div className="px-2 text-xs font-semibold uppercase tracking-wide text-neutral-500">{g.title}</div>
              <ul className="mt-2 space-y-0.5">
                {g.items.map((it) => {
                  const active = pathname === it.href || (it.href !== "/" && pathname?.startsWith(it.href));
                  return (
                    <li key={it.href}>
                      <Link
                        href={it.href}
                        className={`block rounded-md px-2 py-1.5 ${
                          active ? "bg-neutral-100 font-medium text-neutral-900" : "text-neutral-700 hover:bg-neutral-50"
                        }`}
                      >
                        {it.label}
                      </Link>
                    </li>
                  );
                })}
              </ul>
            </div>
          ))}
        </nav>
      </aside>
      <div className="flex min-w-0 flex-1 flex-col">
        <RoleOnboardingBanner nav={navVis} pathname={pathname} />
        <header className="flex h-14 items-center justify-between gap-2 border-b border-neutral-200 bg-white px-4">
          <div className="flex min-w-0 flex-1 items-center gap-2">
            <MobileNavDrawer
              groups={groups}
              brandHref="/"
              brandLabel="Knowledge Layer"
              navAriaLabel="Primary navigation"
            />
            <Link href="/" className="truncate text-sm font-semibold text-neutral-900 md:hidden">
              Knowledge Layer
            </Link>
            <div className="hidden flex-1 md:block" aria-hidden />
            {canAccessZoneSwitcher(navVis) ? <ZoneSwitcher active="product" /> : null}
          </div>
          <div className="flex shrink-0 items-center gap-3 text-sm text-neutral-700">
            {me ? (
              <>
                <span className="hidden sm:inline truncate max-w-[14rem]" title={me.email}>
                  {me.name || me.email}
                </span>
                <button type="button" className="rounded border border-neutral-300 px-2 py-1 text-xs hover:bg-neutral-50" onClick={() => void logout()}>
                  Sign out
                </button>
              </>
            ) : authErr && isDevPrincipalHeader() ? (
              <span className="text-xs text-amber-800">Dev header · /auth/me failed</span>
            ) : !authErr ? (
              <span className="text-xs text-neutral-500">Loading…</span>
            ) : null}
          </div>
        </header>
        <div className="flex-1">{children}</div>
      </div>
    </div>
  );
}
