import { defineConfig, devices } from '@playwright/test';

// E2E-Smoke-Tests gegen den lokalen Docker-Stack (Fahrplan Phase 2, T5).
// Voraussetzung: docker compose -f docker-compose.local.yml up -d --build
// (Backend inkl. gebautem Frontend auf :8084, Mock-IMAP akzeptiert jedes Passwort.)
export default defineConfig({
    testDir: './e2e',
    timeout: 30_000,
    fullyParallel: false, // Flows teilen sich eine DB — seriell bleiben
    workers: 1,
    retries: 0,
    reporter: [['list']],
    use: {
        baseURL: process.env.E2E_BASE_URL || 'http://localhost:8084',
        trace: 'retain-on-failure',
        screenshot: 'only-on-failure',
    },
    projects: [
        {
            name: 'chromium',
            use: { ...devices['Desktop Chrome'] },
        },
    ],
});
