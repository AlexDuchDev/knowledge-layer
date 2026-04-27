import Link from "next/link";
import { CpScaffold } from "@/components/control-plane/CpScaffold";

const QUEUES: { href: string; title: string; body: string }[] = [
  {
    href: "/control-plane/governance/reviews",
    title: "Reviews",
    body: "Open review tasks awaiting analyst attention.",
  },
  {
    href: "/control-plane/governance/approvals",
    title: "Approvals",
    body: "Items awaiting publish approval (publish capability required).",
  },
  {
    href: "/control-plane/governance/stale",
    title: "Stale content",
    body: "Entities flagged stale by the freshness scan.",
  },
  {
    href: "/control-plane/governance/failed-jobs",
    title: "Failed jobs",
    body: "Recent job runs that exited with status=failed.",
  },
  {
    href: "/control-plane/governance/failed-syncs",
    title: "Failed syncs",
    body: "Source feed sync runs that exited with status=failed.",
  },
  {
    href: "/control-plane/governance/policy-exceptions",
    title: "Policy exceptions",
    body: "Active exceptions that override inherited access policy.",
  },
];

export default function GovernanceSummaryDashboardPage() {
  return (
    <CpScaffold
      title="Governance"
      description="Operator-context queues. Each card opens a native CP queue backed by the same APIs used by the product /governance, /reviews, /approvals pages — pick whichever shell suits your workflow."
      screenType="Operational"
      guidanceSlug="governance"
      crossLinks={[
        { href: "/governance", label: "Governance overview (product)" },
        { href: "/control-plane/jobs/runs", label: "All job runs" },
        { href: "/control-plane/sources", label: "Sources" },
      ]}
    >
      <ul className="mt-6 grid gap-3 sm:grid-cols-2">
        {QUEUES.map((q) => (
          <li key={q.href} className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
            <Link href={q.href} className="text-base font-medium text-blue-700 hover:underline">
              {q.title}
            </Link>
            <p className="mt-1 text-sm text-gray-600">{q.body}</p>
          </li>
        ))}
      </ul>
    </CpScaffold>
  );
}
