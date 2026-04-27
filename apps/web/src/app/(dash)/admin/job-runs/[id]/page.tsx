"use client";

import { useParams } from "next/navigation";
import { AppBreadcrumb } from "@/components/AppBreadcrumb";
import { JobRunDetailClient } from "@/components/jobs/JobRunDetailClient";

export default function AdminJobRunDetailPage() {
  const params = useParams<{ id: string }>();
  const id = params?.id ?? "";

  return (
    <main className="mx-auto max-w-3xl px-6 py-10">
      <AppBreadcrumb
        items={[
          { label: "Home", href: "/" },
          { label: "Knowledge jobs", href: "/control-plane/jobs" },
          { label: "Job run" },
        ]}
      />
      <h1 className="text-2xl font-semibold tracking-tight">Job run</h1>
      <JobRunDetailClient runId={id} footerBackHref="/control-plane/jobs" footerBackLabel="Back to jobs" />
    </main>
  );
}
