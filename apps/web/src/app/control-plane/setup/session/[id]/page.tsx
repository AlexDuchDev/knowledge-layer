import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { SetupSessionWizardClient } from "@/components/control-plane/SetupSessionWizardClient";

export default async function ResumeSetupSessionPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <CpScaffold
      title="Setup session"
      description="Five-step wizard: pick a template → toggle connector families → assign initial admin → preview the plan → launch."
      screenType="Builder"
      guidanceSlug="setup"
      crossLinks={[
        { href: "/control-plane/setup", label: "Setup hub" },
        { href: "/control-plane/presets", label: "Preset catalog" },
      ]}
    >
      <SetupSessionWizardClient sessionId={id} />
    </CpScaffold>
  );
}
