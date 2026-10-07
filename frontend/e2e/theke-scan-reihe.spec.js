import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, seedSQL, querySQL, uniqueSuffix, scanneWieScanner } from './helpers.js';

/**
 * Ein Handscanner wartet nicht auf die Antwort des Servers. Kommt sie spät (langsames Netz,
 * wartende Datenbank), trifft der nächste Scan ein, solange die Buchung läuft. Die Theke
 * reiht ihn ein und bucht ihn danach; dieselbe Nummer noch einmal verwirft sie.
 *
 * Gescannt wird blind, ohne Klick ins Feld, und belegt wird an den Ausleihen in der
 * Datenbank. Die Antwort auf den ersten Scan kommt jeweils nach anderthalb Sekunden.
 */

/** @param {import('@playwright/test').Page} page */
const fokus = (page) => page.evaluate(() => document.activeElement?.id ?? '');

/** Ausleihen eines Exemplars als „alle/offene". @param {string} barcode */
function ausleihen(barcode) {
	return querySQL(
		`SELECT count(*) || '/' || count(*) FILTER (WHERE a.rueckgabe_am IS NULL)
		 FROM ausleihen a JOIN buecher_exemplare e ON e.id = a.exemplar_id
		 WHERE e.barcode_id = '${barcode}';`
	);
}

/** Bei wem das Exemplar offen verliehen ist (Ausweisnummer). @param {string} barcode */
function verliehenAn(barcode) {
	return querySQL(
		`SELECT coalesce(string_agg(l.barcode_id, ','), '')
		 FROM ausleihen a JOIN buecher_exemplare e ON e.id = a.exemplar_id
		 JOIN leser l ON l.id = a.schueler_id
		 WHERE e.barcode_id = '${barcode}' AND a.rueckgabe_am IS NULL;`
	);
}

/**
 * Zwei Leser und vier Lernmittel; Lernmittel, damit keine Regel der Bücherei die Ausleihe
 * anhält. Die Bücher x und y sind an Anna verliehen, p und q sind frei.
 * @param {import('@playwright/test').Page} page
 */
async function theke(page) {
	await uiLogin(page);
	const s = uniqueSuffix().toUpperCase();
	const anna = `S-RA${s}`;
	const ben = `S-RB${s}`;
	for (const [vorname, nummer] of [
		['Anna', anna],
		['Ben', ben]
	]) {
		const angelegt = await apiPost(page, '/api/schueler', {
			geburtsdatum: '2012-06-15',
			vorname,
			nachname: `Reihe-${s}`,
			klasse: '07A',
			barcode_id: nummer
		});
		expect(angelegt.ok(), `Schüler-Seeding: ${angelegt.status()}`).toBeTruthy();
	}
	const buch = { x: `B-RX${s}`, y: `B-RY${s}`, p: `B-RP${s}`, q: `B-RQ${s}` };
	seedSQL(`
		WITH t AS (
			INSERT INTO buecher_titel (titel, ist_lernmittel)
			VALUES ('E2E-Reihe X ${s}', true), ('E2E-Reihe Y ${s}', true),
			       ('E2E-Reihe P ${s}', true), ('E2E-Reihe Q ${s}', true)
			RETURNING id, titel
		), e AS (
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
			SELECT id, 'B-R' || substr(titel, 11, 1) || '${s}', true FROM t
			RETURNING id, barcode_id
		)
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		SELECT e.id, l.id, NOW() + INTERVAL '14 days'
		FROM e, leser l
		WHERE e.barcode_id IN ('${buch.x}', '${buch.y}') AND l.barcode_id = '${anna}';
	`);
	expect(ausleihen(buch.x), 'das Seeding hat x verliehen').toBe('1/1');
	expect(ausleihen(buch.y), 'das Seeding hat y verliehen').toBe('1/1');

	await expect(page.getByPlaceholder(/scannen/i).first()).toBeVisible();
	await expect.poll(() => fokus(page)).toBe('omnibox-input');
	return { s, anna, ben, buch };
}

/** @param {string} s */
function raeumeAuf(s) {
	seedSQL(`
		DELETE FROM ausleihen WHERE exemplar_id IN (
			SELECT id FROM buecher_exemplare WHERE barcode_id LIKE 'B-R_${s}');
		DELETE FROM buecher_titel WHERE titel LIKE 'E2E-Reihe _ ${s}';
		DELETE FROM leser WHERE barcode_id IN ('S-RA${s}', 'S-RB${s}');
	`);
}

/**
 * Hält die Antwort auf die erste Buchung anderthalb Sekunden zurück.
 * @param {import('@playwright/test').Page} page
 */
