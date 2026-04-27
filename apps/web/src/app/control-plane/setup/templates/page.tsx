import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { SetupTemplatesPanel } from "@/components/control-plane/SetupControlPlaneClient";

export default function SetupTemplatesPage() {
  return (
    <CpScaffold
      title="Setup templates"
      description="Catalog of onboarding templates that can be turned into real setup sessions. Templates surface only the currently supported preset and connector-family combinations."
      screenType="Catalog"
      guidanceSlug="setup"
      crossLinks={[
        { href: "/control-plane/setup", label: "Setup hub" },
        { href: "/control-plane/setup/session/new", label: "Start session" },
        { href: "/control-plane/presets", label: "Presets" },
      ]}
    >
      <SetupTemplatesPanel />
    </CpScaffold>
  );
}
