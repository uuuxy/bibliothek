import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

// Audit-Befund vom 01.08.2026: /katalog war doppelt vergeben.
//
// Der Router schob den Pfad /katalog für den INTERNEN Medienkatalog, App.svelte rendert
// bei exakt /katalog aber den ÖFFENTLICHEN OPAC — vor jeder Anmeldeprüfung und am Router
// vorbei. Wer angemeldet den Katalog öffnete und neu lud, landete im öffentlichen
// Katalog: ohne Navigation, ohne Verwaltung, ohne Hinweis, dass etwas schiefging.
//
// Der interne Katalog liegt jetzt auf /medienkatalog, /katalog gehört allein dem OPAC.
test('Medienkatalog überlebt das Neuladen und wird nicht zum öffentlichen OPAC', async ({
	page
}) => {
	await uiLogin(page);

	await page.getByTitle('Medienkatalog').click();
	await expect(page).toHaveURL(/\/medienkatalog$/);

	// Der Kern: neu laden. Vorher stand danach der OPAC da.
	await page.reload();

	// Die Navigation beweist, dass wir in der angemeldeten Anwendung sind — der
	// öffentliche OPAC hat keine.
	await expect(
		page.getByTitle('Medienkatalog'),
		'nach dem Neuladen muss die interne Navigation da sein, nicht der oeffentliche OPAC'
	).toBeVisible();
	await expect(page).toHaveURL(/\/medienkatalog$/);

	// Gegenprobe: /katalog ist weiterhin der öffentliche Katalog — dort gibt es keine
	// Navigation, und die Kopfzeile nennt ihn beim Namen.
	await page.goto('/katalog');
	await expect(page.getByText('Öffentlicher Medienkatalog')).toBeVisible();
	await expect(page.getByTitle('Medienkatalog')).toHaveCount(0);
});
