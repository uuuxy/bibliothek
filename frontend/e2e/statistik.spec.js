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

	// „Bestands-Analysen" zeigt Renner UND Ladenhüter in EINER Card; das Segmented Control
	// schaltet um, „Alle anzeigen" führt in die jeweils aktive Detailansicht.
	await page.getByRole('button', { name: 'Ladenhüter', exact: true }).click();
	await page.getByRole('button', { name: /Ladenhüter — Detailansicht öffnen/ }).click();

	const panel = page.getByRole('main');
	await expect(panel.getByRole('heading', { name: 'Ladenhüter' })).toBeVisible();
	await expect(panel.getByText(/von \d+ Einträgen/)).toBeVisible();

	// Clientseitiger Filter: Nonsens-Suchbegriff leert die Liste ohne API-Call
	await panel.getByPlaceholder('Titel oder Autor…').fill('xx-niemals-treffer-xx');
	// Regex statt 'text=A|text=B': Letzteres ist KEINE gültige Playwright-Syntax
	// (wird als ein einziger Textstring gesucht) und matchte deshalb nie.
	await expect(
		page.getByText(/Keine Einträge für diese Filter\.|Noch keine Daten vorhanden\./)
	).toBeVisible();
	await expect(page.getByText(/^0 von \d+ Einträgen/)).toBeVisible();

	// Zurück zur Übersicht. Der Button trägt den Namen „Statistik" (nicht „Zurück");
	// exact:true, damit er nicht mit „Statistiken" in der Sidebar kollidiert.
	await page.getByRole('button', { name: 'Statistik', exact: true }).click();

	// Verlassen der Detailseite über ihr eigenes Suchfeld prüfen — NICHT über die
	// Überschrift „Ladenhüter": die gibt es auf dem Dashboard als Kachel-Header auch.
	await expect(page.getByPlaceholder('Titel oder Autor…')).toBeHidden();
	await expect(page.getByText('Zirkulationsquote')).toBeVisible();
});