async function ersteAntwortKommtSpaet(page) {
	let erste = true;
	await page.route('**/api/action', async (route) => {
		if (erste) {
			erste = false;
			await new Promise((weiter) => setTimeout(weiter, 1500));
		}
		await route.continue();
	});
}

test('Schnellrückgabe: das zweite Buch wird gebucht, auch wenn die erste Antwort spät kommt', async ({
	page
}) => {
	const { s, buch } = await theke(page);
	try {
		await page.getByRole('button', { name: 'Schnellrückgabe' }).click();
		await expect(page.getByPlaceholder('Schnellrückgabe: Bücher scannen')).toBeVisible();
		await expect.poll(() => fokus(page)).toBe('omnibox-input');

		await ersteAntwortKommtSpaet(page);
		await scanneWieScanner(page, buch.x);
		await scanneWieScanner(page, buch.y);

		await expect
			.poll(() => ausleihen(buch.x), { message: 'das erste Buch ist zurück', timeout: 10_000 })
			.toBe('1/0');
		await expect
			.poll(() => ausleihen(buch.y), {
				message: 'das zweite Buch ist zurück — es wurde während der ersten Buchung gescannt',
				timeout: 10_000
			})
			.toBe('1/0');
		await expect.poll(() => fokus(page), 'der nächste Scan landet im Feld').toBe('omnibox-input');
		await expect(page.locator('#omnibox-input')).toHaveValue('');
	} finally {
		raeumeAuf(s);
	}
});

test('Ausweis und Bücher in einem Zug: die Bücher gehen an den Leser, dessen Ausweis davor kam', async ({
	page
}) => {
	const { s, ben, buch } = await theke(page);
	try {
		await ersteAntwortKommtSpaet(page);
		await scanneWieScanner(page, ben);
		await scanneWieScanner(page, buch.p);
		await scanneWieScanner(page, buch.q);

		await expect
			.poll(() => verliehenAn(buch.p), { message: 'das erste Buch ist bei Ben', timeout: 10_000 })
			.toBe(ben);
		await expect
			.poll(() => verliehenAn(buch.q), { message: 'das zweite Buch ist bei Ben', timeout: 10_000 })
			.toBe(ben);
		await expect(page.getByRole('heading', { name: new RegExp(`Ben.*Reihe-${s}`) })).toBeVisible();
	} finally {
		raeumeAuf(s);
	}
});

test('Dieselbe Nummer während der laufenden Buchung wird verworfen', async ({ page }) => {
	const { s, ben, buch } = await theke(page);
	try {
		await scanneWieScanner(page, ben);
		await expect(page.getByRole('heading', { name: new RegExp(`Ben.*Reihe-${s}`) })).toBeVisible();
		await expect.poll(() => fokus(page)).toBe('omnibox-input');

		await ersteAntwortKommtSpaet(page);
		await scanneWieScanner(page, buch.p);
		await scanneWieScanner(page, buch.p);

		// Der zweite Scan gäbe das eben geliehene Buch sonst gleich wieder zurück.
		await expect
			.poll(() => ausleihen(buch.p), { message: 'eine Ausleihe, offen', timeout: 10_000 })
			.toBe('1/1');
		await page.waitForTimeout(500);
		expect(ausleihen(buch.p), 'der Doppelscan hat ein zweites Mal gebucht').toBe('1/1');
		expect(verliehenAn(buch.p)).toBe(ben);
	} finally {
		raeumeAuf(s);
	}
});

test('Nach einem gescheiterten Scan wird das eingereihte Buch nicht gebucht und genannt', async ({
	page
}) => {
	const { s, anna, buch } = await theke(page);
	try {
		// Anna steht noch an der Theke, der nächste Ausweis ist unbekannt.
		await scanneWieScanner(page, anna);
		await expect(page.getByRole('heading', { name: new RegExp(`Anna.*Reihe-${s}`) })).toBeVisible();
		await expect.poll(() => fokus(page)).toBe('omnibox-input');

		await ersteAntwortKommtSpaet(page);
		await scanneWieScanner(page, `S-RZ${s}`);
		await scanneWieScanner(page, buch.p);

		await expect(page.getByText(/ist nicht registriert/)).toBeVisible({ timeout: 10_000 });
		await expect(
			page.getByText(new RegExp(`Gescannt und NICHT gebucht: „${buch.p}“`)),
			'das Banner nennt das Buch, das nicht gebucht wurde'
		).toBeVisible();
		await page.waitForTimeout(500);
		expect(ausleihen(buch.p), 'das Buch ist an den Leser davor gegangen').toBe('0/0');
	} finally {
		raeumeAuf(s);
	}
});
