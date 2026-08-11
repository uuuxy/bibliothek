// Die Selbstprüfung zeigt, was eingerichtet, aber nicht in Betrieb ist.
//
// Warum ein eigener Test und nicht nur die Go-Tests: Die Regeln sind dort vollständig
// geprüft (api/betriebsbereitschaft_test.go, reine Funktion). Was dort NICHT geprüft
// werden kann, ist der Weg dazwischen — Route, Recht, Abruf, Darstellung. Genau daran
// ist auf diesem Projekt schon einmal eine fertige Funktion gescheitert: Der Code war da
// und getestet, erreichbar war er nicht.
//
// Geprüft wird deshalb am INHALT, nicht am Vorhandensein einer Liste: Der lokale Stack hat
// nachweislich kein Auslagerungsziel für Backups, und genau das muss dastehen. Eine Seite,
// die eine leere Liste zeigt, bestünde sonst jeden Test.
import { test, expect } from '@playwright/test';
import { uiLogin, gehZu } from './helpers.js';

test('Betriebsbereitschaft nennt die fehlende Auslagerung samt Abhilfe', async ({ page }) => {
	await uiLogin(page);
	await gehZu(page, '/betriebsbereitschaft');

	// Der Bereich, von dem wir wissen, dass er hier offen ist.
	const auslagerung = page.locator('div', { hasText: 'Auslagerung der Backups' }).last();
	await expect(auslagerung).toBeVisible();

	// Nicht nur „ein Mangel", sondern die Handhabe: Ohne sie landet die Meldung auf einem
	// Zettel statt in der .env.
	await expect(
		page.getByText('S3_ENDPOINT', { exact: false }),
		'die Abhilfe muss die zu setzenden Variablen nennen'
	).toBeVisible();

	// Gegenprobe: Die Seite zeigt auch das, was IN Ordnung ist. Ohne diese Zusage könnte
	// sie schlicht alles als Mangel melden und wäre trotzdem grün.
	await expect(page.getByText('Anmeldung', { exact: false }).first()).toBeVisible();

	// Und sie sagt, wie viel offen ist — die Zahl ist das, was man beim Öffnen sucht.
	await expect(page.getByText(/Punkt(e)? offen\./)).toBeVisible();
});

test('Betriebsbereitschaft hängt am Verwaltungsrecht', async ({ page }) => {
	// Ein Helfer an der Theke darf sie nicht sehen. Die Meldungen nennen Zustände der
	// Anlage — Beispiel-Geheimnisse, mock-Anmeldung, fehlende Auslagerung —, und das ist
	// eine Auskunft über die Angriffsfläche, nicht über die Bibliothek.
	await uiLogin(page, 'e2e-helfer@test.local');

	const antwort = await page.request.get('/api/admin/system/betriebsbereitschaft');
	expect(
		antwort.status(),
		'ohne manage_users darf der Endpunkt keine Auskunft geben'
	).toBeGreaterThanOrEqual(400);
});
