import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

// „Abmelden" muss auch gelten, wenn der Server die Abmeldung nicht erhält.
//
// Nachgestellt am 14.09.2026: handleLogout schickt POST /api/auth/logout ab, verwirft jeden
// Fehler und zeigt sofort die Anmeldung. Das Löschcookie setzt aber nur die Antwort des
// Servers (api/logout_handler.go). Ohne Netz kommt keine an: Das HttpOnly-Cookie bleibt im
// Browser, und nach dem nächsten Neuladen mit Netz meldet restoreSession die vorige Person
// wieder an. An einem geteilten Theken-Rechner arbeitet dann der Nächste unter fremdem Konto.
//
// Die Offline-Schaltung liegt auf dem Browser-Kontext. context.route fängt die Abmeldung
// nicht ab: Sie geht mit keepalive hinaus, und solche Anfragen umgehen die Routen von
// Playwright (am 14.09.2026 gemessen: 0 Treffer, der Server bekam die Abmeldung trotzdem).
test('Abmelden ohne Netz: nach dem Neuladen mit Netz ist niemand angemeldet', async ({
	page,
	context
}) => {
	await uiLogin(page);

	await context.setOffline(true);
	await page.getByRole('button', { name: 'Abmelden' }).click();
	await expect(page.getByRole('button', { name: 'Anmelden' })).toBeVisible();
	await context.setOffline(false);

	await page.reload();
	await expect(page.getByRole('button', { name: 'Anmelden' })).toBeVisible();
	// Pollend: Der Boot braucht einen Moment, bis er die Sitzung geprüft hat.
	await expect
		.poll(async () => (await context.request.get('/api/auth/me')).status(), {
			message:
				'Die Sitzung gilt nach dem Abmelden weiter — das Neuladen meldet die vorige Person an',
			timeout: 5000
		})
		.toBe(401);
	await expect(page.getByRole('button', { name: 'Abmelden' })).toHaveCount(0);
});
