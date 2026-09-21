import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

// Das Band „Keine Verbindung" darf nicht allein an navigator.onLine hängen.
//
// Beobachtet am 21.09.2026 am Testserver: Das Band stand dauerhaft, auch am
// Anmeldebildschirm und nach jedem Neuladen — während die Seite darunter ihre Daten lud.
// Am Anmeldebildschirm ist der Herzschlag der Live-Leitung nicht beteiligt; übrig bleibt
// navigator.onLine. Der Wert ist eine Auskunft des Betriebssystems über die
// Netzwerkschnittstellen, keine Messung: Ein Browser kann „offline" melden und den Server
// trotzdem erreichen. Daran hing mehr als das Band — die Warteschlange übertrug nicht, und
// die Leerlauf-Sperre griff nicht.
const BAND = /Keine Verbindung/;

test('Browser meldet offline, der Server antwortet: kein Band', async ({ page }) => {
	await page.addInitScript(() => {
		Object.defineProperty(Navigator.prototype, 'onLine', { get: () => false });
	});
	await page.goto('/');
	await expect(page.getByRole('button', { name: 'Anmelden' })).toBeVisible();
	await expect(page.getByText(BAND)).toHaveCount(0);
});

// Gegenprobe: Ist das Netz wirklich weg, steht das Band — und geht mit dem Netz wieder.
test('Netz wirklich weg: das Band steht', async ({ page, context }) => {
	await uiLogin(page);
	await expect(page.getByText(BAND)).toHaveCount(0);
	await context.setOffline(true);
	await expect(page.getByText(BAND)).toBeVisible();
	await context.setOffline(false);
	await expect(page.getByText(BAND)).toHaveCount(0);
});
