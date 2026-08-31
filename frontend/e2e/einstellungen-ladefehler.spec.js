import { test, expect } from '@playwright/test';
import { uiLogin, gehZu } from './helpers.js';

// Ein gescheiterter Abruf darf kein ausgefülltes Formular zeigen.
//
// Fund des Raster-Durchgangs 31.08.2026: Scheiterte GET /api/einstellungen, setzte der
// Lader `daten = {}` und schaltete `loading` trotzdem ab. Jede Kategorie liest ihre
// Startwerte als `start.X ?? Vorgabe` — das Formular sah damit vollständig ausgefüllt
// aus, obwohl NICHTS geladen war. Und weil jede Kategorie beim Speichern ALLE ihre
// Felder schickt (nicht nur die geänderten), schrieb ein Klick die Vorgaben über die
// echten Werte: Schulname und Briefkopf leer (Mahnung, Bestellung, Buchetikett),
// Lesehistorie-Frist von 1825 zurück auf 730, Ferien-Leseclub aus.
test('Einstellungen: Ladefehler zeigt Hinweis statt Vorgabewerten zum Überspeichern', async ({
	page
}) => {
	await uiLogin(page);

	// Erst NACH dem Login abklemmen: Der Abruf gehört zur Einstellungsseite.
	await page.route('**/api/einstellungen', async (route) => {
		if (route.request().method() === 'GET') {
			await route.fulfill({ status: 500, contentType: 'application/json', body: '{}' });
			return;
		}
		await route.continue();
	});

	await gehZu(page, '/einstellungen');

	// Der Hinweis steht, das Formular nicht.
	await expect(page.getByText('Einstellungen nicht geladen')).toBeVisible();
	await expect(page.getByRole('button', { name: /speichern$/i })).toHaveCount(0);
	await expect(page.getByRole('navigation', { name: 'Einstellungs-Kategorien' })).toHaveCount(0);

	// Und der Weg zurück ist da: Sobald der Server wieder antwortet, lädt „Erneut
	// versuchen" die echten Werte.
	await page.unroute('**/api/einstellungen');
	await page.getByRole('button', { name: 'Erneut versuchen' }).click();
	await expect(page.getByRole('navigation', { name: 'Einstellungs-Kategorien' })).toBeVisible();
});
