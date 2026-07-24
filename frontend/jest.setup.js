/* eslint-disable @typescript-eslint/no-require-imports */
require('@testing-library/jest-dom');

// Mock IntersectionObserver only if window is defined (for jsdom environment)
if (typeof window !== 'undefined') {
  class IntersectionObserver {
    constructor(callback) {
      this.callback = callback;
    }
    observe() {
      return null;
    }
    unobserve() {
      return null;
    }
    disconnect() {
      return null;
    }
  }
  window.IntersectionObserver = IntersectionObserver;
}
