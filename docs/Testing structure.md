Suggested split:
backend unit and module tests inside domain modules
integration tests near API and worker boundaries
frontend component and feature tests under feature directories
end-to-end tests in a top-level or app-level test suite later
Important priorities:
access logic
policy inheritance
source feed behavior
job execution behavior
provenance and audit behavior
retrieval scoping
AI context scoping
