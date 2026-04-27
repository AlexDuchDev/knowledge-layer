/** Response shape for GET /instance/status (extends over time; unknown fields ignored). */
export type InstanceStatus = {
  needs_bootstrap: boolean;
  domain_count: number;
  auth_mode: string;
  build_version?: string;
  source_feed_total_count?: number;
  source_feed_active_count?: number;
  has_successful_source_sync?: boolean;
};
