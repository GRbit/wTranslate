import { defineConfig } from 'vitest/config';

// Separate from vite.config.ts on purpose: the unit tests cover plain TS
// modules (stores + translate orchestration), so the svelte plugin is not
// needed and the tests run in a plain node environment.
export default defineConfig({
  test: {
    environment: 'node',
    include: ['src/**/*.test.ts'],
  },
});
