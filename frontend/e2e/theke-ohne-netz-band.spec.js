import { test, expect } from '@playwright/test';
import { ADMIN_PASSWORD, seedSQL, uiLogin } from './helpers.js';
import {
	OHNE_NETZ,
	legeBuecherAn,
	legeLehrkraftAn,
	legeSchuelerAn,
	neuerTeil,
	oeffneTheke,
	raeumeAuf,
	scanne,
	verleihe,
	verliehenAn,
	warteschlange
} from './ohne-netz.js';

/**
 * Fällt die Verbindung aus, scannt die Theke weiter: Ein Band oben sagt es, ein Ausweis wird als
 * Nummer gemerkt, und die folgenden Bücher gehen mit ihm in die Warteschlange, jedes in der
 * Form, die auf ihm klebt. Gesperrt wird ohne Netz nicht, denn aufgeschlossen wird am Server.
 * Kommt die Verbindung zurück, bucht der Rechner erst nach und sperrt dann; was beim Nachbuchen
 * anders lief als beim Scan, steht als Meldung an jedem Arbeitsplatz (docs/FACHKONZEPT.md 18.4).
 */

/** Eine Zahl, wie sie als nackte Exemplarnummer im Altbestand steht: sieben Ziffern. */
function nackteNummer() {
	return String(1_000_000 + Math.floor(Math.random() * 8_999_999));
}

/**
 * Was der Strichcode eines Littera-Etiketts liefert: die Nummer, auf acht Stellen mit Nullen,
 * eine Bibliotheksnummer, die Stellenzahl und die Prüfziffer der EAN-13.
 * @param {string} nummer
 */
function litteraEtikett(nummer) {
	const ohnePruefziffer = `${nummer.padEnd(8, '0')}123${nummer.length}`;
	let summe = 0;
	for (const [i, z] of [...ohnePruefziffer].entries()) summe += Number(z) * (i % 2 === 1 ? 3 : 1);
	return ohnePruefziffer + ((10 - (summe % 10)) % 10);
}

/**
 * Was ein Lesegerät von einem älteren Code-39-Etikett liefert: die Nummer und ihr
 * Mod-43-Prüfzeichen. Gewählt wird eine Nummer, deren Zeichen ein Buchstabe ist; ein
 * Leerzeichen als Prüfzeichen läse die Theke als Suche.
 */
function code39MitPruefzeichen() {
	const zeichensatz = '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ-. $/+%';
	for (;;) {
		const nummer = nackteNummer();
		const summe = [...nummer].reduce((s, z) => s + zeichensatz.indexOf(z), 0);
		const zeichen = zeichensatz[summe % 43];
		if (/[A-Z]/.test(zeichen)) return { nummer, scan: nummer + zeichen };
	}
}

/**
 * Die Buchnummern, die der Rechner bei der Anmeldung geholt hat (buchBarcodes.svelte.js).
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<string[]>}
 */
function buchliste(page) {
	return page.evaluate(
		() =>
			new Promise((fertig, scheitert) => {
				const auf = indexedDB.open('bibliothek-barcodes-db');
				auf.onerror = () => scheitert(auf.error);
				auf.onsuccess = () => {
					const db = auf.result;
					if (!db.objectStoreNames.contains('liste')) {
						db.close();
						fertig([]);
						return;
					}
					const lesen = db.transaction('liste').objectStore('liste').get('aktuell');
					lesen.onsuccess = () => {
						db.close();
						fertig(lesen.result?.barcodes ?? []);
					};
					lesen.onerror = () => scheitert(lesen.error);
				};
			})
	);
}

