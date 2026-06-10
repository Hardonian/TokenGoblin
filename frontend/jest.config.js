module.exports = {
  testEnvironment: 'jsdom',
  setupFilesAfterEnv: [],
  transform: {},
  testMatch: ['<rootDir>/src/**/*.test.{js,jsx,ts,tsx}'],
  moduleFileExtensions: ['ts', 'tsx', 'js', 'jsx', 'json', 'node'],
  testPathIgnorePatterns: ['/node_modules/'],
};
