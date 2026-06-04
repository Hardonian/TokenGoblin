import Link from "next/link";
import { Button } from "@/components/shared";

interface DashboardHeaderProps {
  tenant: string;
  setTenant: (tenant: string) => void;
  busyAction: string;
  runAction: (path: string, method: "POST" | "DELETE") => Promise<void>;
  apiBase: string;
}

export function DashboardHeader({
  tenant,
  setTenant,
  busyAction,
  runAction,
  apiBase,
}: DashboardHeaderProps) {
  return (
    <section className="border-b border-[#d7dccf] bg-[#fbfcf8]">
      <div className="mx-auto flex w-full max-w-7xl flex-col gap-5 px-5 py-6 md:flex-row md:items-end md:justify-between">
        <div>
          <p className="text-sm font-semibold uppercase tracking-[0.16em] text-[#61705a]">
            TokenGoblin
          </p>
          <h1 className="mt-2 text-3xl font-semibold tracking-normal md:text-5xl">
            Token efficiency review
          </h1>
          <p className="mt-3 max-w-2xl text-sm leading-6 text-[#52604e]">
            Estimated spend, output bloat, worker patterns, and deterministic
            recommendations from persisted usage evidence.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <input
            className="h-10 w-44 border border-[#c5cdbb] bg-white px-3 text-sm outline-none focus:border-[#426b51]"
            value={tenant}
            onChange={(event) => setTenant(event.target.value)}
            aria-label="Tenant ID"
          />
          <Button
            variant="primary"
            size="md"
            disabled={busyAction !== ""}
            onClick={() => runAction("/api/dashboard/seed", "POST")}
          >
            Seed Demo
          </Button>
          <a
            className="flex h-10 items-center border border-[#c5cdbb] bg-white px-4 text-sm font-medium"
            href={`${apiBase}/api/dashboard/report.md`}
            target="_blank"
            rel="noopener noreferrer"
          >
            Report
          </a>
        </div>
      </div>
    </section>
  );
}