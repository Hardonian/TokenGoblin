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
    const storedTenant = localStorage.getItem("tg_tenant_id");
    const storedKey = localStorage.getItem("tg_api_key");
    if (storedTenant) setTenantId(storedTenant);
    if (storedKey) setApiKey(storedKey);
    setIsLoading(false);
  }, []);

  const login = (key: string, tenant: string) => {
    localStorage.setItem("tg_api_key", key);
    localStorage.setItem("tg_tenant_id", tenant);
    setApiKey(key);
    setTenantId(tenant);
  };

  const logout = () => {
    localStorage.removeItem("tg_api_key");
    localStorage.removeItem("tg_tenant_id");
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
  const token = localStorage.getItem("tg_api_key");
  const tenant = localStorage.getItem("tg_tenant_id");
  
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }
  if (tenant) {
    headers["x-tenant-id"] = tenant;
  }

  const res = await fetch(url, { headers });
  const json = await res.json();
  
  if (res.status === 401) {
    // Optionally trigger a logout or redirect here if unauthorized
    throw new Error("Unauthorized");
  }
  
  if (!json.ok) throw new Error(json.error?.message || "Failed to fetch");
  return json.data;
};
