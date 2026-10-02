import { defineConfig, devices } from '@playwright/test';

// E2E-Smoke-Tests gegen den lokalen Docker-Stack (Fahrplan Phase 2, T5).
// Voraussetzung: docker compose -f docker-compose.local.yml up -d --build
// (Backend inkl. gebautem Frontend auf :8084, Mock-IMAP akzeptiert jedes Passwort.)
export default defineConfig({
	testDir: './e2e',
	// Merkt sich den Hauptlieferanten und räumt die von den Flows angelegten
	// Testlieferanten wieder ab. Ohne das wuchs die Lieferantenliste mit jedem Lauf
	// (zuletzt 224 Testeinträge gegen 3 echte) und die Rolle des Hauptlieferanten war
	// nach jedem Lauf verschwunden.
	globalSetup: './e2e/global-setup.js',
	globalTeardown: './e2e/global-teardown.js',
	timeout: 30_000,
	fullyParallel: false, // Flows teilen sich eine DB — seriell bleiben
	workers: 1,
	retries: 0,
	// Ein vergessenes test.only lässt die anderen 100 Tests still ausfallen und meldet
	// trotzdem "passed". Lokal bleibt .only erlaubt (man debuggt damit), in CI nicht.
	forbidOnly: !!process.env.CI,
	reporter: [['list']],
	use: {
		// Maßstab ist der gebaute Stand. Am Entwicklungsserver (E2E_BASE_URL=http://localhost:5173)
		// braucht die Barrierefreiheits-Prüfung großer Seiten ein Mehrfaches der Zeit; dort
		// `--timeout=300000` mitgeben.
		baseURL: process.env.E2E_BASE_URL || 'http://localhost:8084',
		trace: 'retain-on-failure',
		screenshot: 'only-on-failure'
	},
	projects: [
		{
			name: 'chromium',
			use: { ...devices['Desktop Chrome'] }
		}
	]
});
