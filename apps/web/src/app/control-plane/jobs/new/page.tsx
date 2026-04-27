import { CpScaffold } from "@/components/control-plane/CpScaffold";

export default function CreateJobFromPresetPage() {
  return (
    <CpScaffold
      title="Create job from preset"
      description="Pick a job preset, then configure scope, trigger, and output policy."
      screenType="Builder"
      guidanceSlug="jobs"
      crossLinks={[
        { href: "/control-plane/presets", label: "Preset catalog" },
        { href: "/control-plane/jobs/new/custom", label: "Custom instead" },
        { href: "/control-plane/jobs", label: "Job list" },
      ]}
    />
  );
}
