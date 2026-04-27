import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { EffectiveAccessClient } from "@/components/control-plane/EffectiveAccessClient";

export default async function UserEffectiveAccessPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <CpScaffold
      title="Effective access"
      description={`Inspect the 9-step access pipeline for user ${id}. Pick an action + resource and see exactly which gate allowed or denied.`}
      screenType="Detail"
      guidanceSlug="roles"
      crossLinks={[
        { href: `/control-plane/users/${encodeURIComponent(id)}`, label: "User detail" },
        { href: "/control-plane/roles", label: "Roles" },
        { href: "/control-plane/scenarios", label: "Scenarios" },
      ]}
    >
      <EffectiveAccessClient userId={id} />
    </CpScaffold>
  );
}
