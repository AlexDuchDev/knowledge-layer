"use client";

import Link from "next/link";
import { TrustBadge, TrustLine } from "@/components/TrustBadge";

export type KnowledgeCardVariant = "entity" | "answer" | "digest";

export type KnowledgeCardCitation = { entity_id: string; quote?: string };

export type KnowledgeCardProps = {
  variant?: KnowledgeCardVariant;
  density?: "compact" | "comfortable";
  title: string;
  href?: string;
  entityType?: string;
  truthMode?: string;
  lifecycleState?: string;
  freshnessStatus?: string;
  snippet?: string;
  /** Answer variant: short citations list */
  citations?: KnowledgeCardCitation[];
  footer?: React.ReactNode;
  className?: string;
};

/**
 * Compact trusted surface for lists, Ask, search, Home — extension-friendly (Slack/Teams/browser later).
 */
export function KnowledgeCard({
  variant = "entity",
  density = "comfortable",
  title,
  href,
  entityType,
  truthMode,
  lifecycleState,
  freshnessStatus,
  snippet,
  citations,
  footer,
  className = "",
}: KnowledgeCardProps) {
  const isCompact = density === "compact";
  const titleNode = href ? (
    <Link href={href} className="font-medium text-blue-800 underline">
      {title}
    </Link>
  ) : (
    <span className="font-medium text-neutral-900">{title}</span>
  );

  return (
    <article
      className={`rounded-lg border border-neutral-200 bg-white ${isCompact ? "p-2" : "p-3"} ${className}`}
      data-variant={variant}
      data-density={density}
    >
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0 flex-1">{titleNode}</div>
        {truthMode ? <TrustBadge truthMode={truthMode} /> : null}
      </div>
      {entityType ? <div className={`text-neutral-600 ${isCompact ? "mt-0.5 text-[10px]" : "mt-1 text-xs"}`}>{entityType}</div> : null}
      {truthMode ? (
        <div className={isCompact ? "mt-1" : "mt-2"}>
          <TrustLine truthMode={truthMode} lifecycleState={lifecycleState} freshnessStatus={freshnessStatus} />
        </div>
      ) : null}
      {snippet ? (
        <p className={`mt-2 text-neutral-700 ${isCompact ? "line-clamp-2 text-xs" : "line-clamp-3 text-sm"}`}>{snippet}</p>
      ) : null}
      {variant === "answer" && citations && citations.length > 0 ? (
        <ul className={`mt-2 space-y-0.5 ${isCompact ? "text-[10px]" : "text-xs"} text-neutral-600`}>
          {citations.slice(0, isCompact ? 2 : 4).map((c) => (
            <li key={c.entity_id}>
              <Link href={`/entities/${c.entity_id}`} className="text-blue-700 underline">
                {c.entity_id.slice(0, 8)}…
              </Link>
              {c.quote ? <span className="text-neutral-500"> — {c.quote.slice(0, 60)}</span> : null}
            </li>
          ))}
        </ul>
      ) : null}
      {footer ? <div className={`${isCompact ? "mt-1" : "mt-2"} border-t border-neutral-100 pt-2`}>{footer}</div> : null}
    </article>
  );
}
