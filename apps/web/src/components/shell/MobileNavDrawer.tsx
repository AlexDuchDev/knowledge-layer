"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useCallback, useEffect, useId, useRef, useState } from "react";

export type MobileNavItem = { href: string; label: string };
export type MobileNavGroup = { title: string; items: MobileNavItem[] };

type Props = {
  groups: MobileNavGroup[];
  brandHref: string;
  brandLabel: string;
  navAriaLabel: string;
};

export function MobileNavDrawer({ groups, brandHref, brandLabel, navAriaLabel }: Props) {
  const [open, setOpen] = useState(false);
  const pathname = usePathname();
  const panelId = useId();
  const closeBtnRef = useRef<HTMLButtonElement>(null);
  const openBtnRef = useRef<HTMLButtonElement>(null);

  const close = useCallback(() => setOpen(false), []);

  useEffect(() => {
    close();
  }, [pathname, close]);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") close();
    };
    document.addEventListener("keydown", onKey);
    document.body.style.overflow = "hidden";
    return () => {
      document.removeEventListener("keydown", onKey);
      document.body.style.overflow = "";
    };
  }, [open, close]);

  useEffect(() => {
    if (open) {
      closeBtnRef.current?.focus();
    } else {
      openBtnRef.current?.focus();
    }
  }, [open]);

  return (
    <div className="md:hidden">
      <button
        ref={openBtnRef}
        type="button"
        className="rounded-md border border-neutral-300 bg-white px-2.5 py-1.5 text-xs font-medium text-neutral-800"
        aria-expanded={open}
        aria-controls={panelId}
        onClick={() => setOpen(true)}
      >
        Menu
      </button>
      {open ? (
        <div className="fixed inset-0 z-50 flex">
          <button type="button" className="absolute inset-0 bg-black/40" aria-label="Close menu" onClick={close} />
          <div
            id={panelId}
            role="dialog"
            aria-modal="true"
            aria-label={navAriaLabel}
            className="relative ml-0 flex h-full w-[min(20rem,92vw)] flex-col border-r border-neutral-200 bg-white shadow-xl"
          >
            <div className="flex h-14 items-center justify-between border-b border-neutral-200 px-4">
              <Link href={brandHref} className="text-sm font-semibold text-neutral-900" onClick={close}>
                {brandLabel}
              </Link>
              <button
                ref={closeBtnRef}
                type="button"
                className="rounded border border-neutral-300 px-2 py-1 text-xs"
                onClick={close}
              >
                Close
              </button>
            </div>
            <nav className="flex-1 overflow-y-auto p-3 text-sm" aria-label={navAriaLabel}>
              {groups.map((g) => (
                <div key={g.title} className="mb-5">
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
                            onClick={close}
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
          </div>
        </div>
      ) : null}
    </div>
  );
}
