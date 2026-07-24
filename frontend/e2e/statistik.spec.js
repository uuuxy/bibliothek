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

	// Erwartet eine Detailseite, kein Dialog mehr
	await expect(page.getByRole('heading', { name: 'Ladenhüter' })).toBeVisible();
	await expect(page.getByText(/von \d+ Einträgen/)).toBeVisible();

	// Clientseitiger Filter: Nonsens-Suchbegriff leert die Liste ohne API-Call
	await page.getByPlaceholder('Titel oder Autor…').fill('xx-niemals-treffer-xx');
	await expect(page.getByText(/Keine Einträge für diese Filter.|Noch keine Daten vorhanden./)).toBeVisible();
	await expect(page.getByText(/^0 von \d+ Einträgen/)).toBeVisible();

	// Zurück zur Übersicht
	await page.getByRole('button', { name: 'Statistik' }).click();
	await expect(page.getByRole('heading', { name: 'Ladenhüter' })).not.toBeVisible();
});
