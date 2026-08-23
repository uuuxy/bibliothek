import { test, expect } from '@playwright/test';
import {
	uiLogin,
	seedSQL,
	querySQL,
	gehZu,
	einstellungsKategorie,
	csrfToken,
	pruefeFeldreihen
} from './helpers.js';

// Die Zusicherung der neuen Einstellungsseite (23.08.2026), am Draht geprüft:
// Wer EINE Kategorie speichert, ändert nichts in den anderen.
//
// Warum das ein eigenes Gate braucht, obwohl das Repository dieselbe Aussage schon
// beweist (system_settings_roundtrip_pg_test.go): Der Beweis dort beginnt beim
// Patch-Objekt. Ob die OBERFLÄCHE wirklich nur ihre eigenen Felder hineinlegt, sieht
// er nicht — und genau dort saß der Fehler bis gestern. Eine Sektion schickte das
// ganze Formular, und das Backend schrieb elf Schlüssel aus lauter Nullwerten.
//
// Bis zum 22.08. hätte dieser Test die Anwendung in einem Zustand hinterlassen, in
// dem Ferien-Leseclub und Preiserfassung aus sind und drei Fristen auf der Vorgabe
// stehen — mit einer grünen Erfolgsmeldung auf dem Bildschirm.

/** @param {string} schluessel */
const wert = (schluessel) =>
	querySQL(`SELECT wert FROM system_einstellungen WHERE schluessel = '${schluessel}'`);

/** Setzt einen von der Vorgabe klar unterscheidbaren Ausgangsstand. */
function ausgangsstand() {
	seedSQL(`
		INSERT INTO system_einstellungen (schluessel, wert) VALUES
			('frist_buch_tage', '28'),
			('frist_medien_tage', '11'),
			('max_ausleihen_schueler', '9'),
			('bestellbedarf_schwelle', '7'),
			('ferien_leseclub_aktiv', 'true'),
			('preise_erfassen', 'true'),
			('lmf_stichtag', '08-15'),
			('sperre_minuten', '15')
		ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert;
	`);
}

// Die Einstellungen sind global und die Suite teilt sich eine Datenbank: ohne das
// Zurücksetzen liefen die nachfolgenden Specs mit fremden Fristen.
test.afterEach(() => {
	seedSQL(`
		INSERT INTO system_einstellungen (schluessel, wert) VALUES
			('frist_buch_tage', '21'),
			('frist_medien_tage', '7'),
			('max_ausleihen_schueler', '5'),
			('bestellbedarf_schwelle', '3'),
			('ferien_leseclub_aktiv', 'false'),
			('preise_erfassen', 'true'),
			('lmf_stichtag', '07-31'),
			('sperre_minuten', '15')
		ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert;
	`);
});

test('Eine Kategorie speichern lässt alle anderen unangetastet', async ({ page }) => {
	ausgangsstand();
	await uiLogin(page);
	await gehZu(page, '/einstellungen');

	await einstellungsKategorie(page, 'Datenschutz & Sitzung').click();

	const feld = page.getByLabel('Sperrbildschirm nach (Min.)');
	await feld.waitFor();
	await feld.fill('12');
	await page.getByRole('button', { name: 'Datenschutz & Sitzung speichern' }).click();
	await expect(page.getByText('Gespeichert.')).toBeVisible();

	// Was gespeichert werden sollte, ist gespeichert …
	expect(wert('sperre_minuten')).toBe('12');

	// … und was in fremden Kategorien steht, steht unverändert da.
	expect(
		{
			frist_buch_tage: wert('frist_buch_tage'),
			frist_medien_tage: wert('frist_medien_tage'),
			max_ausleihen_schueler: wert('max_ausleihen_schueler'),
			bestellbedarf_schwelle: wert('bestellbedarf_schwelle'),
			ferien_leseclub_aktiv: wert('ferien_leseclub_aktiv'),
			preise_erfassen: wert('preise_erfassen'),
			lmf_stichtag: wert('lmf_stichtag')
		},
		'Speichern in „Datenschutz & Sitzung" hat fremde Einstellungen überschrieben'
	).toEqual({
		frist_buch_tage: '28',
		frist_medien_tage: '11',
		max_ausleihen_schueler: '9',
		bestellbedarf_schwelle: '7',
		ferien_leseclub_aktiv: 'true',
		preise_erfassen: 'true',
		lmf_stichtag: '08-15'
	});
});

