import { defineConfig, devices } from '@playwright/test';

// Proben von Hand — nicht Teil der E2E-Suite.
//
// Hier liegt, was eine Vorbereitung braucht, die kein Rechner nebenbei hat: die
// Kamera-Probe etwa fährt ein 120 MB grosses Kamerabild auf, das scripts/kamera_probe.sh
// vorher mit ffmpeg erzeugt. In e2e/ wäre sie entweder ein Test, der sich still
// überspringt (und damit nie rot wird), oder einer, der die Suite auf jedem Rechner ohne
// ffmpeg rot färbt. Beides ist schlechter als ein eigener Ort mit einem Skript davor.
export default defineConfig({
	testDir: './e2e-proben',
	timeout: 120_000,
	workers: 1,
	reporter: [['list']],
	use: {
		baseURL: process.env.E2E_BASE_URL || 'http://localhost:8084'
	},
	projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }]
});
