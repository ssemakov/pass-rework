import { defineConfig } from "@playwright/test";
export default defineConfig({
  testDir: "tests",
  workers: 1,
  timeout: 60000,
  use: {
    baseURL: "http://127.0.0.1:4173",
    headless: true,
    launchOptions: {
      executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
      // Compare legacy/new text with the same grayscale rasterization. Linux
      // LCD antialiasing otherwise changes with compositor-layer placement.
      args: ["--disable-lcd-text"],
    },
  },
  webServer: {
    command: "cd .. && go run -tags e2e ./cmd/e2e",
    url: "http://127.0.0.1:4173",
    reuseExistingServer: false,
    timeout: 120000,
  },
});
