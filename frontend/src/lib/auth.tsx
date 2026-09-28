"use client";

import { useState, useEffect, createContext, useContext } from "react";
import { useRouter } from "next/navigation";

interface AuthContextType {
  tenantId: string | null;
  apiKey: string | null;
  login: (key: string, tenant: string) => void;
  logout: () => void;
  isLoading: boolean;
}

const AuthContext = createContext<AuthContextType>({
  tenantId: null,
  apiKey: null,
  login: () => {},
  logout: () => {},
  isLoading: true,
});

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [tenantId, setTenantId] = useState<string | null>(null);
  const [apiKey, setApiKey] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const router = useRouter();

  useEffect(() => {
    let active = true;
    void fetch("/api/tenant/session", { cache: "no-store" })
      .then((response) => response.json())
      .then((payload) => {
        if (!active || !payload?.data?.authenticated) return;
        setTenantId(payload.data.tenant_id);
        setApiKey("http-only-session");
      })
      .finally(() => {
        if (active) setIsLoading(false);
      });
    return () => {
      active = false;
    };
  }, []);

  const login = (_key: string, tenant: string) => {
    setApiKey("http-only-session");
    setTenantId(tenant);
  };

  const logout = () => {
    void fetch("/api/tenant/logout", { method: "POST" });
    setApiKey(null);
    setTenantId(null);
    router.push("/login");
  };

  return (
    <AuthContext.Provider value={{ tenantId, apiKey, login, logout, isLoading }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}

// Global fetcher to be used with SWR
export const authFetcher = async (url: string) => {
  const res = await fetch(url, {
    headers: { Accept: "application/json" },
    cache: "no-store",
  });
  const json = await res.json().catch(() => null);
  
  if (res.status === 401) {
    // Optionally trigger a logout or redirect here if unauthorized
    throw new Error("Unauthorized");
  }
  
  if (!res.ok || !json?.ok) {
    throw new Error(json?.error?.message || `Request failed (${res.status})`);
  }
  return json.data;
};
