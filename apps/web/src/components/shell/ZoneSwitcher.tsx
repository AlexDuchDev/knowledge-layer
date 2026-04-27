"use client";

import Link from "next/link";

type Zone = "product" | "control_plane";

export function ZoneSwitcher({ active }: { active: Zone }) {
  return (
    <div className="flex items-center gap-1 rounded-lg border border-neutral-200 bg-neutral-100/80 p-0.5 text-xs font-medium">
      <Link
        href="/"
        className={`rounded-md px-2.5 py-1 ${
          active === "product" ? "bg-white text-neutral-900 shadow-sm" : "text-neutral-600 hover:text-neutral-900"
        }`}
        aria-current={active === "product" ? "page" : undefined}
      >
        Product
      </Link>
      <Link
        href="/control-plane/governance"
        className={`rounded-md px-2.5 py-1 ${
          active === "control_plane" ? "bg-white text-neutral-900 shadow-sm" : "text-neutral-600 hover:text-neutral-900"
        }`}
        aria-current={active === "control_plane" ? "page" : undefined}
      >
        Control plane
      </Link>
    </div>
  );
}
