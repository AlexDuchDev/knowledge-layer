import { CpScaffold } from "@/components/control-plane/CpScaffold";

export default function ControlPlaneUsersListPage() {
  return (
    <CpScaffold
      title="Users"
      description="Directory and assignments; effective visibility summary on user detail."
      screenType="Catalog"
      guidanceSlug="roles"
      crossLinks={[
        { href: "/control-plane/roles", label: "Roles" },
      ]}
    />
  );
}
