// Was die Browser-Tests der Theke ohne Netz gemeinsam brauchen: die Warteschlange im Browser
// lesen, Leser und Bücher anlegen, scannen und an der Datenbank ablesen, was gebucht ist.
// Wie die Theke ohne Verbindung arbeitet, steht in docs/FACHKONZEPT.md 18.4.

import { expect } from '@playwright/test';
import { apiPost, querySQL, seedSQL, uiLogin, uniqueSuffix } from './helpers.js';

/** Das Band, solange kein Vorgang wartet (OfflineAnzeige.svelte). */
export const OHNE_NETZ = /Keine Verbindung — Scannen geht weiter/;

/** Lädt die Warteschlange aus IndexedDB. Läuft im Browser (page.evaluate), deshalb ohne Import. */
const LADE_WARTESCHLANGE = async () => {
	/** @param {IDBRequest} anfrage */
	const ergebnis = (anfrage) =>
		new Promise((fertig, scheitert) => {
			anfrage.onsuccess = () => fertig(anfrage.result);
			anfrage.onerror = () => scheitert(anfrage.error);
		});
	const db = /** @type {IDBDatabase} */ (await ergebnis(indexedDB.open('bibliothek-offline-db')));
	try {
		return await ergebnis(
			db.transaction('offline_actions').objectStore('offline_actions').getAll()
		);
	} finally {
		db.close();
	}
};

/**
 * Die Warteschlange, wie sie im Browser liegt (offlineWarteschlange.js), nach Scan-Zeitpunkt.
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<any[]>}
 */
export async function warteschlange(page) {
	const eintraege = await page.evaluate(LADE_WARTESCHLANGE);
	return /** @type {any[]} */ (eintraege).sort((a, b) => a.gescannt_am - b.gescannt_am);
}

/**
 * Die Kennungen der Leser, bei denen ein Exemplar offen verliehen ist; '' = bei niemandem.
 * @param {string} barcode
 */
export function verliehenAn(barcode) {
	return querySQL(
		`SELECT coalesce(string_agg(a.schueler_id::text, ',' ORDER BY a.ausgeliehen_am), '')
		 FROM ausleihen a JOIN buecher_exemplare e ON e.id = a.exemplar_id
		 WHERE e.barcode_id = '${barcode}' AND a.rueckgabe_am IS NULL;`
	);
}

/**
 * Die Ausleihen eines Exemplars als „alle/offene".
 * @param {string} barcode
 */
export function ausleihen(barcode) {
	return querySQL(
		`SELECT count(*) || '/' || count(*) FILTER (WHERE a.rueckgabe_am IS NULL)
		 FROM ausleihen a JOIN buecher_exemplare e ON e.id = a.exemplar_id
		 WHERE e.barcode_id = '${barcode}';`
	);
}

/**
 * Ein Schüler über die Tür der Leserdatei.
 * @param {import('@playwright/test').Page} page
 * @param {string} vorname
 * @param {string} s eindeutiger Teil des Namens und der Ausweisnummer
 */
export async function legeSchuelerAn(page, vorname, s) {
	const nummer = `S-${vorname.slice(0, 2).toUpperCase()}${s}`;
	const nachname = `OhneNetz-${vorname}-${s}`;
	const antwort = await apiPost(page, '/api/schueler', {
		geburtsdatum: '2012-06-15',
		vorname,
		nachname,
		klasse: '07A',
		barcode_id: nummer
	});
	expect(antwort.ok(), `Schüler anlegen: ${antwort.status()}`).toBeTruthy();
	const { id } = await antwort.json();
	return { id: /** @type {string} */ (id), nummer, nachname };
}

/**
 * Eine Lehrkraft samt Konto; die Ausweisnummer vergibt der Server.
 * @param {import('@playwright/test').Page} page
 * @param {string} s
 */
export async function legeLehrkraftAn(page, s) {
	const nachname = `OhneNetz-Lena-${s}`;
	const antwort = await apiPost(page, '/api/schueler', {
		art: 'lehrkraft',
		vorname: 'Lena',
		nachname,
		email: `lena.ohnenetz.${s.toLowerCase()}@test.local`
	});
	expect(antwort.ok(), `Lehrkraft anlegen: ${antwort.status()}`).toBeTruthy();
	const { id, barcode_id: nummer } = await antwort.json();
	expect(nummer, 'die Lehrkraft hat keine Ausweisnummer').toMatch(/^A-/);
	return { id: /** @type {string} */ (id), nummer: /** @type {string} */ (nummer), nachname };
}

