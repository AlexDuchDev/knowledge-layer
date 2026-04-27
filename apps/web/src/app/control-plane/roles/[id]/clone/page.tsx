import { CpScaffold } from "@/components/control-plane/CpScaffold";

export default async function CloneRolePage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <CpScaffold
      title="Clone role"
      description={`Create a new role from ${id} (scaffold).`}
      screenType="Builder"
      guidanceSlug="roles"
      crossLinks={[
        { href: `/control-plane/roles/${encodeURIComponent(id)}`, label: "Source role" },
        { href: "/control-plane/roles/new", label: "Blank create" },
        { href: "/control-plane/roles", label: "Role list" },
      ]}
    />
  );
}
