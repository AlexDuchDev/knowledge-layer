import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { NewSetupSessionPanel } from "@/components/control-plane/SetupControlPlaneClient";
import { Suspense } from "react";

export default function NewSetupSessionPage() {
  return (
    <CpScaffold
      title="Start setup session"
      description="Create a real onboarding session that you can reopen, preview, and launch. Use the template catalog first if you want the session prefilled."
      screenType="Builder"
      guidanceSlug="setup"
      crossLinks={[
        { href: "/control-plane/setup/templates", label: "Setup templates" },
        { href: "/control-plane/setup/launch-preview", label: "Launch preview" },
      ]}
    >
      <Suspense fallback={<div className="text-sm text-neutral-500">Loading session builder…</div>}>
        <NewSetupSessionPanel />
      </Suspense>
    </CpScaffold>
  );
}
