import { redirect } from "next/navigation";

export default async function ControlPlaneSourceIdRedirect({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  redirect(`/control-plane/sources/feeds/${encodeURIComponent(id)}`);
}
