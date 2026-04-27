"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";

export function FromCpBanner() {
  const sp = useSearchParams();
  if (sp.get("from") !== "cp") return null;
  return (
    <div className="border-b border-blue-200 bg-blue-50 px-4 py-2 text-center text-xs text-neutral-800">
      You opened this from the{" "}
      <Link href="/control-plane/sources" className="font-medium text-blue-800 underline">
        control plane
      </Link>
      . When finished, return to{" "}
      <Link href="/control-plane/sources" className="font-medium text-blue-800 underline">
        Sources hub
      </Link>{" "}
      or{" "}
      <Link href="/control-plane/governance" className="font-medium text-blue-800 underline">
        Governance
      </Link>
      .
    </div>
  );
}
