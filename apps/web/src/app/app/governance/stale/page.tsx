import { redirect } from "next/navigation";

export default function LegacyAppGovernanceStaleRedirect() {
  redirect("/control-plane/governance/stale");
}
