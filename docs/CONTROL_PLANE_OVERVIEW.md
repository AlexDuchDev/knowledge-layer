# Control plane overview

The **control plane** is where operators configure the platform: identity, sources, automation, and presets. It complements the **user-facing product** (search, ask, explorer, governance queues).

**One product shell:** End users and reviewers live in the root **dash** layout (`/`, `/search`, `/ask`, `/governance`, …). The control plane is a second **layout** for operator URLs under `/control-plane/`*; it is not a separate “Knowledge app.” Deprecated `/app/*` URLs redirect into the dash shell—see [INFORMATION_ARCHITECTURE_V1.md](./INFORMATION_ARCHITECTURE_V1.md).

## Main areas (web routes)


| Area           | Typical routes                | Purpose                                                                                               |
| -------------- | ----------------------------- | ----------------------------------------------------------------------------------------------------- |
| **Setup**      | `/control-plane/setup/`*      | First-run **setup sessions**, templates, launch preview, and launch result over real onboarding APIs. |
| **Roles**      | `/control-plane/roles/`*      | **Role definitions** catalog; assignments on separate screens.                                        |
| **Scenarios**  | `/control-plane/scenarios/`*  | **Scenario** definitions; bindings to roles, sources, jobs.                                           |
| **Jobs**       | `/control-plane/jobs/`*       | **Knowledge job** definitions, preview, test, runs.                                                   |
| **Sources**    | `/control-plane/sources/`*    | **Connectors** vs **source feeds**; health and sync.                                                  |
| **Presets**    | `/control-plane/presets/`*    | **Preset catalog**; instantiate into editable objects.                                                |
| **Governance** | `/control-plane/governance/`* | Operational queues: reviews, approvals, failures, stale.                                              |
| **Users**      | `/control-plane/users/`*      | User directory and access views.                                                                      |


Legacy **admin** builders under `/admin/`* may still exist alongside; prefer control-plane routes for new work.

## How this maps to concepts

- **Connector** (type) ≠ **Source feed** (instance) — see [GLOSSARY.md](GLOSSARY.md).
- **Scenario** orchestrates **when** things run; **Job** is the executable definition.
- **Preset** accelerates creation; it is not the live governed object until instantiated.
- **Setup session** is now a persisted operator workflow: create a session, apply a template, preview the planned roles/scenarios/jobs, then launch instantiated objects.

## Deep dives

- IA detail: [CONTROL_PLANE_UI_IA.md](CONTROL_PLANE_UI_IA.md) (canonical: [control-plane-ui-ia.md](control-plane-ui-ia.md)).
- Builders: [role-builder.md](role-builder.md), [scenario-builder.md](scenario-builder.md), [job-builder.md](job-builder.md).
- Onboarding: [onboarding-setup-flow.md](onboarding-setup-flow.md), [preset-catalog.md](preset-catalog.md).