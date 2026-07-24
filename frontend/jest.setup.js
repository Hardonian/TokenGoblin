/* eslint-disable @typescript-eslint/no-require-imports */
require('@testing-library/jest-dom');

// Mock IntersectionObserver
class IntersectionObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}
global.IntersectionObserver = IntersectionObserver;
