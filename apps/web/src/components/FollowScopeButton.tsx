"use client";

import { useCallback, useEffect, useState } from "react";
import { apiJson } from "@/lib/api";

type FollowRow = {
  scope_type: string;
  ref_id: string;
  entity_type?: string;
};

/**
 * Surfacing-only follow (does not grant access). See GET/POST/DELETE /me/follows.
 */
export function FollowScopeButton({
  scopeType,
  refId,
  entityType = "",
  className = "",
}: {
  scopeType: "domain" | "content_hub" | "knowledge_topic" | "digest_stream";
  refId: string;
  entityType?: string;
  className?: string;
}) {
  const [following, setFollowing] = useState<boolean | null>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setErr(null);
    try {
      const list = await apiJson<FollowRow[]>("/me/follows");
      const on = list.some(
        (x) =>
          x.scope_type === scopeType &&
          x.ref_id === refId &&
          (x.entity_type ?? "") === (entityType ?? ""),
      );
      setFollowing(on);
    } catch (e) {
      setFollowing(null);
      setErr(e instanceof Error ? e.message : String(e));
    }
  }, [scopeType, refId, entityType]);

  useEffect(() => {
    if (!refId) return;
    void refresh();
  }, [refId, refresh]);

  const toggle = async () => {
    if (!refId || following === null) return;
    setBusy(true);
    setErr(null);
    try {
      if (following) {
        const q = new URLSearchParams({
          scope_type: scopeType,
          ref_id: refId,
          entity_type: entityType ?? "",
        });
        await apiJson(`/me/follows?${q}`, { method: "DELETE" });
        setFollowing(false);
      } else {
        await apiJson("/me/follows", {
          method: "POST",
          body: JSON.stringify({ scope_type: scopeType, ref_id: refId, entity_type: entityType ?? "" }),
        });
        setFollowing(true);
      }
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  if (!refId) {
    return (
      <p className={`text-xs text-neutral-500 ${className}`}>
        Choose a domain to follow this slice in your feed (surfacing only, not access).
      </p>
    );
  }

  return (
    <div className={className}>
      <button
        type="button"
        disabled={busy || following === null}
        onClick={() => void toggle()}
        className="rounded-md border border-neutral-300 bg-white px-3 py-1.5 text-xs font-medium text-neutral-900 disabled:opacity-50"
      >
        {following === null
          ? "…"
          : following
            ? scopeType === "digest_stream"
              ? "Following digests"
              : "Following (feed)"
            : scopeType === "digest_stream"
              ? "Follow digest stream"
              : "Follow in feed"}
      </button>
      <p className="mt-1 text-[10px] text-neutral-500">
        Surfacing preference only — does not change permissions.
      </p>
      {err ? <p className="mt-1 text-[10px] text-red-700">{err}</p> : null}
    </div>
  );
}
