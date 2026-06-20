import { useState, useEffect } from "react";

/**
 * A mock auth hook for the TokenGoblin frontend.
 * In a real application, this would interface with a provider like NextAuth, Clerk, or Firebase.
 */
export function useAuth() {
  const [tenantId, setTenantId] = useState<string>("tenant_123");
  const [apiKey, setApiKey] = useState<string>("dev_key_456");

  useEffect(() => {
    // Attempt to load from local storage if available, to allow testing different tenants
    const storedTenant = localStorage.getItem("tg_tenant_id");
    const storedKey = localStorage.getItem("tg_api_key");
    if (storedTenant) setTenantId(storedTenant);
    if (storedKey) setApiKey(storedKey);
  }, []);

  return {
    tenantId,
    apiKey,
  };
}
