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
import { uiLogin, seedBenutzer, einstellungsKategorie } from './helpers.js';

// Eigene Anmeldeadresse, nicht die von helfer-kiosk.spec.js.
//
// Bis zum 12.08.2026 stand hier `e2e-helfer@test.local` — angelegt wird der aber von
// helfer-kiosk.spec.js, und diese Datei läuft alphabetisch davor. Lokal war das
// unsichtbar (der Benutzer lag von früheren Läufen in der Datenbank), in CI mit frischer
// Datenbank scheiterte die Anmeldung und der Test lief in den 30-Sekunden-Timeout.
const HELFER = 'e2e-bb-helfer@test.local';

test('Betriebsbereitschaft nennt die fehlende Auslagerung samt Abhilfe', async ({ page }) => {
	await uiLogin(page);

	// Über das MENÜ, nicht über die URL. Der erste Anlauf navigierte direkt — und war
	// grün, obwohl der Menüeintrag fehlte: Die Seite war nur für den erreichbar, der den
	// Pfad kennt. Ein Bildschirm, den niemand findet, ist so gut wie keiner.
	//
	// Seit 16.08.2026 ist die Selbstprüfung Teil der Einstellungen (Betreiber-
	// Entscheidung: schlankeres Menü), seit 23.08.2026 als Eintrag der Kategorienliste
	// — der Weg der Verwaltungskraft führt über die zugeklappte System-Gruppe, die
	// Einstellungen und die Kategorie.
	await page.getByRole('button', { name: 'System', exact: true }).click();
	await page.getByTitle('Einstellungen').click();
	await expect(page).toHaveURL(/\/einstellungen$/);
	await einstellungsKategorie(page, 'Betriebsbereitschaft').click();

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
	seedBenutzer(HELFER, 'helfer');
	await uiLogin(page, HELFER);

	const antwort = await page.request.get('/api/admin/system/betriebsbereitschaft');
	expect(
		antwort.status(),
		'ohne manage_users darf der Endpunkt keine Auskunft geben'
	).toBeGreaterThanOrEqual(400);

	// Und der Bildschirm selbst bleibt zu.
	//
	// Geprüft wird der EINLEITUNGSSATZ der Ansicht, nicht ein Befund. Der erste Anlauf sah
	// nach „Auslagerung der Backups" — und war grün, ohne irgendetwas zu belegen: Die
	// Befunde bleiben für einen Helfer ohnehin aus, weil der Endpunkt oben sperrt. Der
	// Einleitungssatz dagegen steht fest im Markup und erscheint, sobald die Ansicht
	// überhaupt gerendert wird.
	//
	// Die Selbstprüfung ist eine Kategorie der EINSTELLUNGEN; deren Menüeintrag speist
	// tabIstGesperrt(). Eine eigene /betriebsbereitschaft-Route existiert nicht mehr —
	// der frühere URL-Schleichweg ist damit strukturell weg, geprüft wird der Weg über
	// die Einstellungs-URL.
	await page.goto('/einstellungen');
	await expect(
		page.getByText('Was ist eingerichtet, aber nicht in Betrieb?'),
		'ein Helfer darf den Bildschirm gar nicht erst zu sehen bekommen'
	).toHaveCount(0);
});
