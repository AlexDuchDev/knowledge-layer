/** Explains what preview screens represent (configuration intent vs live runtime). */
export function PreviewGuidance({
  kind,
}: {
  kind: "role" | "scenario" | "job" | "preset" | "setup";
}) {
  const copy: Record<"role" | "scenario" | "job" | "preset" | "setup", string> = {
    role:
      "Preview shows how this role template combines permissions and domain scope before you assign it to users. Assignments can still narrow or expand effective access per person.",
    scenario:
      "Preview summarizes bindings and timing: which roles, feeds, and jobs participate. Saving does not run jobs—it defines the scenario for future triggers.",
    job:
      "Preview reflects triggers, source scope, and output policy as configured. A successful preview does not enqueue a run; use test/run screens for execution.",
    preset:
      "Preview shows what will be created when you instantiate this preset: starter fields and relationships. You can edit the live objects after creation.",
    setup:
      "Launch preview estimates domains, presets, and objects affected by this setup session. Confirm before applying; some steps may still be audit-only depending on API mode.",
  };
  return (
    <div className="mt-4 rounded-md border border-amber-100 bg-amber-50/80 px-3 py-2 text-sm text-amber-950">
      <strong className="font-medium">What this preview means:</strong> {copy[kind]}
    </div>
  );
}
