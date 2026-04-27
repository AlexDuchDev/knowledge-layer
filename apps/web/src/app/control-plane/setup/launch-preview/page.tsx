import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { SetupPreviewPanel } from "@/components/control-plane/SetupControlPlaneClient";
import { Suspense } from "react";

export default function SetupLaunchPreviewPage() {
  return (
    <CpScaffold
      title="Setup launch preview"
      description="Preview the actual roles, scenarios, jobs, connector families, and validation issues for a real onboarding session before launch."
      screenType="Preview"
      guidanceSlug="setup"
      crossLinks={[
        { href: "/control-plane/setup/launch-result", label: "Launch result" },
        { href: "/control-plane/roles", label: "Roles list" },
        { href: "/control-plane/scenarios", label: "Scenarios list" },
        { href: "/control-plane/jobs", label: "Jobs list" },
      ]}
    >
      <Suspense fallback={<div className="text-sm text-neutral-500">Loading launch preview…</div>}>
        <SetupPreviewPanel />
      </Suspense>
    </CpScaffold>
  );
}
