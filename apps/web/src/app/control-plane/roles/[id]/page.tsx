import { CpScaffold } from "@/components/control-plane/CpScaffold";

export default async function EditRolePage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <CpScaffold
      title="Edit role"
      description={`Role id ${id}. Role definition vs assignment: use Assignments for user links.`}
      screenType="Builder"
      guidanceSlug="roles"
      crossLinks={[
        { href: `/control-plane/roles/${encodeURIComponent(id)}/preview`, label: "Preview" },
        { href: `/control-plane/roles/${encodeURIComponent(id)}/assignments`, label: "Assignments" },
        { href: `/control-plane/scenarios`, label: "Linked scenarios" },
        { href: `/control-plane/roles/new`, label: "Clone flow (new)" },
      ]}
    />
  );
}
