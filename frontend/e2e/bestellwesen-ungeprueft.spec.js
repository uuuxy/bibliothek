// Gate gegen die Warnung, die ausgerechnet bei einer Stoerung verschwindet.
//
// Der Waechter ueber dem Bestellwesen (BestelllinkHinweis) meldet: „Es gibt einen
// Hauptlieferanten, aber keine oeffentliche Adresse — seine Bestellmails gehen ohne
// Bestaetigungs-Link raus." Er haengt an /api/bestellungen/konfiguration.
//
// Bis zum 08.08.2026 fing orderStore.loadKonfiguration jeden Fehler still ab und liess
// das Feld auf seinem Anfangswert `false`. Antwortete der Server also NICHT — 429 unter
// Last, 500, Netz weg —, verschwand der Waechter, und das Bestellwesen behauptete
// „alles in Ordnung". Wer dann bestellte, verschickte Mails ohne Link, und die
// Bestellhistorie wartete auf eine Bestaetigung, die niemand geben kann.
//
// Gefunden hat das kein Audit, sondern die eigene Suite: Der Server antwortete unter
// ihrer Last mit 429, und der Ausfall sah aus wie ein sprunghafter Test. Deshalb steht
// hier ein Gate und nicht nur ein Kommentar.
import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

test('Antwortet die Konfiguration nicht, sagt das Bestellwesen das — statt zu schweigen', async ({
	page
}) => {
	await uiLogin(page);

	// Genau die Antwort nachstellen, die im echten Lauf auftrat.
	await page.route('**/api/bestellungen/konfiguration', (route) =>
		route.fulfill({ status: 429, contentType: 'application/json', body: '{}' })
	);

	await page.goto('/bestellungen');

	const hinweis = page.getByRole('status').filter({ hasText: 'nicht geladen' });
	await expect(
		hinweis,
		'Bei unbeantworteter Konfiguration muss das Bestellwesen es sagen'
	).toBeVisible();

	// Gegenprobe: Ohne die Stoerung ist der graue Streifen weg. Ohne sie waere der Test
	// auch dann gruen, wenn der Hinweis IMMER stuende — und ein Dauerhinweis wird zur
	// Moeblierung, die niemand mehr liest.
	await page.unroute('**/api/bestellungen/konfiguration');
	await page.reload();
	await expect(page.getByRole('tab', { name: 'Bestellhistorie', exact: true })).toBeVisible();
	await expect(hinweis).toHaveCount(0);
});
