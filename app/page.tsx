export default function Home() {
  return (
    <main className="shell">
      <section className="intro">
        <p className="eyebrow">MVP execution layer</p>
        <h1>TokenGoblin</h1>
        <p>
          Deterministic token usage ingestion, cost estimates, anomaly signals,
          and tenant-scoped dashboard APIs.
        </p>
      </section>
      <section className="endpoints" aria-label="API endpoints">
        <h2>Tenant-Scoped APIs</h2>
        <ul>
          <li>
            <code>POST /api/ingest/token-usage</code>
          </li>
          <li>
            <code>GET /api/dashboard/overview</code>
          </li>
          <li>
            <code>GET /api/dashboard/workers</code>
          </li>
          <li>
            <code>GET /api/dashboard/anomalies</code>
          </li>
          <li>
            <code>GET /api/dashboard/events</code>
          </li>
        </ul>
      </section>
    </main>
  );
}