// Die Kehrseite derselben Entscheidung: Weil es keine „leer = lass es wie es war"-Regel
// mehr gibt, darf ein leer geräumtes Zahlenfeld auch nicht stillschweigend als 0
// durchgehen — genau so schaltete sich am 22.08. die Lesehistorie-Befristung ab.
test('Ein leer geräumtes Zahlenfeld speichert gar nichts und sagt, welches Feld fehlt', async ({
	page
}) => {
	ausgangsstand();
	await uiLogin(page);
	await gehZu(page, '/einstellungen');

	await einstellungsKategorie(page, 'Ausleihe & Fristen').click();
	await page.getByLabel('Tage / Buch').fill('');
	await page.getByLabel('Tage / Medien').fill('19');
	await page.getByRole('button', { name: 'Ausleihe & Fristen speichern' }).click();

	// Die Meldung nennt die BESCHRIFTUNG des Feldes, nicht seinen API-Schlüssel —
	// „frist_buch_tage fehlt" hilft am Bildschirm niemandem.
	await expect(page.getByText('Bitte eine gültige Zahl eintragen bei: Tage / Buch')).toBeVisible();
	expect(wert('frist_buch_tage'), 'das leere Feld darf keine 0 werden').toBe('28');
	expect(wert('frist_medien_tage'), 'auch das ausgefüllte Feld bleibt ungespeichert').toBe('11');
});

// Ein PUT ohne ein einziges Feld ist ein Aufruferfehler, keine leere Speicherung.
//
// Vorher antwortete der Server darauf mit „ok" und schrieb einen Audit-Eintrag
// UPDATE_SETTINGS mit leerem Rumpf — eine protokollierte Änderung, die nie
// stattgefunden hat, und einer kaputten Oberfläche die Bescheinigung, sie habe
// gespeichert. Genau die Klasse „stiller Erfolg", die dieses Projekt schon zweimal
// Monate gekostet hat.
test('Ein PUT ohne Felder wird abgelehnt, statt Erfolg zu melden', async ({ page }) => {
	await uiLogin(page);

	const vorher = querySQL(`SELECT count(*) FROM audit_logs WHERE aktion = 'UPDATE_SETTINGS'`);
	const antwort = await page.request.put('/api/einstellungen', {
		headers: { 'X-CSRF-Token': await csrfToken(page) },
		data: {}
	});

	expect(antwort.status(), 'leerer Rumpf muss 400 sein, nicht 200').toBe(400);
	expect(
		querySQL(`SELECT count(*) FROM audit_logs WHERE aktion = 'UPDATE_SETTINGS'`),
		'ein abgelehnter Aufruf darf keine Änderung protokollieren'
	).toBe(vorher);
});

// „Ein Feld ist verrutscht" (Peter, 23.08.2026, Bildschirmfoto von flasch3): In
// „Datenschutz & Sitzung" bricht die Beschriftung „Lesehistorie Schülerbücherei (Tage)"
// auf zwei Zeilen um — und ihr Eingabefeld stand dadurch eine Zeilenhöhe tiefer als die
// drei Nachbarn daneben. Ursache war nicht die lange Beschriftung, sondern dass jedes
// Feld sich nur an sich selbst ausrichtete; behoben mit subgrid in SettingField.
//
// Gemessen wird im BROWSER, nicht an Klassennamen: Eine Klassen-Inventur hat in diesem
// Projekt schon einmal 29 Fundstellen gemeldet, wo real fünf waren.
//
// Die Regel: Zwei Eingabefelder stehen entweder auf DERSELBEN Höhe (eine Reihe) oder
// deutlich auseinander (verschiedene Reihen, gemessen 104 px). Ein Abstand dazwischen —
// eine einzelne Zeilenhöhe — ist genau der Fehler und nichts sonst.
const MIT_MEHREREN_FELDERN = [
	'Schule',
	'Ausleihe & Fristen',
	'Mahnwesen',
	'Datenschutz & Sitzung',
	'Erreichbarkeit & Alarme',
	'Mail'
];

test('Kein Eingabefeld hängt eine Zeile tiefer als seine Nachbarn', async ({ page }) => {
	await page.setViewportSize({ width: 1440, height: 900 });
	await uiLogin(page);
	await gehZu(page, '/einstellungen');

	for (const kategorie of MIT_MEHREREN_FELDERN) {
		await einstellungsKategorie(page, kategorie).click();
		await page.locator('input:visible').first().waitFor();
		await pruefeFeldreihen(page, `Einstellungen → ${kategorie}`);
	}
});

// Dieselbe Prüfung im Kollegiums-Portal: Dort ist der Fehler eine Stunde nach seiner
// Behebung ein zweites Mal entstanden — nicht durch eine umbrechende Beschriftung,
// sondern durch eine <div class="sm:col-span-2">-Hülle um ein Feld. Die Hülle ist
// selbst das Rasterelement und spannt eine Zeile, während ihre Nachbarn drei spannen.
test('Portal: kein Eingabefeld hängt eine Zeile tiefer als seine Nachbarn', async ({ page }) => {
	await page.setViewportSize({ width: 1440, height: 900 });
	await uiLogin(page);
	await gehZu(page, '/kollegium-portal');
	await page.getByRole('tab', { name: /Meine Anliegen/ }).click();
	await page.locator('input:visible').first().waitFor();
	await pruefeFeldreihen(page, 'Portal → Meine Anliegen');
});
