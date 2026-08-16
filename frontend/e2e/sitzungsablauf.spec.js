import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

// Sitzungsverlust mitten in der Arbeit (Betreiber-Befund 16.08. abends): Nach
// Ablauf der 12h-Sitzung pollte das Badge in ein 401 nach dem anderen, die
// Konsole lief voll und die Seite wirkte kaputt — statt abgemeldet. Der zentrale
// Haken in apiFetch meldet den ersten unerwarteten 401 an den authStore: saubere
// Abmeldung, ein Toast, Anmeldemaske.
test('Abgelaufene Sitzung führt zur Anmeldung statt zu stillen Fehlern', async ({ page }) => {
	await uiLogin(page);

	// Der Sitzungstod: Cookies weg (wie nach Ablauf von Max-Age), App weiß nichts davon.
	await page.context().clearCookies();

	// Die nächste beliebige API-Berührung — hier der Medienkatalog — trifft den 401.
	await page.getByTitle('Medienkatalog').click();

	await expect(page.getByText('Sitzung abgelaufen — bitte neu anmelden.')).toBeVisible();
	await expect(page.locator('#login-email')).toBeVisible();
});
