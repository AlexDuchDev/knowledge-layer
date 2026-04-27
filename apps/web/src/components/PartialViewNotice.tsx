type PartialViewNoticeProps = {
  className?: string;
  /** When true (domain stewards / operators), show governance-level detail. Otherwise plain language for readers. */
  detailed?: boolean;
};

/** Explains why lists may look sparse without implying hidden objects exist. */
export function PartialViewNotice({ className, detailed }: PartialViewNoticeProps) {
  return (
    <div
      className={`rounded-md border border-amber-200 bg-amber-50/90 px-3 py-2 text-xs text-amber-950 ${className ?? ""}`}
      role="note"
    >
      {detailed ? (
        <>
          <span className="font-semibold">Why you might see less here: </span>
          lists and search only include entities your account can read in granted domains, at or below your sensitivity cap. Missing items are not shown as
          placeholders—access is enforced before retrieval.
        </>
      ) : (
        <>
          <span className="font-semibold">About what you see: </span>
          Search, Ask, and lists only show content your account is allowed to open. If something you expect is missing, ask your workspace admin for access.
        </>
      )}
    </div>
  );
}
