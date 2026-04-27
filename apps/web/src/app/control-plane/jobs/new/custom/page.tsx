import { CpScaffold } from "@/components/control-plane/CpScaffold";

export default function CreateCustomJobPage() {
  return (
    <CpScaffold
      title="Create custom job"
      description="Job Builder without preset seed (scaffold)."
      screenType="Builder"
      guidanceSlug="jobs"
      crossLinks={[
        { href: "/control-plane/jobs/new", label: "From preset" },
        { href: "/control-plane/jobs", label: "Job list" },
      ]}
    />
  );
}
