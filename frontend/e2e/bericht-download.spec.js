import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

// "Ich klicke auf PDF herunterladen und lande in der Ausleihe."
//
// Ursache war der Service Worker, nicht der Bericht. Seine NavigationRoute beantwortet
// jede Navigation aus dem Cache mit der App-Shell — richtig für App-Pfade, falsch für
// Server-Antworten. Ein Klick auf den Download-Link ist wegen target="_blank" eine
// Navigation: Statt des Berichts kam die App-Shell zurück, die SPA startete auf
// /api/bestellhistorie/bericht, fand den Pfad nicht in tabToPath und zeigte ihren
// Standard-Reiter.
//
// Heimtückisch daran: Dieselbe URL lieferte per fetch weiterhin ein sauberes PDF — nur
// die Navigation nicht. Der Test geht deshalb über den ECHTEN Klick und prüft die
// Antwort, die im neuen Tab ankommt.
test('Bericht-Download liefert das PDF, nicht die App', async ({ page, context }) => {
	await uiLogin(page);
	await page.getByTitle('Bestellungen').click();
	await page.getByRole('tab', { name: 'Berichte', exact: true }).click();

	const link = page.getByRole('link', { name: /PDF herunterladen/ });
	await expect(link).toHaveAttribute('href', /\/api\/bestellhistorie\/bericht\?/);

	// Vor dem Klick lauschen: Die Antwort kann da sein, bevor wir den neuen Tab in der
	// Hand halten.
	/** @type {string[]} */
	const inhaltstypen = [];
	context.on('response', (r) => {
		if (r.url().includes('/api/bestellhistorie/bericht')) {
			inhaltstypen.push(r.headers()['content-type'] ?? '');
		}
	});

	const neuerTab = context.waitForEvent('page');
	await link.click();
	const tab = await neuerTab;

	// BEWEIS: Was im neuen Tab ankommt, ist ein PDF. Mit dem alten Verhalten stand hier
	// "text/html" — die vom Service Worker ausgelieferte App-Shell.
	await expect.poll(() => inhaltstypen.length).toBeGreaterThan(0);
	expect(inhaltstypen[0]).toContain('application/pdf');

	await tab.close();
});
