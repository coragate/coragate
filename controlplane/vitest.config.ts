import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    include: ["i18n/**/*.test.ts", "lib/**/*.test.ts"],
  },
});
