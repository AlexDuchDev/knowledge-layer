"use client";

import type { InputHTMLAttributes } from "react";

export function Toggle({
  label,
  id,
  className = "",
  ...props
}: InputHTMLAttributes<HTMLInputElement> & { label: string; id: string }) {
  return (
    <label htmlFor={id} className={`inline-flex cursor-pointer items-center gap-2 text-sm text-neutral-700 ${className}`}>
      <span className="relative inline-block h-5 w-9 shrink-0">
        <input id={id} type="checkbox" role="switch" className="peer sr-only" {...props} />
        <span className="block h-full w-full rounded-full bg-neutral-200 transition peer-checked:bg-neutral-800 peer-focus-visible:ring-2 peer-focus-visible:ring-neutral-400" />
        <span className="absolute left-0.5 top-0.5 h-4 w-4 rounded-full bg-white shadow transition peer-checked:translate-x-4" />
      </span>
      {label}
    </label>
  );
}
