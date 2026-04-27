import type { ReactNode } from "react";

/** Short inline hint under form fields (accessibility-friendly; not a replacement for labels). */
export function FieldHint({ children }: { children: ReactNode }) {
  return <p className="mt-1 text-xs text-neutral-500">{children}</p>;
}
