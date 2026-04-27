import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";

/**
 * Rewrites selected /control-plane/* detail URLs to dash implementations while
 * keeping the canonical URL in the browser. As native CP pages ship the
 * matching rewrite is removed; once Jobs detail and Setup session are ported
 * this middleware can be deleted entirely (ADMIN_UI_CONSOLIDATION_PLAN.md Phase 4).
 *
 * Native so far: Roles (2.1.1), Scenarios (2.1.2), Presets (2.1.4).
 */
export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  const job = /^\/control-plane\/jobs\/([^/]+)$/.exec(pathname);
  if (job && job[1] !== "new" && job[1] !== "runs") {
    return NextResponse.rewrite(new URL(`/jobs/${job[1]}`, request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/control-plane/jobs/:path*"],
};
