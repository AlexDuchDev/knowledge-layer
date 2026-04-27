import { redirect } from "next/navigation";

export default async function LegacyAppDecisionDetailRedirect({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  redirect(`/decisions/${encodeURIComponent(id)}`);
}