test('Ohne Netz: alle Buchformen und zwei Ausweise im Band, keine Sperre, danach nachgebucht und an einem zweiten Platz gemeldet', async ({
	page,
	context,
	browser
}) => {
	test.setTimeout(90_000);
	const s = neuerTeil();
	const nackt = nackteNummer();
	const littera = nackteNummer();
	const code39 = code39MitPruefzeichen();
	const buch = { b: `B-OB${s}`, alt: `B-OL${s}`, z: `B-OZ${s}` };
	// Das Exemplar alt trägt eine kurze Nummer und klebt noch mit seinem alten LMF-Etikett.
	const altesEtikett = `LMF-OL${s}`;
	// Die nackten Nummern stehen nur auf der Buchliste, und die holt der Rechner bei der Anmeldung.
	legeBuecherAn(s, [buch.b, buch.alt, nackt, littera, code39.nummer, buch.z]);
	seedSQL(
		`UPDATE buecher_exemplare SET lmf_barcode = '${altesEtikett}' WHERE barcode_id = '${buch.alt}';`
	);

	const zweiter = await browser.newContext();
	try {
		await page.clock.install();
		await oeffneTheke(page);
		const anna = await legeSchuelerAn(page, 'Anna', s);
		const ben = await legeSchuelerAn(page, 'Ben', s);
		const lena = await legeLehrkraftAn(page, s);
		verleihe(buch.z, ben.id);
		await expect
			.poll(() => buchliste(page))
			.toEqual(expect.arrayContaining([nackt, littera, code39.nummer]));

		// Ein zweiter Arbeitsplatz, angemeldet und mit Netz.
		const platz2 = await zweiter.newPage();
		await uiLogin(platz2);
		const meldungenVorher = (
			await (await platz2.request.get('/api/action/nachbuch-meldungen/anzahl')).json()
		).offen;

		await context.setOffline(true);
		await expect(page.getByText(OHNE_NETZ)).toBeVisible();

		/** @type {[string, string][]} */
		const reihe = [
			[anna.nummer, ''],
			[buch.b, buch.b],
			[altesEtikett, altesEtikett],
			[nackt, nackt],
			[lena.nummer, ''],
			[litteraEtikett(littera), littera],
			[code39.scan, code39.nummer],
			[buch.z, buch.z]
		];
		let abgelegt = 0;
		for (const [scan, nummer] of reihe) {
			await scanne(page, scan);
			if (nummer === '') {
				await expect(page.getByText(`Ohne Netz gemerkt: Ausweis „${scan}"`)).toBeVisible();
			} else {
				abgelegt++;
				await expect.poll(() => warteschlange(page).then((q) => q.length)).toBe(abgelegt);
			}
		}

		const ausleihe = (/** @type {string} */ barcode, /** @type {string} */ ausweis) =>
			expect.objectContaining({
				art: 'ausleihe',
				barcode,
				leser_id: null,
				ausweis_barcode: ausweis
			});
		expect(await warteschlange(page)).toEqual([
			ausleihe(buch.b, anna.nummer),
			ausleihe(altesEtikett, anna.nummer),
			ausleihe(nackt, anna.nummer),
			ausleihe(littera, lena.nummer),
			ausleihe(code39.nummer, lena.nummer),
			ausleihe(buch.z, lena.nummer)
		]);

		// Ein Band, kein Vollbild: Das Scanfeld liegt frei und nimmt den nächsten Klick an.
		await expect(page.getByText(/6\s+Vorgänge nur auf diesem Rechner/)).toBeVisible();
		await expect(page.getByRole('dialog')).toHaveCount(0);
		await page.locator('#omnibox-input').click({ trial: true, timeout: 5000 });

		// Weit über die Frist der Sperre (Vorgabe 15 Minuten): ohne Netz keine Sperre.
		await page.clock.fastForward('20:00');
		await expect(page.getByText(/6\s+Vorgänge nur auf diesem Rechner/)).toBeVisible();
		await expect(page.getByTestId('sperrbildschirm')).toHaveCount(0);

		// Die Verbindung kommt zurück: erst nachbuchen, dann die fällige Sperre. Hinter der Sperre
		// nähme der Server das Nachbuchen nicht an; seine Antwort kommt deshalb eine Sekunde später,
		// damit die Reihenfolge nicht vom Zufall abhängt.
		await page.route('**/api/action/nachbuchen', async (route) => {
			await new Promise((weiter) => setTimeout(weiter, 1000));
			await route.continue();
		});
		await context.setOffline(false);
		await expect(page.getByTestId('sperrbildschirm')).toBeVisible();
		for (const nummer of [buch.b, buch.alt, nackt]) {
			await expect.poll(() => verliehenAn(nummer), { timeout: 10_000 }).toBe(anna.id);
		}
		for (const nummer of [littera, code39.nummer, buch.z]) {
			await expect.poll(() => verliehenAn(nummer), { timeout: 10_000 }).toBe(lena.id);
		}
		await expect.poll(() => warteschlange(page)).toEqual([]);

		await page.locator('#sperre-passwort').fill(ADMIN_PASSWORD);
		await page.getByRole('button', { name: 'Entsperren' }).click();
		await expect(page.getByTestId('sperrbildschirm')).toHaveCount(0);

		// Das Buch lag bei Ben und ging an Lena: Das sieht der zweite Platz, ohne neu zu laden.
		const knopf = platz2.getByRole('button', { name: `Meldungen (${meldungenVorher + 1})` });
		await expect(knopf).toBeVisible({ timeout: 10_000 });
		await knopf.click();
		const zeile = platz2.getByRole('row', { name: new RegExp(buch.z) });
		await expect(zeile).toContainText('lag bei jemand anderem');
		await expect(zeile).toContainText(`vorher: Ben ${ben.nachname}`);
	} finally {
		await zweiter.close();
		raeumeAuf(s);
	}
});
