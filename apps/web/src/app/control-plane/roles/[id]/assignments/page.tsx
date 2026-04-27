import { CpScaffold } from "@/components/control-plane/CpScaffold";

export default async function RoleAssignmentsPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <CpScaffold
      title="Role assignments"
      description={`Binding screen: users assigned to role ${id}.`}
      screenType="Binding"
      guidanceSlug="roles"
      crossLinks={[
        { href: `/control-plane/roles/${encodeURIComponent(id)}/assign`, label: "Assign user" },
        { href: `/control-plane/users`, label: "User directory" },
        { href: `/control-plane/roles/${encodeURIComponent(id)}`, label: "Role definition" },
      ]}
    />
  );
}
