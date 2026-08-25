import { test, expect } from '@playwright/test';

// Round-Trip-Sonde für die Feld-Migration (25.08.2026): Sieben Schreibpfade, die nach dem
// Umbau von 78 rohen <input> auf ui/Feld keinen E2E-Beweis hatten, bis in die Datenbank.
// Warum bis in die DB: Der Fund des Tages (SettingField hatte type="number" als Standard,
// Feld "text" — 13 Aufrufer schickten Strings ans Backend) war in der Oberfläche unsichtbar
// und fiel erst am 400 des Servers auf. Ein Attribut-Vergleich alt/neu fand ihn nicht.
import { uiLogin, seedSQL, querySQL, uniqueSuffix } from './helpers.js';
const s = uniqueSuffix().slice(0, 6);
const LEHRER = 'e2e-rt-lehrer@test.local';
// Gültige ISBN-13 aus der Zeit: 978 + 9 Ziffern + Prüfziffer.
const kern = ('978' + String(Date.now()).slice(-9)).slice(0, 12);
const pruef = (10 - ([...kern].reduce((a, d, i) => a + Number(d) * (i % 2 ? 3 : 1), 0) % 10)) % 10;
const ISBN = kern + pruef;
/** @type {string} */
let FRIST_VORHER = '21';
test.describe.serial('Round-Trip-Sonde migrierter Felder', () => {
	test.beforeAll(() => {
		FRIST_VORHER =
			querySQL(`SELECT wert FROM system_einstellungen WHERE schluessel = 'frist_buch_tage'`) ||
			'21';
		seedSQL(`
			INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv) VALUES ('RT', 'Lehrer', '${LEHRER}', 'kollegium', true) ON CONFLICT (email) DO UPDATE SET aktiv = true;
			INSERT INTO schueler (vorname, nachname, klasse, barcode_id, geburtsdatum, abgaenger_jahr) VALUES ('Rt${s}', 'Sonde', '7A', 'RT-${s}', '2012-01-01', 2030);
			WITH t AS (INSERT INTO buecher_titel (isbn, titel, autor, signatur) VALUES ('978rt${s}', 'RT Buch ${s}', 'Autor', 'BIB Rt') RETURNING id)
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar) SELECT id, 'RT-EX-${s}', true FROM t;
			INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist)
			SELECT e.id, sch.id, now(), now() + interval '14 days' FROM buecher_exemplare e, schueler sch WHERE e.barcode_id = 'RT-EX-${s}' AND sch.barcode_id = 'RT-${s}';
		`);
	});
	test.afterAll(() => {
		seedSQL(`
			UPDATE system_einstellungen SET wert = '${FRIST_VORHER}' WHERE schluessel = 'frist_buch_tage';
			DELETE FROM ausleihen WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE barcode_id LIKE 'RT-%${s}%');
			DELETE FROM lehrer_anliegen WHERE titel_text LIKE 'RT Wunsch ${s}%';
			DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE isbn IN ('978rt${s}','${ISBN}'));
			DELETE FROM buecher_titel WHERE isbn IN ('978rt${s}','${ISBN}');
			DELETE FROM schueler WHERE barcode_id = 'RT-${s}';
		`);
	});

	test('Buch anlegen: Bestand, Zähldatum, Standort kommen in der DB an', async ({ page }) => {
		await uiLogin(page);
		await page.goto('/medienkatalog');
		await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
		await page.getByRole('button', { name: 'Neues Buch' }).first().click();
		await page.locator('#buch-titel').fill(`RT Neu ${s}`);
		await page.locator('#buch-isbn').fill(ISBN);
		await page.locator('#buch-signatur').fill('BIB Rt');
		await page.locator('#buch-bestand').fill('3');
		await page.locator('#buch-zaehldatum').fill('2026-08-25');
		await page.locator('#buch-standort').fill(`Regal ${s}`);
		await page.getByRole('button', { name: 'Speichern' }).click();
		await expect(page.getByText(`RT Neu ${s}`).first()).toBeVisible({ timeout: 10000 });
		const row = querySQL(
			`SELECT last_counted::text || '|' || coalesce(erweiterte_eigenschaften->>'standort','') || '|' || (SELECT count(*) FROM buecher_exemplare e WHERE e.titel_id = t.id) FROM buecher_titel t WHERE titel = 'RT Neu ${s}'`
		);
		expect(row).toBe(`2026-08-25|Regal ${s}|3`);
	});

	test('Abgangsjahr und Rückgabedatum im Profil', async ({ page }) => {
		await uiLogin(page);
		await page.goto('/schuelerdatei');
		await page.getByRole('searchbox', { name: 'Schüler suchen' }).fill(`Rt${s}`);
		await page.getByText(`Rt${s} Sonde`).first().click();
		await page.getByRole('button', { name: /^Abgang 2030/ }).click();
		await page.getByLabel('Abgangsjahr').fill('2031');
		await page.getByRole('button', { name: 'Speichern' }).first().click();
		await expect
			.poll(() => querySQL(`SELECT abgaenger_jahr FROM schueler WHERE barcode_id = 'RT-${s}'`))
			.toBe('2031');
		await page.getByRole('button', { name: /Ausleihen & Historie/ }).click();
		await page.getByRole('button', { name: 'Rückgabedatum bearbeiten' }).first().click();
		await page.getByLabel('Rückgabedatum', { exact: true }).fill('2026-12-24');
		await page.getByRole('button', { name: 'Rückgabedatum speichern' }).click();
		await expect
			.poll(() =>
				querySQL(
					`SELECT rueckgabe_frist::date::text FROM ausleihen a JOIN buecher_exemplare e ON e.id = a.exemplar_id WHERE e.barcode_id = 'RT-EX-${s}'`
				)
			)
			.toBe('2026-12-24');
	});

	test('Exemplar-Barcode in der Buch-Akte', async ({ page }) => {
		await uiLogin(page);
		const id = querySQL(`SELECT id FROM buecher_titel WHERE isbn = '978rt${s}'`);
		await page.goto(`/medienkatalog/buch/${id}`);
		await page.getByRole('tab', { name: /^Exemplare/ }).click();
		await page.getByRole('button', { name: 'Barcode zuweisen oder ändern' }).first().click();
		await page.getByLabel('Barcode des Exemplars').fill(`RT-NEU-${s}`);
		await page.getByRole('button', { name: 'Speichern' }).first().click();
		await expect
			.poll(() =>
				querySQL(`SELECT count(*) FROM buecher_exemplare WHERE barcode_id = 'RT-NEU-${s}'`)
			)
			.toBe('1');
	});

	test('Anliegen anlegen (Lehrkraft) und erledigen mit Notiz (Bibliothek)', async ({ browser }) => {
		const l = await browser.newContext();
		const lp = await l.newPage();
		await uiLogin(lp, LEHRER);
		await lp.getByTitle('Mein Portal').click();
		await lp.getByRole('tab', { name: 'Meine Anliegen' }).click();
		await lp.getByLabel('Welches Buch?').fill(`RT Wunsch ${s}`);
		await lp.getByLabel('Klasse / Kurs').fill('7A');
		await lp.getByRole('button', { name: 'Absenden' }).click();
		await expect
			.poll(() =>
				querySQL(`SELECT klasse FROM lehrer_anliegen WHERE titel_text = 'RT Wunsch ${s}'`)
			)
			.toBe('7A');
		await l.close();
		const a = await browser.newContext();
		const ap = await a.newPage();
		await uiLogin(ap);
		await ap.goto('/bestellungen');
		await ap.getByRole('tab', { name: /Wünsche & Meldungen/ }).click();
		await expect(ap.getByText(`RT Wunsch ${s}`)).toBeVisible();
		await ap.getByRole('button', { name: 'Abhaken' }).first().click();
		await ap.getByLabel('Notiz für die Mail an die Lehrkraft').fill(`Notiz ${s}`);
		await ap.getByRole('button', { name: 'Erledigt & Mail senden' }).click();
		await expect
			.poll(() =>
				querySQL(
					`SELECT coalesce(erledigt_notiz,'') FROM lehrer_anliegen WHERE titel_text = 'RT Wunsch ${s}'`
				)
			)
			.toBe(`Notiz ${s}`);
		await a.close();
	});

	test('Einstellungs-Zahlenfeld kommt als Zahl an (Fund vom 25.08.: type-Standard kippte)', async ({
		page
	}) => {
		await uiLogin(page);
		await page.goto('/einstellungen');
		await page.getByText('Ausleihe & Fristen').first().click();
		const feld = page.getByLabel('Tage / Buch');
		await feld.fill('23');
		await page.getByRole('button', { name: 'Ausleihe & Fristen speichern' }).click();
		await expect
			.poll(() =>
				querySQL(`SELECT wert FROM system_einstellungen WHERE schluessel = 'frist_buch_tage'`)
			)
			.toBe('23');
	});
});
