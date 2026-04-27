/** API base; optional dev header (NEXT_PUBLIC_USE_DEV_HEADER) vs session cookies (credentials). */

const DEFAULT_API = "http://localhost:8080";
const DEFAULT_PRINCIPAL = "30000000-0000-0000-0000-000000000001"; // admin@local.test seed

export function apiBase(): string {
  return process.env.NEXT_PUBLIC_API_URL ?? DEFAULT_API;
}

export function principalUserId(): string {
  return process.env.NEXT_PUBLIC_PRINCIPAL_USER_ID ?? DEFAULT_PRINCIPAL;
}

/** When true, sends X-Principal-User-ID (local dev with AUTH_MODE=development_header). */
export function isDevPrincipalHeader(): boolean {
  return process.env.NEXT_PUBLIC_USE_DEV_HEADER === "true";
}

export function apiHeaders(): HeadersInit {
  const h: Record<string, string> = {
    Accept: "application/json",
    "Content-Type": "application/json",
  };
  if (isDevPrincipalHeader()) {
    h["X-Principal-User-ID"] = principalUserId();
  }
  return h;
}

/** Maps browser network failures to a short hint (wrong NEXT_PUBLIC_API_URL vs compose, API down, etc.). */
export function formatApiClientError(e: unknown): string {
  const raw = e instanceof Error ? e.message : String(e);
  const low = raw.toLowerCase();
  if (low.includes("failed to fetch") || low.includes("networkerror") || low.includes("load failed")) {
    return `Could not reach the API at ${apiBase()}. Is it running and reachable from the browser? With Docker Compose the API defaults to port 18080 — rebuild the web image so NEXT_PUBLIC_API_URL matches: docker compose build web && docker compose up -d web.`;
  }
  return raw;
}

export async function apiJson<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${apiBase()}${path}`, {
    credentials: "include",
    ...init,
    headers: {
      ...(apiHeaders() as Record<string, string>),
      ...(init?.headers as Record<string, string> | undefined),
    },
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`${res.status} ${text.slice(0, 400)}`);
  }
  if (res.status === 204) {
    return undefined as T;
  }
  return res.json() as Promise<T>;
}
