/* eslint-disable @typescript-eslint/no-require-imports */
require('@testing-library/jest-dom');

// Mock IntersectionObserver for tests
class MockIntersectionObserver {
  constructor() {}
  observe() {}
  unobserve() {}
  disconnect() {}
}
global.IntersectionObserver = MockIntersectionObserver;
