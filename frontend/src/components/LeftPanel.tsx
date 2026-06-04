import { Panel, Badge, formatMoney, formatInt } from "@/components/shared";

interface LeftPanelProps {
  summary: any;
  workers: any;
  events: any;
}

export function LeftPanel({ summary, workers, events }: LeftPanelProps) {
  const empty = summary?.data?.total_events === 0;

  return (
    <div className="space-y-5">
      {empty && (
        <div className="border border-[#d7dccf] bg-white p-5">
          <h2 className="text-lg font-semibold">No review data yet</h2>
          <p className="mt-2 text-sm leading-6 text-[#52604e]">
            Ingest token usage events or seed demo data. Until records exist,
            TokenGoblin will not invent cost leaks, quality scores, or worker
            recommendations.
          </p>
        </div>
      )}

      <Panel title="Waste Signals">
        <div className="grid gap-3 md:grid-cols-2">
          {(summary?.data?.top_cost_drivers ?? []).map((driver: any) => (
            <div key={`${driver.type}:${driver.key}`} className="border border-[#e0e4d8] p-4">
              <p className="text-xs uppercase text-[#61705a]">{driver.type}</p>
              <h3 className="mt-1 font-semibold">{driver.label}</h3>
              <p className="mt-2 text-sm text-[#52604e]">
                {formatMoney(driver.total_cost_usd)} estimated across{" "}
                {driver.event_count} events.
              </p>
            </div>
          ))}
        </div>
      </Panel>

      <Panel title="Worker Review">
        <div className="overflow-x-auto">
          <table className="w-full min-w-[680px] text-left text-sm">
            <thead className="border-b border-[#e0e4d8] text-xs uppercase text-[#61705a]">
              <tr>
                <th className="py-2">Worker</th>
                <th>Tokens</th>
                <th>Cost</th>
                <th>Outputs</th>
                <th>Waste</th>
                <th>Trend</th>
              </tr>
            </thead>
            <tbody>
              {(workers?.data ?? []).map((worker: any) => (
                <tr key={worker.worker_id} className="border-b border-[#edf0e8]">
                  <td className="py-3 font-medium">{worker.worker_name}</td>
                  <td>{formatInt(worker.total_tokens)}</td>
                  <td>${formatMoney(worker.total_cost_usd)}</td>
                  <td>{worker.output_count}</td>
                  <td>{worker.unknown_cost_event_count} unknown-cost events</td>
                  <td>{worker.trend || worker.efficiency_rating || "stable"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Panel>

      <Panel title="Recent Review Runs">
        <div className="space-y-2">
          {(events?.data ?? []).map((event: any) => (
            <div key={event.event_id} className="flex flex-wrap items-center justify-between gap-3 border border-[#e0e4d8] p-3 text-sm">
              <div>
                <p className="font-medium">{event.worker_name || event.worker_id}</p>
                <p className="text-[#52604e]">{event.provider}:{event.model_id} · {event.task_category || "uncategorized"}</p>
              </div>
              <div className="text-right">
                <p>{formatInt(event.total_tokens)} tokens</p>
                <p className="text-[#52604e]">
                  {event.cost_estimate_usd == null ? "Cost unknown" : `$${formatMoney(event.cost_estimate_usd)} estimated`}
                </p>
              </div>
            </div>
          ))}
        </div>
      </Panel>
    </div>
  );
}