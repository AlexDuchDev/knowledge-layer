import type { NextConfig } from "next";
import path from "path";

const nextConfig: NextConfig = {
  outputFileTracingRoot: path.resolve(__dirname, "../.."),
  async redirects() {
    return [
      // Deprecate /app/* mirrors in favor of canonical dash routes (see docs/INFORMATION_ARCHITECTURE_V1.md).
      { source: "/app", destination: "/", permanent: true },
      { source: "/app/search", destination: "/search", permanent: true },
      { source: "/app/ask", destination: "/ask", permanent: true },
      { source: "/app/explorer", destination: "/entities", permanent: true },
      { source: "/app/projects", destination: "/projects", permanent: true },
      { source: "/app/projects/:path*", destination: "/projects/:path*", permanent: true },
      { source: "/app/decisions", destination: "/decisions", permanent: true },
      { source: "/app/decisions/:path*", destination: "/decisions/:path*", permanent: true },
      { source: "/app/governance/reviews", destination: "/reviews", permanent: true },
      { source: "/app/governance/approvals", destination: "/approvals", permanent: true },
      { source: "/app/governance", destination: "/governance", permanent: true },
      { source: "/app/governance/stale", destination: "/control-plane/governance/stale", permanent: true },
      { source: "/app/digests", destination: "/insights", permanent: true },
      { source: "/app/digests/:path*", destination: "/insights", permanent: true },
      { source: "/knowledge/decisions", destination: "/decisions", permanent: true },
      { source: "/knowledge/policies", destination: "/policies", permanent: true },
      { source: "/knowledge/processes", destination: "/processes", permanent: true },
      { source: "/knowledge/meetings", destination: "/meetings", permanent: true },
      { source: "/knowledge/insights", destination: "/insights", permanent: true },
      { source: "/knowledge/projects", destination: "/projects", permanent: true },
      // Canonical URLs use /control-plane; legacy /admin/* 308 to CP. CP list/detail routes rewrite to
      // dash implementations where control-plane pages are still scaffolds (see middleware + rewrites).
      { source: "/admin/access", destination: "/control-plane/users", permanent: true },
      { source: "/admin/audit", destination: "/audit", permanent: true },
      { source: "/admin/settings", destination: "/settings", permanent: true },
      { source: "/admin/ops/answer-diagnostics", destination: "/ops/answer-diagnostics", permanent: true },
      { source: "/admin/ops/search-insights", destination: "/ops/search-insights", permanent: true },
      { source: "/admin/roles", destination: "/control-plane/roles", permanent: true },
      { source: "/admin/scenarios", destination: "/control-plane/scenarios", permanent: true },
      { source: "/admin/scenarios/:path*", destination: "/control-plane/scenarios/:path*", permanent: true },
      { source: "/admin/presets", destination: "/control-plane/presets", permanent: true },
      { source: "/admin/presets/:path*", destination: "/control-plane/presets/:path*", permanent: true },
      { source: "/admin/setup", destination: "/control-plane/setup", permanent: true },
      { source: "/admin/setup/:sessionId", destination: "/control-plane/setup/session/:sessionId", permanent: true },
      { source: "/admin/source-feeds", destination: "/control-plane/sources", permanent: true },
      { source: "/admin/source-feeds/:path*", destination: "/control-plane/sources/feeds/:path*", permanent: true },
      { source: "/admin/connectors", destination: "/control-plane/sources/connectors", permanent: true },
      { source: "/admin/jobs", destination: "/control-plane/jobs", permanent: true },
      { source: "/admin/jobs/:path*", destination: "/control-plane/jobs/:path*", permanent: true },
      { source: "/admin/job-runs/:id", destination: "/control-plane/jobs/runs/:id", permanent: true },
      { source: "/admin/users", destination: "/control-plane/users", permanent: true },
      { source: "/admin/users/:path*", destination: "/control-plane/users/:path*", permanent: true },
      // Parity matrix (ADMIN_UI_CONSOLIDATION_PLAN.md §5): directory users live under canonical CP URLs.
      { source: "/access", destination: "/control-plane/users", permanent: true },
      // Any remaining /app/* path not covered above falls back to home (deprecated shell; non-permanent for safety).
      { source: "/app/:path*", destination: "/", permanent: false },
    ];
  },
  async rewrites() {
    return {
      beforeFiles: [
        { source: "/decisions", destination: "/knowledge/decisions" },
        { source: "/policies", destination: "/knowledge/policies" },
        { source: "/processes", destination: "/knowledge/processes" },
        { source: "/meetings", destination: "/knowledge/meetings" },
        { source: "/insights", destination: "/knowledge/insights" },
        { source: "/projects", destination: "/knowledge/projects" },
        // Serve legacy implementations under canonical /control-plane URLs (ADMIN_UI_CONSOLIDATION_PLAN.md).
        // Roles, Scenarios, and Presets ship natively (Phase 2.1.1, 2.1.2, 2.1.4) — rewrites removed.
        { source: "/control-plane/jobs", destination: "/jobs" },
        { source: "/control-plane/sources/connectors", destination: "/connectors" },
        { source: "/control-plane/users/:id", destination: "/admin/users/:id" },
        { source: "/control-plane/users", destination: "/access" },
      ],
    };
  },
};

export default nextConfig;
