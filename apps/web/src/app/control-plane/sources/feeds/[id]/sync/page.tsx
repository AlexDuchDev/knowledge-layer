import { redirect } from "next/navigation";

export default async function ControlPlaneFeedSyncRedirect({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  redirect(`/source-feeds/${encodeURIComponent(id)}/sync?from=cp`);
}