/**
 * Lernmittel unter den genannten Nummern, je ein Titel; Lernmittel, damit keine Regel der
 * Bücherei die Ausleihe anhält.
 * @param {string} s
 * @param {string[]} nummern
 */
export function legeBuecherAn(s, nummern) {
	const zeilen = nummern.map((n) => `('E2E-OhneNetz ${n} ${s}', '${n}')`).join(', ');
	seedSQL(`
		WITH b (titel, nummer) AS (VALUES ${zeilen}),
		t AS (
			INSERT INTO buecher_titel (titel, ist_lernmittel) SELECT titel, true FROM b RETURNING id, titel
		)
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
		SELECT t.id, b.nummer, true FROM t JOIN b ON b.titel = t.titel;
	`);
}

/**
 * Verleiht ein Exemplar an einen Leser, seit zwei Tagen. Die gestellte Uhr läuft nur im
 * Browser: Nach dem Vorspulen geht er für den Server vor, und der Server rechnet die
 * Scan-Zeitpunkte um so viel zurück. Eine Ausleihe von eben läge dann nach dem Scan, der sie
 * beendet, und die Datenbank wiese die Rückgabe ab (check_return_date).
 * @param {string} barcode
 * @param {string} leserID
 */
export function verleihe(barcode, leserID) {
	seedSQL(`
		INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, erfasst_am, rueckgabe_frist)
		SELECT id, '${leserID}', NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days',
		       NOW() + INTERVAL '12 days'
		FROM buecher_exemplare WHERE barcode_id = '${barcode}';
	`);
}

/**
 * Entfernt, was ein Fall angelegt hat, samt Ausleihen, Meldungen und Konten der Lehrkräfte.
 * Der eindeutige Teil steht in Titeln, Nachnamen und Nummern.
 * @param {string} s
 */
export function raeumeAuf(s) {
	seedSQL(`
		BEGIN;
		CREATE TEMP TABLE weg_exemplar ON COMMIT DROP AS
			SELECT e.id FROM buecher_exemplare e JOIN buecher_titel t ON t.id = e.titel_id
			WHERE t.titel LIKE 'E2E-OhneNetz % ${s}';
		CREATE TEMP TABLE weg_leser ON COMMIT DROP AS
			SELECT id FROM leser WHERE nachname LIKE 'OhneNetz-%-${s}';
		DELETE FROM nachbuch_meldungen WHERE exemplar_id IN (SELECT id FROM weg_exemplar)
			OR ausleiher_schueler_id IN (SELECT id FROM weg_leser) OR barcode LIKE '%${s}%';
		DELETE FROM ausleihen WHERE exemplar_id IN (SELECT id FROM weg_exemplar)
			OR schueler_id IN (SELECT id FROM weg_leser);
		DELETE FROM benutzer WHERE email LIKE '%.ohnenetz.${s.toLowerCase()}@test.local';
		DELETE FROM leser WHERE id IN (SELECT id FROM weg_leser);
		DELETE FROM buecher_titel WHERE titel LIKE 'E2E-OhneNetz % ${s}';
		COMMIT;
	`);
}

/** Ein eindeutiger Teil für Namen und Nummern, in großen Buchstaben wie auf einem Ausweis. */
export function neuerTeil() {
	return uniqueSuffix().toUpperCase();
}

/**
 * Meldet an und wartet, bis das Scanfeld der Theke den Fokus hat.
 * @param {import('@playwright/test').Page} page
 */
export async function oeffneTheke(page) {
	await uiLogin(page);
	await expect(page.getByPlaceholder(/scannen/i).first()).toBeVisible();
	await expect
		.poll(() => page.evaluate(() => document.activeElement?.id ?? ''))
		.toBe('omnibox-input');
}

/**
 * Ein Scan ins Scanfeld der Theke.
 * @param {import('@playwright/test').Page} page
 * @param {string} nummer
 */
export async function scanne(page, nummer) {
	const feld = page.locator('#omnibox-input');
	await feld.fill(nummer);
	await feld.press('Enter');
}
