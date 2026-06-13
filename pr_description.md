🎯 **What:** This PR addresses a testing gap by adding unit tests for the `HashPrompt` function in `internal/intelligence/engine.go`. `HashPrompt` is a pure function responsible for normalizing and hashing strings, and was previously lacking explicit test coverage.

📊 **Coverage:** The new `TestHashPrompt` table-driven test covers:
- Basic hashing (happy path).
- Whitespace trimming (leading, trailing, and mixed).
- Lowercase conversion (case insensitivity).
- Empty string behavior.
- Strings containing only whitespace.
- Hash stability (same input consistently yields the same 64-character SHA-256 hex string).

✨ **Result:** Test coverage for `internal/intelligence/engine.go` is improved, documenting and enforcing the normalization behavior (trimming, lowercasing) of the `HashPrompt` utility function.
