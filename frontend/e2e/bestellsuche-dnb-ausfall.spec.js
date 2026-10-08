import { test, expect } from '@playwright/test';
import { uiLogin, gehZu } from './helpers.js';

// Die Bestellsuche fragt den eigenen Katalog und die DNB. Antwortet die DNB nicht, nennt die
// Tür es im Kopf X-DNB-Ausfall, und die Trefferliste sagt es: Ein Buch, das die DNB kennt,
// sähe sonst aus wie eines, das sie nicht kennt.
//
// Die Antwort der Tür ist hier nachgestellt; der Stack fragte sonst die echte DNB. Dass die
// Tür den Kopf setzt, prüft api/bestellsuche_pg_test.go.

const HINWEIS = 'Die DNB ist nicht erreichbar — bitte später erneut versuchen.';

/**
 * @param {import('@playwright/test').Page} page
 * @param {any[]} treffer
 * @param {boolean} dnbAusfall
 */
async function sucheAntwortet(page, treffer, dnbAusfall) {
	await page.route('**/api/bestellungen/suche', (route) =>
		route.fulfill({
			status: 200,
			contentType: 'application/json',
			headers: dnbAusfall ? { 'X-DNB-Ausfall': '1' } : {},
			body: JSON.stringify(treffer)
		})
	);
}

/** @param {import('@playwright/test').Page} page */
async function suche(page, text) {
	await uiLogin(page);
	await gehZu(page, '/bestellungen');
	const antwort = page.waitForResponse('**/api/bestellungen/suche');
	await page.getByRole('searchbox', { name: 'Titel suchen & hinzufügen' }).fill(text);
	await antwort;
}

test.describe('Bestellsuche: Die DNB antwortet nicht', () => {
	test('ohne Treffer im Katalog öffnet die Trefferliste mit dem Hinweis', async ({ page }) => {
		await sucheAntwortet(page, [], true);
		await suche(page, '9783060130764');

		await expect(page.getByRole('alert').filter({ hasText: HINWEIS })).toBeVisible();
		await expect(page.getByText('Im lokalen Bestand')).toHaveCount(0);
		await expect(page.getByText('Neu aus DNB (Externe Suche)')).toHaveCount(0);
	});

	// Unter vielen Treffern aus dem Bestand läse den Hinweis niemand: Er steht über ihnen.
	test('der Hinweis steht über den Treffern aus dem Katalog', async ({ page }) => {
		const imHaus = {
			id: '00000000-0000-4000-8000-000000000001',
			titel: 'Faust im Haus',
			autor: 'Goethe',
			isbn: '9783150000014',
			source: 'local',
			current_stock: 3
		};
		await sucheAntwortet(page, [imHaus], true);
		await suche(page, 'faust');

		const treffer = page.getByRole('button', { name: /Faust im Haus/ });
		const hinweis = page.getByRole('alert').filter({ hasText: HINWEIS });
		await expect(treffer).toBeVisible();
		await expect(hinweis).toBeVisible();
		const [oben, unten] = [await hinweis.boundingBox(), await treffer.boundingBox()];
		expect(oben && unten && oben.y < unten.y, 'der Hinweis steht über dem Katalogtreffer').toBe(
			true
		);
		await expect(hinweis).toBeInViewport({ ratio: 1 });
	});

	test('Gegenprobe: Antwortet die DNB ohne Treffer, bleibt die Liste zu', async ({ page }) => {
		await sucheAntwortet(page, [], false);
		await suche(page, 'gibt es nirgends');

		await expect(page.getByText(/^Suche läuft/)).toHaveCount(0);
		await expect(page.getByText(HINWEIS)).toHaveCount(0);
		await expect(page.getByText('Neu aus DNB (Externe Suche)')).toHaveCount(0);
	});
});
