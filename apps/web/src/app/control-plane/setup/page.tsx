import Link from "next/link";
import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { SetupHubClient } from "@/components/control-plane/SetupHubClient";

export default function ControlPlaneSetupHubPage() {
  return (
    <CpScaffold
      title="Setup"
      description="Session-based onboarding. Pick a template, toggle connector families, set the initial admin, preview the plan, then launch — the wizard creates roles, scenarios, and jobs in your instance."
      screenType="Setup"
      guidanceSlug="setup"
      crossLinks={[
        { href: "/bootstrap", label: "Instance bootstrap" },
        { href: "/control-plane/presets", label: "Preset catalog" },
        { href: "/control-plane/governance", label: "Governance dashboard" },
      ]}
    >
      <p className="mt-2 text-sm text-gray-600">
        Distinct from ongoing administration. Use{" "}
        <Link href="/bootstrap" className="text-blue-700 underline">
          /bootstrap
        </Link>{" "}
        when the instance has not been bootstrapped yet.
      </p>
      <SetupHubClient />
    </CpScaffold>
  );
}
