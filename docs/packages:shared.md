Shared contracts and utilities.
Suggested contents:
API schemas
shared types
event payload shapes
frontend/backend contract definitions
shared validation utilities
maybe generated clients
Suggested layout:


packages/shared/
├─ src/
│  ├─ api/
│  ├─ schemas/
│  ├─ events/
│  ├─ enums/
│  └─ utils/
└─ package.json

Guidance
keep shared package small and durable
do not turn it into a dumping ground
share contracts, not random convenience code
