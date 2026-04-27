import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { PolicyExceptionsQueueClient } from "@/components/control-plane/GovernanceQueueClients";

export default function PolicyExceptionsPage() {
  return (
    <CpScaffold
      title="Policy exceptions"
      description="Active exceptions that override inherited access policy on a specific resource. Review periodically; mutations (create / approve / revoke) live in the entity workflow tools."
      screenType="Operational"
      guidanceSlug="governance"
      crossLinks={[
        { href: "/control-plane/governance/approvals", label: "Approvals" },
        { href: "/control-plane/roles", label: "Roles" },
        { href: "/control-plane/governance", label: "Governance dashboard" },
      ]}
    >
      <PolicyExceptionsQueueClient />
    </CpScaffold>
  );
}
