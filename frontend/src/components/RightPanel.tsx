import { Panel, Badge, Button, formatMoney, formatInt } from "@/components/shared";

interface RightPanelProps {
  degraded: any[];
  recommendations: any;
  analyses: any;
  members: any;
  auditEvents: any;
  busyAction: string;
  setRecommendationStatus: (recommendationID: string, status: string) => Promise<void>;
  runAction: (path: string, method: "POST" | "DELETE") => Promise<void>;
}

export function RightPanel({
  degraded,
  recommendations,
  analyses,
  members,
  auditEvents,
  busyAction,
  setRecommendationStatus,
  runAction,
}: RightPanelProps) {
  return (
    <aside className="space-y-5">
      <Panel title="Degraded States">
        {degraded.length === 0 ? (
          <p className="text-sm text-[#52604e]">No degraded states reported by the API.</p>
        ) : (
          <div className="space-y-2">
            {degraded.map((issue: any, index: number) => (
              <p key={`${issue.code}-${index}`} className="border border-[#e0e4d8] p-3 text-sm">
                <span className="font-medium">{issue.code}</span>: {issue.message}
              </p>
            ))}
          </div>
        )}
      </Panel>

      <Panel title="Efficiency Wins">
        <div className="space-y-3">
          {(recommendations?.data ?? []).map((rec: any) => (
            <div key={rec.recommendation_id} className="border border-[#e0e4d8] p-3">
              <p className="text-sm font-medium">{rec.task_category}</p>
              <p className="mt-1 text-sm text-[#52604e]">{rec.reason}</p>
              <p className="mt-2 text-xs uppercase text-[#61705a]">
                {rec.confidence} confidence · {rec.evidence_count} events · {rec.status}
              </p>
              {rec.status_note && (
                <p className="mt-2 text-xs text-[#52604e]">{rec.status_note}</p>
              )}
              <div className="mt-3 flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={busyAction !== ""}
                  onClick={() =>
                    setRecommendationStatus(rec.recommendation_id, "accepted")
                  }
                >
                  Accept
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={busyAction !== ""}
                  onClick={() =>
                    setRecommendationStatus(rec.recommendation_id, "rejected")
                  }
                >
                  Reject
                </Button>
              </div>
            </div>
          ))}
        </div>
      </Panel>

      <Panel title="Goblin Score">
        <div className="space-y-3">
          {(analyses?.data ?? []).map((item: any) => (
            <div key={item.analysis_id} className="border border-[#e0e4d8] p-3">
              <div className="flex items-center justify-between">
                <p className="text-sm font-medium">{item.worker_id}</p>
                <p className="text-lg font-semibold">{item.goblin_score}</p>
              </div>
              <p className="mt-1 text-xs text-[#61705a]">event {item.event_id}</p>
              <p className="mt-2 text-sm text-[#52604e]">
                {item.issues[0]?.message || "No deterministic waste issue found."}
              </p>
            </div>
          ))}
        </div>
      </Panel>

      <Panel title="Team Boundary">
        <div className="space-y-2">
          {(members?.data ?? []).map((member: any) => (
            <div key={member.subject_id} className="border border-[#e0e4d8] p-3 text-sm">
              <p className="font-medium">{member.email || member.subject_id}</p>
              <p className="text-xs uppercase text-[#61705a]">{member.role}</p>
            </div>
          ))}
          {(members?.data ?? []).length === 0 && (
            <p className="text-sm text-[#52604e]">
              No explicit tenant members have been configured.
            </p>
          )}
        </div>
      </Panel>

      <Panel title="Audit Trail">
        <div className="space-y-2">
          {(auditEvents?.data ?? []).map((event: any) => (
            <div key={event.event_id} className="border border-[#e0e4d8] p-3 text-sm">
              <p className="font-medium">{event.type}</p>
              <p className="text-[#52604e]">
                {event.actor} · {event.resource || "tenant"} ·{" "}
                {new Date(event.timestamp).toLocaleString()}
              </p>
            </div>
          ))}
          {(auditEvents?.data ?? []).length === 0 && (
            <p className="text-sm text-[#52604e]">
              No persisted audit events yet.
            </p>
          )}
        </div>
      </Panel>

      <Button
        variant="danger"
        size="md"
        className="w-full"
        disabled={busyAction !== ""}
        onClick={() => runAction("/api/dashboard/reset", "DELETE")}
      >
        Reset Tenant Data
      </Button>
    </aside>
  );
}