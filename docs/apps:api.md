Main backend API service.
Suggested responsibilities:
auth/session handling
HTTP APIs
webhook endpoints
domain service orchestration
synchronous validation paths
control-plane operations
read APIs for UI surfaces
Suggested internal layout:


apps/api/
├─ cmd/
│  └─ api/
├─ internal/
│  ├─ auth/
│  ├─ http/
│  ├─ db/
│  ├─ identity_access/
│  ├─ knowledge_core/
│  ├─ ingestion_connectors/
│  ├─ retrieval_intelligence/
│  └─ knowledge_operations/
├─ migrations/
├─ configs/
└─ go.mod

Backend guidance
domain logic belongs in domain modules, not in handlers
handlers should stay thin
use explicit service interfaces inside modules
keep policy evaluation centralized
prefer clear package names over clever ones
