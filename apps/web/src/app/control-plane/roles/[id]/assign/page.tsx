import { CpScaffold } from "@/components/control-plane/CpScaffold";

export default async function UserRoleAssignmentFlowPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <CpScaffold
      title="User–role assignment"
      description={`Flow: pick user and confirm assignment to role ${id} (scaffold).`}
      screenType="Binding"
      guidanceSlug="roles"
      crossLinks={[
        { href: `/control-plane/roles/${encodeURIComponent(id)}/assignments`, label: "Assignments list" },
        { href: `/control-plane/users`, label: "Users" },
      ]}
    />
  );
}
