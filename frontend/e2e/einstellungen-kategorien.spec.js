import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, gehZu, einstellungsKategorie } from './helpers.js';

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
	await expect(page.getByText('Bitte eine Zahl eintragen bei: Tage / Buch')).toBeVisible();
	expect(wert('frist_buch_tage'), 'das leere Feld darf keine 0 werden').toBe('28');
	expect(wert('frist_medien_tage'), 'auch das ausgefüllte Feld bleibt ungespeichert').toBe('11');
});
