Frontend application.

**Route shells (scaffold):** `/control-plane/*` (operators) and `/app/*` (end-user knowledge app) live beside the legacy `(dash)` tree; see [control-plane-ui-ia.md](./control-plane-ui-ia.md), [user-facing-product-surface.md](./user-facing-product-surface.md), and [design-system-and-page-templates.md](./design-system-and-page-templates.md).

Suggested responsibilities:
admin UI
governance UI
search UI
Q&A UI
review / approval UI
source feed setup
job management
entity detail pages
Suggested internal layout:
apps/web/
├─ src/
│  ├─ app/
│  ├─ components/
│  ├─ features/
│  │  ├─ auth/
│  │  ├─ users-access/
│  │  ├─ source-feeds/
│  │  ├─ entities/
│  │  ├─ knowledge-jobs/
│  │  ├─ governance/
│  │  ├─ search/
│  │  └─ qa/
│  ├─ lib/
│  ├─ hooks/
│  ├─ styles/
│  └─ types/
├─ public/
└─ package.json

Frontend guidance
organize by feature, not by generic component buckets only
keep API contracts typed
keep trust indicators and workflow state visible in UI models
avoid pushing policy logic into the browser
