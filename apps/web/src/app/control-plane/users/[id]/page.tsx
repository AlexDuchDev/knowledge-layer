import { CpScaffold } from "@/components/control-plane/CpScaffold";

export default async function ControlPlaneUserDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <CpScaffold
      title="User detail"
      description={`User ${id}: role assignments and effective access summary (scaffold).`}
      screenType="Detail"
      guidanceSlug="roles"
      crossLinks={[
        { href: "/control-plane/roles", label: "Roles" },
        { href: "/control-plane/users", label: "User list" },
      ]}
    />
  );
}
