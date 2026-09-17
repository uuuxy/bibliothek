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
	//
	// Der Klick darf dabei auch ins Leere laufen, und das ist kein Nachlassen: Sieht ein
	// HINTERGRUND-Abruf den 401 zuerst (Badge-Zähler alle 30 s, Live-Verbindung,
	// Sitzungs-Auffrischung), meldet sich die App schon vor dem Klick ab, die Anmeldemaske
	// legt sich über die Seitenleiste, und der Knopf nimmt keinen Klick mehr an — Playwright
	// wartet dann 30 Sekunden auf eine Bedienbarkeit, die nicht mehr kommt (CI-Lauf
	// 35245097344 vom 17.09.2026; lokal ist das Fenster zu kurz dafür).
	//
	// Gemessen wird ohnehin nicht, WER den 401 zuerst sieht — es ist derselbe Haken in
	// apiFetch —, sondern was die App daraus macht. Kurze Frist, damit der Toast (5 s)
	// darunter noch steht, wenn er im frühen Fall schon ausgelöst wurde.
	await page
		.getByTitle('Medienkatalog')
		.click({ timeout: 3000 })
		.catch(() => {});

	await expect(page.getByText('Sitzung abgelaufen — bitte neu anmelden.')).toBeVisible();
	await expect(page.locator('#login-email')).toBeVisible();
});
