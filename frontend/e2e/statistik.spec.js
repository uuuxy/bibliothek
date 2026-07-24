import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

// Smoke-Flow Statistik-Drill-Down: Kachel-Header öffnet das Sidepanel,
// der Filter arbeitet rein clientseitig, Escape schließt.
test('Statistik: Drill-Down-Panel öffnen, filtern, schließen', async ({ page }) => {
	await uiLogin(page);

	// „Statistiken" liegt in der eingeklappten System-Gruppe der Sidebar
	await page.getByRole('button', { name: 'System' }).click();
	await page.getByRole('button', { name: 'Statistiken' }).click();

	// Neue Kennzahl-Kacheln sind da
	await expect(page.getByText('Zirkulationsquote')).toBeVisible();
	await expect(page.getByText('Wiederbeschaffungswert')).toBeVisible();
	await expect(page.getByText('Aktuell verliehen')).toBeVisible();

	// Bestandsfilter: Umschalten auf LMF lädt neu und rendert weiter sauber
	await page.getByRole('button', { name: 'LMF', exact: true }).click();
	await expect(page.getByText('Gesamtbestand')).toBeVisible();
	await page.getByRole('button', { name: 'Gesamt', exact: true }).click();
	await expect(page.getByText('Zirkulationsquote')).toBeVisible();

	// Kachel-Header ist der Drill-Down-Einstieg
	await page.getByRole('button', { name: /Ladenhüter — Detailansicht öffnen/ }).click();

	// Seit dem Refactoring ist es eine eigene Route, kein Dialog-Panel mehr.
	// Die Hauptüberschrift (h1) der Detailseite beweist den erfolgreichen Navigationswechsel.
	const pageHeading = page.getByRole('heading', { name: 'Ladenhüter' });
	await expect(pageHeading).toBeVisible();
	await expect(page.getByText(/von \d+ Einträgen/)).toBeVisible();

	// Clientseitiger Filter: Nonsens-Suchbegriff leert die Liste ohne API-Call
	await page.getByPlaceholder('Titel oder Autor…').fill('xx-niemals-treffer-xx');
	await expect(page.getByText('Keine Einträge für diese Filter.')).toBeVisible();
	await expect(page.getByText(/^0 von \d+ Einträgen/)).toBeVisible();

	// Der Zurück-Button führt zurück zum Dashboard (Button-Text: 'Statistik')
	await page.getByRole('button', { name: 'Statistik', exact: true }).click();
	// Verify we are back on the dashboard
	await expect(page.getByRole('button', { name: /Ladenhüter — Detailansicht öffnen/ })).toBeVisible();
});
