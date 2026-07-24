💡 **What:**
Replaced the `SaveAnomalySignal` call in a `for` loop with a new bulk saving method `SaveAnomalySignals` within the `ingestion.Service`. This required updating the `storage.Repository` interface and implementing the new bulk method in both the SQLite and Postgres storage layers. SQLite uses a transaction with a prepared statement, and Postgres uses `pgx.Batch` to achieve maximum performance.

🎯 **Why:**
The previous implementation iterated over anomaly signals and executed an `INSERT` statement for each signal sequentially, resulting in an N+1 query problem. This degraded performance linearly as the number of detected anomalies in a batch increased, leading to unnecessary database round trips, CPU cycles, and lock contention.

📊 **Measured Improvement:**
A benchmark was introduced (`BenchmarkSaveAnomalySignals_NPlus1` vs `BenchmarkSaveAnomalySignals_Bulk`) to measure the performance impact of inserting 100 anomaly signals per batch into an in-memory SQLite database.
- **Baseline (N+1)**: ~4.91 ms / op
- **Improvement (Bulk)**: ~1.29 ms / op
- **Change over baseline**: 3.8x faster (73% reduction in execution time)
