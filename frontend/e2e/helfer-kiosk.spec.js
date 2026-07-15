import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, seedSQL } from './helpers.js';

// Die Rolle "helfer" war lange unerreichbar: Router und Rechtesystem kannten sie,
// das benutzer_rolle-ENUM aber nicht — sie liess sich niemand zuweisen (behoben mit
// Migration 042). Damit hatte der Kiosk-Modus für Hilfskräfte nie einen Test gesehen.
//
// Diese Spec sichert das ab, was die Rolle ausmacht: Sie ist vergebbar, sie führt in
// den Kiosk, und sie kommt an nichts anderes heran.
//
// Bewusst NICHT hier geprüft: ob ein Helfer im Kiosk scannen darf. Die geseedeten
// Rechte stehen derzeit alle auf false (db/seed.go), womit jeder Scan 403 liefert.
// Ob eine Hilfskraft Schülerdaten sehen darf, ist eine fachliche und
// datenschutzrechtliche Entscheidung des Betreibers — kein Test darf sie vorwegnehmen,
// in keine der beiden Richtungen.

const HELFER_EMAIL = 'e2e-helfer@test.local';

function seedHelfer() {
	seedSQL(`
        INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
        VALUES ('E2E', 'Helfer', '${HELFER_EMAIL}', 'helfer', true)
        ON CONFLICT DO NOTHING;
    `);
}

test('Helfer: Rolle ist vergebbar und landet im Kiosk', async ({ page }) => {
	// Das INSERT selbst ist der Test für Migration 042: Ohne den ENUM-Wert schlägt
	// bereits das Seeding fehl.
	seedHelfer();

	await uiLogin(page, HELFER_EMAIL);

	// Router.svelte zwingt Helfer auf den Kiosk — unabhängig davon, wo sie landen.
	await expect(page).toHaveURL(/\/(kiosk)?$/);

	// Die Omnibox ist die Kiosk-Oberfläche.
	await expect(page.getByRole('button', { name: 'Abmelden' })).toBeVisible();
});

test('Helfer: kommt weder per UI noch per API an fremde Bereiche', async ({ page }) => {
	seedHelfer();
	await uiLogin(page, HELFER_EMAIL);

	// Server blockt die Bereiche, die dem Helfer nicht gehören.
	const abgaenger = await page.request.get('/api/abgaenger');
	expect(abgaenger.status(), 'Abgänger-API für Helfer').toBe(403);

	const backup = await page.request.get('/api/admin/system/backup-status');
	expect(backup.status(), 'Backup-Status für Helfer').toBe(403);

	const createUser = await apiPost(page, '/api/benutzer', {
		vorname: 'Boese',
		nachname: 'Absicht',
		email: 'helfer-eskalation@test.local',
		rolle: 'admin'
	});
	expect(createUser.status(), 'Benutzer anlegen als Helfer').toBe(403);

	// Direkter URL-Aufruf umgeht die Router-Weiche nicht: keine Daten, kein Absturz.
	await page.goto('/schuelerdatei');
	await expect(page.getByText('Barcode-ID')).toHaveCount(0);
});
