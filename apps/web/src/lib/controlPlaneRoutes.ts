/**
 * Minimum control-plane route set for tests and IA verification.
 * Keep in sync with app/control-plane/** and docs/control-plane-ui-ia.md.
 */
export const CONTROL_PLANE_ROUTE_PREFIX = "/control-plane";

export const REQUIRED_CONTROL_PLANE_PATHS: string[] = [
  "/control-plane/setup",
  "/control-plane/setup/templates",
  "/control-plane/setup/session/new",
  "/control-plane/setup/wizard",
  "/control-plane/setup/launch-preview",
  "/control-plane/setup/launch-result",
  "/control-plane/roles",
  "/control-plane/roles/new",
  "/control-plane/scenarios",
  "/control-plane/scenarios/new",
  "/control-plane/jobs",
  "/control-plane/jobs/new",
  "/control-plane/sources",
  "/control-plane/presets",
  "/control-plane/governance",
  "/control-plane/users",
];
