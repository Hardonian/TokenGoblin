1. **Identify the root cause**:
   - The CI check "build-test" is failing at the `golangci-lint` step.
   - The log says: `Error: can't load config: the Go language version (go1.23) used to build golangci-lint is lower than the targeted Go version (1.25.0)`
   - The action `golangci/golangci-lint-action@v6` with version `v1.61` is using a binary that is compiled for go1.23, but our project's `go.mod` has probably `go 1.25.0` or higher, which requires a newer `golangci-lint` version.

2. **Fix the GitHub Action**:
   - Update `.github/workflows/ci.yml` (or similar file name) to use a newer version of `golangci-lint` that supports `go 1.25.0`. According to golangci-lint releases, `v1.64` or `latest` is appropriate. Let's set version to `v1.64` or `latest`.

3. **Verify locally**:
   - Ensure the tests pass locally, but since we are modifying github actions, we can't test it directly locally without running action. We will use a safe, recent version of `golangci-lint` like `v1.64` which is compatible.

4. **Commit the changes**.
