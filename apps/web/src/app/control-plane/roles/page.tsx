import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { RolesCatalogClient } from "@/components/control-plane/RolesCatalogClient";

export default function ControlPlaneRolesListPage() {
  return (
    <CpScaffold
      title="Roles & Access"
      description="Reusable permission patterns. Browse roles and presets, inspect detail/access preview/assignments. Scenarios bind roles to users in time-boxed contexts."
      screenType="Catalog"
      guidanceSlug="roles"
      crossLinks={[
        { href: "/control-plane/roles/new", label: "Create role" },
        { href: "/control-plane/scenarios", label: "Scenarios using roles" },
        { href: "/control-plane/users", label: "Users" },
      ]}
    >
      <RolesCatalogClient />
    </CpScaffold>
  );
}
