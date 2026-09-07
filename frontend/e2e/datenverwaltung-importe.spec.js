import { test, expect } from '@playwright/test';
import {
	uiLogin,
	seedSQL,
	querySQL,
	uniqueSuffix,
	einstellungsKategorie,
	csrfToken
} from './helpers.js';

// Die Datenverwaltung hat drei Importwege, die sich in EINER Frage unterscheiden: übernimmt
// der Weg vorhandene Exemplare samt Nummer, oder erzeugt er neue? Der Listenimport (ISBN +
// Stückzahl) erzeugt — und hatte vom 21.06. bis zum 07.09.2026 keinen Knopf: Er saß in der
// Titel-Verwaltung und ging beim Entschlacken der Toolbar ohne Ersatz verloren; nur die
// Backend-Route blieb (Befund-Register). Der Bestands-Import übernimmt — und zog am
// 07.09.2026 aus DataManagement.svelte in ein eigenes Widget.
//
// Beide Wege laufen hier bis in die Datenbank, denn genau dort liegt der Unterschied:
// frische B-Nummern ohne Etikett gegen die mitgebrachte Nummer mit Etikett.
const s = uniqueSuffix();
// Gültige ISBN-13 aus der Zeit: 978 + 9 Ziffern + Prüfziffer (wie feld-roundtrip).
const kern = ('978' + String(Date.now()).slice(-9)).slice(0, 12);
const pruef = (10 - ([...kern].reduce((a, d, i) => a + Number(d) * (i % 2 ? 3 : 1), 0) % 10)) % 10;
const ISBN = kern + pruef;
const KOMBI_BARCODE = `E2E-KOMBI-${s}`;

// Zweite gültige ISBN für den Idempotenz-Fall (Kern um eins verschoben).
const kern2 = String(Number(kern) + 1).padStart(12, '0');
const pruef2 =
	(10 - ([...kern2].reduce((a, d, i) => a + Number(d) * (i % 2 ? 3 : 1), 0) % 10)) % 10;
const ISBN2 = kern2 + pruef2;

test.afterAll(() => {
	// buecher_exemplare hängt per ON DELETE CASCADE am Titel.
	seedSQL(
		`DELETE FROM buecher_titel WHERE titel IN ('Listenimport ${s}', 'Kombiimport ${s}', 'Doppelklick ${s}');`
	);
});

// Der Listenimport ist additiv. Nach einer verlorenen Antwort (Timeout, Netz) drückt der
// Mensch nochmal — und hätte bis zum 07.09.2026 den Bestand verdoppelt, ohne dass es
// irgendwo stand. Mit demselben X-Idempotency-Key liefert der Server die Antwort des
// ersten Laufs; gemessen am Draht und in der Datenbank, nicht am Knopf.
test('Listenimport: derselbe Idempotenz-Schlüssel legt die Exemplare nicht ein zweites Mal an', async ({
	page
}) => {
	await uiLogin(page);
	const token = await csrfToken(page);
	const schluessel = crypto.randomUUID();
	const sende = () =>
		page.request.post('/api/books/import', {
			headers: { 'X-CSRF-Token': token, 'X-Idempotency-Key': schluessel },
			multipart: {
				file: {
					name: 'doppelklick.csv',
					mimeType: 'text/csv',
					buffer: Buffer.from(`isbn,titel,autor,bestand\n${ISBN2},Doppelklick ${s},Autor,3\n`)
				}
			}
		});

	const erster = await sende();
	expect(erster.status(), await erster.text()).toBe(200);
	const zweiter = await sende();
	expect(zweiter.status(), await zweiter.text()).toBe(200);
	expect(await zweiter.json(), 'zweite Antwort ist die gespeicherte erste').toEqual(
		await erster.json()
	);

	expect(
		querySQL(
			`SELECT count(*) FROM buecher_exemplare e JOIN buecher_titel t ON t.id = e.titel_id
			 WHERE t.titel = 'Doppelklick ${s}'`
		),
		'drei Exemplare, nicht sechs'
	).toBe('3');

	// Ein Schlüssel, der keine UUID ist, wird abgewiesen, bevor etwas passiert.
	const kaputt = await page.request.post('/api/books/import', {
		headers: { 'X-CSRF-Token': token, 'X-Idempotency-Key': 'nochmal' },
		multipart: { file: { name: 'x.csv', mimeType: 'text/csv', buffer: Buffer.from('isbn\n') } }
	});
	expect(kaputt.status()).toBe(400);
});

test('Datenverwaltung: Listenimport erzeugt B-Nummern ohne Etikett, Bestands-Import übernimmt die Nummer mit Etikett', async ({
	page
}) => {
	await uiLogin(page);
	await page.goto('/einstellungen');
	await einstellungsKategorie(page, 'Datenverwaltung').click();

	// Listenimport: eine Zeile, Bestand 2 → ein Titel, zwei Exemplare mit frischen B-Nummern.
	await page.getByTestId('listenimport-datei').setInputFiles({
		name: 'liste.csv',
		mimeType: 'text/csv',
		buffer: Buffer.from(`isbn,titel,autor,bestand\n${ISBN},Listenimport ${s},Autor,2\n`)
	});
	await page.getByRole('button', { name: 'Liste importieren' }).click();
	// 20 s statt der üblichen 5: Der Import holt je ISBN Metadaten aus dem Internet
	// (inventur/metadaten_client.go, 8 s Timeout) — die erfundene Test-ISBN läuft dort ins
	// Leere, und erst danach schreibt der Import die Dateiwerte. Beim ersten Lauf kam die
	// Antwort nach 5,0 s, exakt an der Wartegrenze vorbei (Trace: POST 200, 5.0 s).
	await expect(page.getByTestId('listenimport-ergebnis')).toHaveText(/1 Titel importiert/, {
		timeout: 20_000
	});
	expect(
		querySQL(
			`SELECT count(*) FILTER (WHERE e.barcode_id ~ '^B-[0-9]+$') || '|' || count(*) FILTER (WHERE NOT e.etikett_gedruckt) || '|' || count(*)
			 FROM buecher_exemplare e JOIN buecher_titel t ON t.id = e.titel_id WHERE t.titel = 'Listenimport ${s}'`
		),
		'zwei Exemplare, beide mit B-Nummer, beide ohne Etikett'
	).toBe('2|2|2');

	// Bestands-Import: die Nummer kommt aus der Datei und gilt als etikettiert.
	await page.getByTestId('bestandimport-datei').setInputFiles({
		name: 'bestand.csv',
		mimeType: 'text/csv',
		buffer: Buffer.from(
			`Titel;Autor;Verlag;ISBN;Jahr;Kategorie;Barcode;Zustand\nKombiimport ${s};Autor;Verlag;;2020;;${KOMBI_BARCODE};verfuegbar\n`
		)
	});
	await page.getByRole('button', { name: 'Bestand importieren' }).click();
	await expect(page.getByTestId('bestandimport-ergebnis')).toHaveText(/Kombi-Import erfolgreich/, {
		timeout: 20_000
	});
	expect(
		querySQL(
			`SELECT e.barcode_id || '|' || e.etikett_gedruckt FROM buecher_exemplare e
			 JOIN buecher_titel t ON t.id = e.titel_id WHERE t.titel = 'Kombiimport ${s}'`
		),
		'die mitgebrachte Nummer, als etikettiert vermerkt'
	).toBe(`${KOMBI_BARCODE}|true`); // Boolean in der Verkettung: 'true', nicht 't'
});
