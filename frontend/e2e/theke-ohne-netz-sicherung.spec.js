import { test, expect } from '@playwright/test';
import { einstellungsKategorie } from './helpers.js';
import {
	ausleihen,
	legeBuecherAn,
	legeSchuelerAn,
	neuerTeil,
	oeffneTheke,
	raeumeAuf,
	verleihe,
	verliehenAn,
	warteschlange
} from './ohne-netz.js';

/**
 * Bleibt ein Theken-Rechner ohne Verbindung, sichert er seine Warteschlange als Datei, und ein
 * anderer Rechner spielt sie unter Datenverwaltung ein. Der Schlüssel jedes Vorgangs wandert
 * mit: Was zweimal ankommt, bucht der Server einmal. Ältere Rechner schreiben Format 1
 * (action_type, barcode_id, schueler_id); auch das wird nachgebucht.
 */

/** @type {string[]} */
const angelegt = [];
test.afterEach(() => {
	for (const s of angelegt.splice(0)) raeumeAuf(s);
});

/**
 * Wählt unter Datenverwaltung die Sicherungsdateien aus, wie es ein Mensch im Dialog tut.
 * @param {import('@playwright/test').Page} page
 * @param {Record<string, unknown[]>} dateien Name der Datei → ihr Inhalt
 */
async function spieleEin(page, dateien) {
	const dialog = page.waitForEvent('filechooser');
	await page.getByRole('button', { name: 'Sicherungsdateien auswählen' }).click();
	await (
		await dialog
	).setFiles(
		Object.entries(dateien).map(([name, inhalt]) => ({
			name,
			mimeType: 'application/json',
			buffer: Buffer.from(JSON.stringify(inhalt))
		}))
	);
}

/**
 * Merkt sich, welche Schlüssel an die Nachbuch-Tür gingen und was sie zu jedem antwortete.
 * @param {import('@playwright/test').Page} page
 */
function beobachteNachbuchen(page) {
	/** @type {{ gesendet: string[], ergebnisse: string[] }} */
	const lauf = { gesendet: [], ergebnisse: [] };
	page.on('request', (anfrage) => {
		if (!anfrage.url().endsWith('/api/action/nachbuchen')) return;
		for (const e of anfrage.postDataJSON()?.eintraege ?? []) lauf.gesendet.push(e.schluessel);
	});
	page.on('response', async (antwort) => {
		if (!antwort.url().endsWith('/api/action/nachbuchen') || !antwort.ok()) return;
		for (const e of (await antwort.json())?.ergebnisse ?? []) lauf.ergebnisse.push(e.ergebnis);
	});
	return lauf;
}

test('Zwei Sicherungen auf einmal: jeder Vorgang geht einmal hinaus, und eine Datei zweimal bucht nichts doppelt', async ({
	page
}) => {
	await oeffneTheke(page);
	const s = neuerTeil();
	angelegt.push(s);
	const anna = await legeSchuelerAn(page, 'Anna', s);
	const ben = await legeSchuelerAn(page, 'Ben', s);
	const [b1, b2, b3, b4] = ['B-OS1', 'B-OS2', 'B-OS3', 'B-OS4'].map((n) => n + s);
	legeBuecherAn(s, [b1, b2, b3, b4]);
	verleihe(b4, ben.id);

	const jetzt = Date.now();
	const [k1, k2, k3, k4] = Array.from({ length: 4 }, () => crypto.randomUUID());
	const rechnerA = [
		{ id: k1, art: 'ausleihe', barcode: b1, leser_id: anna.id, gescannt_am: jetzt },
		{ id: k2, art: 'ausleihe', barcode: b2, leser_id: anna.id, gescannt_am: jetzt + 1 }
	];
	const rechnerB = [
		{ id: k3, art: 'ausleihe', barcode: b3, leser_id: ben.id, gescannt_am: jetzt + 2 },
		{ id: k4, art: 'rueckgabe', barcode: b4, leser_id: null, gescannt_am: jetzt + 3 }
	];

	const lauf = beobachteNachbuchen(page);
	await page.goto('/einstellungen');
	await einstellungsKategorie(page, 'Datenverwaltung').click();
	await spieleEin(page, { 'rechner-a.json': rechnerA, 'rechner-b.json': rechnerB });
	await expect(page.getByText('4 Vorgang/Vorgänge aus 2 Datei(en) übernommen')).toBeVisible();

	await expect.poll(() => warteschlange(page), { timeout: 10_000 }).toEqual([]);
	expect(verliehenAn(b1)).toBe(anna.id);
	expect(verliehenAn(b2)).toBe(anna.id);
	expect(verliehenAn(b3)).toBe(ben.id);
	expect(verliehenAn(b4)).toBe('');
	expect(lauf.gesendet.toSorted(), 'jeder Schlüssel genau einmal an die Nachbuch-Tür').toEqual(
		[k1, k2, k3, k4].toSorted()
	);
	expect(lauf.ergebnisse.toSorted()).toEqual([
		'ausgeliehen',
		'ausgeliehen',
		'ausgeliehen',
		'zurueckgegeben'
	]);

	// Dieselbe Datei noch einmal: Zu einem Schlüssel, den er schon gebucht hat, wiederholt der
	// Server seine Antwort und bucht nichts (api/theke/nachbuchen_schluessel.go).
	lauf.ergebnisse.length = 0;
	await spieleEin(page, { 'rechner-a.json': rechnerA });
	await expect(page.getByText('2 Vorgang/Vorgänge aus 1 Datei(en) übernommen')).toBeVisible();
	await expect.poll(() => warteschlange(page), { timeout: 10_000 }).toEqual([]);
	await expect.poll(() => lauf.ergebnisse).toEqual(['ausgeliehen', 'ausgeliehen']);
	expect(ausleihen(b1)).toBe('1/1');
	expect(ausleihen(b2)).toBe('1/1');
});

test('Eine Sicherung in Format 1 eines anderen Rechners wird nachgebucht', async ({ page }) => {
	await oeffneTheke(page);
	const s = neuerTeil();
	angelegt.push(s);
	const anna = await legeSchuelerAn(page, 'Anna', s);
	const [frei, verliehen] = ['B-OS5', 'B-OS6'].map((n) => n + s);
	legeBuecherAn(s, [frei, verliehen]);
	verleihe(verliehen, anna.id);

	const jetzt = Date.now();
	const format1 = [
		{
			id: crypto.randomUUID(),
			action_type: 'checkout',
			barcode_id: frei,
			schueler_id: anna.id,
			timestamp: jetzt
		},
		{
			id: crypto.randomUUID(),
			action_type: 'checkin',
			barcode_id: verliehen,
			schueler_id: null,
			timestamp: jetzt + 1
		}
	];

	await page.goto('/einstellungen');
	await einstellungsKategorie(page, 'Datenverwaltung').click();
	await spieleEin(page, { 'offline_scans_backup_alt.json': format1 });
	await expect(page.getByText('2 Vorgang/Vorgänge aus 1 Datei(en) übernommen')).toBeVisible();

	await expect.poll(() => verliehenAn(frei), { timeout: 10_000 }).toBe(anna.id);
	await expect.poll(() => verliehenAn(verliehen)).toBe('');
	await expect.poll(() => warteschlange(page)).toEqual([]);
});
