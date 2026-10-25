import { test, expect } from '@playwright/test';
import { menuepunkt, uiLogin } from './helpers.js';

// Nach einem Update läuft ein offenes Fenster mit dem alten Programm weiter. Der Server nennt
// in jeder Antwort den Stand der Oberfläche, den Namen der Programmdatei aus index.html; weicht
// er von dem des Fensters ab, steht über dem Inhalt der Hinweis „Neu laden". Geprüft am gebauten
// Stack, weil nur dort Server und Oberfläche aus demselben Bau stammen.

const HINWEIS = 'Eine neue Fassung des Programms ist eingespielt.';

test('der Server nennt den Stand, mit dem die Seite geladen ist', async ({ page }) => {
	await uiLogin(page);
	const programmdatei = await page
		.locator('script[type="module"][src]')
		.first()
		.getAttribute('src');
	expect(programmdatei).toMatch(/^\/assets\/index-.+\.js$/);

	const res = await page.request.get('/health');
	expect(res.headers()['x-oberflaechenstand']).toBe(programmdatei);
	await expect(page.getByText(HINWEIS)).toBeHidden();
});

// Ab jetzt nennt der Server einen anderen Stand, das Fenster bleibt offen: wie nach einem Update,
// ohne den Stack neu zu bauen.
const umgeleitet = (/** @type {URL} */ url) =>
	url.pathname.startsWith('/api/') || url.pathname === '/health';

/** @param {import('@playwright/test').Page} page */
async function serverNenntNeuenStand(page) {
	await page.route(umgeleitet, async (route) => {
		const antwort = await route.fetch();
		await route.fulfill({
			response: antwort,
			headers: { ...antwort.headers(), 'x-oberflaechenstand': '/assets/index-neue-fassung.js' }
		});
	});
}

test('ein anderer Stand am Server zeigt den Hinweis, und „Neu laden" lädt die Seite neu', async ({
	page
}) => {
	await uiLogin(page);
	await serverNenntNeuenStand(page);
	await menuepunkt(page, 'Medienkatalog').click();

	const hinweis = page.getByRole('status').filter({ hasText: HINWEIS });
	await expect(hinweis).toBeVisible();
	// Er hält nichts an: Die Navigation bleibt bedienbar.
	await menuepunkt(page, 'Leserdatei').click();
	await expect(hinweis).toBeVisible();

	const geladen = page.waitForEvent('load');
	await hinweis.getByRole('button', { name: 'Neu laden' }).click();
	await geladen;
	await page.getByRole('button', { name: 'Abmelden' }).waitFor();
	await page.unrouteAll({ behavior: 'ignoreErrors' });
});

// Nach einem Rückbau nennt der Server wieder den Stand des Fensters; dann ist nichts neu zu laden.
test('nennt der Server wieder den Stand des Fensters, geht der Hinweis weg', async ({ page }) => {
	await uiLogin(page);
	await serverNenntNeuenStand(page);
	await menuepunkt(page, 'Medienkatalog').click();
	const hinweis = page.getByRole('status').filter({ hasText: HINWEIS });
	await expect(hinweis).toBeVisible();

	await page.unrouteAll({ behavior: 'ignoreErrors' });
	await menuepunkt(page, 'Leserdatei').click();
	await expect(hinweis).toBeHidden();
});
