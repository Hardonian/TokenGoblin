1. **Modify `StripeSyncer` in `internal/billing/stripe.go` for Testability**
   - Add `interval time.Duration` field to `StripeSyncer`.
   - Update `NewStripeSyncer` to initialize `interval` to `1 * time.Hour`.
   - Expose a way to override it for tests, e.g., `WithInterval` option or just add `interval` parameter, or modify `interval` directly if we put the tests in the `billing` package.
   - Update `Start` to use `s.interval`.

2. **Create Mock Storage Repository**
   - Use `UnavailableRepository` from `storage` or a custom mock in the test file that satisfies `storage.Repository`. A simple mock like `mockRepository` embedding `*storage.UnavailableRepository` and overriding necessary methods if needed, but right now `syncAllTenants` is mostly a stub that just logs. Wait, `StripeSyncer.syncAllTenants` does:
     ```go
     // tenants, _ := s.repo.ListTenants(ctx)
     ```
     So we don't necessarily need to mock the full logic, just provide a valid repository (like `storage.NewUnavailableRepository(nil)` or our own dummy) so it doesn't crash if someone uncommented it.

3. **Add Tests in `internal/billing/stripe_test.go`**
   - **Context Cancellation:** Test that calling `Start(ctx)` with a canceled context immediately returns.
   - **Ticker Mechanics:** Set `s.interval` to `10 * time.Millisecond`, pass a context with a timeout, and verify via logs or a custom mock that `syncAllTenants` is triggered. Since it uses `slog`, we can use `slog.New` with a custom handler or buffer to verify it logged "Starting Stripe usage sync".

4. **Run tests to verify**
   - `go test ./internal/billing/...`

5. **Complete pre-commit steps**

6. **Submit PR**
