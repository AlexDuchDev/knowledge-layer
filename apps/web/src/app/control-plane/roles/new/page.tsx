import { CpScaffold } from "@/components/control-plane/CpScaffold";

export default function CreateRolePage() {
  return (
    <CpScaffold
      title="Create role"
      description="Builder: define role; save then assign to users."
      screenType="Builder"
      guidanceSlug="roles"
      crossLinks={[
        { href: "/control-plane/presets", label: "Start from preset" },
        { href: "/control-plane/roles", label: "Role list" },
        { href: "/control-plane/roles/role-demo/preview", label: "Preview (example)" },
      ]}
    />
  );
}
