import { CpScaffold } from "@/components/control-plane/CpScaffold";

export default function SetupWizardPage() {
  return (
    <CpScaffold
      title="Setup wizard"
      description="Stepwise onboarding: presets, sources, preview (scaffold)."
      screenType="Setup"
      guidanceSlug="setup"
      crossLinks={[
        { href: "/control-plane/presets", label: "Select presets" },
        { href: "/control-plane/sources", label: "Connect sources" },
        { href: "/control-plane/setup/launch-preview", label: "Launch preview" },
      ]}
    />
  );
}
