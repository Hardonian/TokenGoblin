/* eslint-disable @typescript-eslint/no-require-imports */
require('@testing-library/jest-dom');

class IntersectionObserver {
  constructor() {}
  observe() {}
  unobserve() {}
  disconnect() {}
}

global.IntersectionObserver = IntersectionObserver;
