import { redirect } from "next/navigation";

export default async function LegacyAppDigestDetailRedirect({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  redirect(`/insights/${encodeURIComponent(id)}`);
}
