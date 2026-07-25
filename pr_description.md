🎯 What: Added unit tests for `buildCheckoutSuccessURL` and `buildCheckoutCancelURL` in `frontend/src/lib/billing.ts` to cover the untested functions.
📊 Coverage: Tested normal origins without a trailing slash, origins with a trailing slash, and URL encoding for plans with special characters. Fixed existing tests in frontend/src/app/__tests__/page.test.tsx that were failing because of IntersectionObserver and text match mismatch.
✨ Result: Successfully filled the test gap ensuring checkout URL builders will remain correct. All front-end tests and backend tests pass.
