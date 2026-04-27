import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { SetupLaunchResultPanel } from "@/components/control-plane/SetupControlPlaneClient";
import { Suspense } from "react";

export default function SetupLaunchResultPage() {
  return (
    <CpScaffold
      title="Setup launch result"
      description="Inspect the real launch outcome for an onboarding session, including created roles, scenarios, and jobs."
      screenType="Operational"
      guidanceSlug="setup"
      crossLinks={[
        { href: "/control-plane/roles", label: "Open roles" },
        { href: "/control-plane/scenarios", label: "Open scenarios" },
        { href: "/control-plane/jobs", label: "Open jobs" },
        { href: "/control-plane/governance", label: "Governance dashboard" },
      ]}
    >
      <Suspense fallback={<div className="text-sm text-neutral-500">Loading launch result…</div>}>
        <SetupLaunchResultPanel />
      </Suspense>
    </CpScaffold>
  );
}
