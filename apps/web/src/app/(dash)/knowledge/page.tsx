import Link from "next/link";
import { BROWSE_ROUTES } from "@/lib/entityTypes";

export default function KnowledgeIndexPage() {
  return (
    <main className="mx-auto max-w-3xl px-6 py-10">
      <h1 className="text-2xl font-semibold tracking-tight">Knowledge</h1>
      <p className="mt-2 text-sm text-neutral-600">Browse governed entities by type. Lists respect domain grants.</p>
      <ul className="mt-6 space-y-2">
        {Object.entries(BROWSE_ROUTES).map(([k, v]) => (
          <li key={k}>
            <Link href={v.path} className="text-blue-700 underline">
              {v.label}
            </Link>
          </li>
        ))}
      </ul>
      <p className="mt-8">
        <Link href="/" className="text-sm text-blue-700 underline">
          Home
        </Link>
      </p>
    </main>
  );
}
