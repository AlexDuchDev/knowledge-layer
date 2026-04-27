# Control plane UI information architecture (index)

**Canonical long-form IA:** [control-plane-ui-ia.md](control-plane-ui-ia.md) — **all substantive CP navigation and screen inventory lives there.** This file is an intentional short stub (links + policy); do not duplicate long sections here. Product + dash IA: [INFORMATION_ARCHITECTURE_V1.md](INFORMATION_ARCHITECTURE_V1.md). Dedup policy: [ADMIN_UI_CONSOLIDATION_PLAN.md](ADMIN_UI_CONSOLIDATION_PLAN.md) §3.1.

**Overview:** [CONTROL_PLANE_OVERVIEW.md](CONTROL_PLANE_OVERVIEW.md).

**Admin URL policy:** Primary nav and docs use `**/control-plane/*`** as the canonical admin prefix. Legacy `**/admin/*`** redirects (308) to the matching control-plane path (including `**/admin/job-runs/:id**` → `**/control-plane/jobs/runs/:id**`); list and some detail URLs then **rewrite** to existing dash implementations so full builders stay usable until CP pages subsume them — see [ADMIN_UI_CONSOLIDATION_PLAN.md](ADMIN_UI_CONSOLIDATION_PLAN.md) and `apps/web/next.config.ts` + `apps/web/src/middleware.ts`.