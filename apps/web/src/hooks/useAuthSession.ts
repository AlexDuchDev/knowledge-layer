"use client";

import { usePathname, useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { apiBase, apiJson, isDevPrincipalHeader } from "@/lib/api";
import type { NavigationVisibility } from "@/lib/navigation";

export type MeResponse = {
  id: string;
  email: string;
  name: string;
  navigation?: NavigationVisibility;
};

export function useAuthSession() {
  const pathname = usePathname();
  const router = useRouter();
  const [me, setMe] = useState<MeResponse | null>(null);
  const [navVis, setNavVis] = useState<NavigationVisibility | null>(null);
  const [authErr, setAuthErr] = useState<string | null>(null);

  const loadMe = useCallback(async () => {
    setAuthErr(null);
    try {
      const u = await apiJson<MeResponse>("/auth/me");
      setMe(u);
      setNavVis(u.navigation ?? null);
    } catch (e) {
      setMe(null);
      setNavVis(null);
      setAuthErr(e instanceof Error ? e.message : String(e));
    }
  }, []);

  useEffect(() => {
    void loadMe();
  }, [loadMe]);

  useEffect(() => {
    if (!authErr) return;
    if (authErr.includes("401") && !isDevPrincipalHeader()) {
      router.replace(`/login?next=${encodeURIComponent(pathname || "/")}`);
    }
  }, [authErr, pathname, router]);

  const logout = useCallback(async () => {
    try {
      await fetch(`${apiBase()}/auth/logout`, { method: "POST", credentials: "include", headers: { Accept: "application/json" } });
    } catch {
      /* ignore */
    }
    router.replace("/login");
  }, [router]);

  return { me, navVis, authErr, loadMe, logout };
}
