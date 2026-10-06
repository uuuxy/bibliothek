import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, seedSQL, querySQL, uniqueSuffix, scanneWieScanner } from './helpers.js';

/**
 * Ein Stapel vom Rückgabetisch (docs/OFFEN.md 4.32): zwei verliehene Bücher, ein freies
 * dazwischen und ein doppelt gescanntes.
 *
 * Ohne Schnellrückgabe lädt die erste Rückgabe den Leser des Buchs, und das freie Buch geht
 * an ihn. Mit ihr wird nur zurückgenommen. Getippt wird blind wie mit dem Handscanner, belegt
 * wird an den Ausleihen in der Datenbank.
 */

/** @param {import('@playwright/test').Page} page */
const fokus = (page) => page.evaluate(() => document.activeElement?.id ?? '');

/** Ausleihen eines Exemplars: alle und die offenen. @param {string} barcode */
function ausleihen(barcode) {
	return querySQL(
		`SELECT count(*) || '/' || count(*) FILTER (WHERE a.rueckgabe_am IS NULL)
		 FROM ausleihen a JOIN buecher_exemplare e ON e.id = a.exemplar_id
		 WHERE e.barcode_id = '${barcode}';`
	);
}

/**
 * Zwei Leser mit je einem Lernmittel auf dem Konto, dazu ein freies Buch. Lernmittel, damit
 * keine Regel der Bücherei die Ausleihe anhält.
 * @param {import('@playwright/test').Page} page
 */
async function stapel(page) {
	await uiLogin(page);
	const s = uniqueSuffix().toUpperCase();
	for (const [kind, nummer] of [
		['Anna', `S-SRA${s}`],
		['Ben', `S-SRB${s}`]
	]) {
		const angelegt = await apiPost(page, '/api/schueler', {
			geburtsdatum: '2012-06-15',
			vorname: kind,
			nachname: `Schnell-${s}`,
			klasse: '07A',
			barcode_id: nummer
		});
		expect(angelegt.ok(), `Schüler-Seeding: ${angelegt.status()}`).toBeTruthy();
	}
	seedSQL(`
		WITH t AS (
			INSERT INTO buecher_titel (titel, ist_lernmittel)
			VALUES ('E2E-Schnell X ${s}', true), ('E2E-Schnell Y ${s}', true),
			       ('E2E-Schnell F ${s}', true)
			RETURNING id, titel
		)
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
		SELECT id, 'B-SR' || substr(titel, 13, 1) || '${s}', true FROM t;
	`);
	const buch = { x: `B-SRX${s}`, y: `B-SRY${s}`, frei: `B-SRF${s}` };

	await expect(page.getByPlaceholder(/scannen/i).first()).toBeVisible();
	await expect.poll(() => fokus(page)).toBe('omnibox-input');
	for (const [nummer, kind, barcode] of [
		[`S-SRA${s}`, 'Anna', buch.x],
		[`S-SRB${s}`, 'Ben', buch.y]
	]) {
		await scanneWieScanner(page, nummer);
		await expect(
			page.getByRole('heading', { name: new RegExp(`${kind}.*Schnell-${s}`) })
		).toBeVisible();
		await scanneWieScanner(page, barcode);
		await expect
			.poll(() => ausleihen(barcode), { message: `${barcode} ist nicht verliehen` })
			.toBe('1/1');
		await page.keyboard.press('Escape');
		await expect(page.getByPlaceholder('Scannen oder Namen eingeben')).toBeVisible();
	}
	return { s, buch };
}

/** @param {string} s */
function raeumeAuf(s) {
	seedSQL(`
		DELETE FROM ausleihen WHERE exemplar_id IN (
			SELECT id FROM buecher_exemplare WHERE barcode_id LIKE 'B-SR_${s}');
		DELETE FROM buecher_titel WHERE titel LIKE 'E2E-Schnell _ ${s}';
		DELETE FROM leser WHERE barcode_id IN ('S-SRA${s}', 'S-SRB${s}');
	`);
}

test('Schnellrückgabe: ein Stapel wird nur zurückgenommen', async ({ page }) => {
	const { s, buch } = await stapel(page);
	try {
		const knopf = page.getByRole('button', { name: 'Schnellrückgabe' });
		await expect(knopf).toHaveAttribute('aria-pressed', 'false');
		await knopf.click();
		await expect(knopf).toHaveAttribute('aria-pressed', 'true');
		await expect(page.getByPlaceholder('Schnellrückgabe: Bücher scannen')).toBeVisible();
		await expect
			.poll(() => fokus(page), { message: 'der Klick hat den Fokus behalten' })
			.toBe('omnibox-input');

		// Erstes Buch: zurück, die Meldung nennt den Leser, sein Konto erscheint nicht.
		await scanneWieScanner(page, buch.x);
		await expect.poll(() => ausleihen(buch.x)).toBe('1/0');
		await expect(page.getByText(`zurückgegeben, war bei Anna Schnell-${s} (07A).`)).toBeVisible();
		await expect(page.getByRole('heading', { name: new RegExp(`Schnell-${s}`) })).toHaveCount(0);
		await expect(knopf).toHaveAttribute('aria-pressed', 'true');

		// Freies Buch: abgelehnt, nichts ausgeliehen.
		await scanneWieScanner(page, buch.frei);
		await expect(page.getByText(/nicht ausgeliehen/)).toBeVisible();
		expect(ausleihen(buch.frei), 'das freie Buch ist ausgeliehen worden').toBe('0/0');

		// Dasselbe Buch ein zweites Mal: abgelehnt, es bleibt bei der einen, beendeten Ausleihe.
		await expect.poll(() => fokus(page)).toBe('omnibox-input');
		await scanneWieScanner(page, buch.x);
		await expect(page.getByText(/nicht ausgeliehen/)).toBeVisible();
		expect(ausleihen(buch.x), 'der zweite Scan hat wieder ausgeliehen').toBe('1/0');

		// Buch eines anderen Lesers: zurück wie das erste.
		await expect.poll(() => fokus(page)).toBe('omnibox-input');
		await scanneWieScanner(page, buch.y);
		await expect.poll(() => ausleihen(buch.y)).toBe('1/0');
		await expect(page.getByText(`zurückgegeben, war bei Ben Schnell-${s} (07A).`)).toBeVisible();

		// Ein Ausweis beendet den Modus; danach leiht die Theke wieder aus.
		await expect.poll(() => fokus(page)).toBe('omnibox-input');
		await scanneWieScanner(page, `S-SRA${s}`);
		await expect(
			page.getByRole('heading', { name: new RegExp(`Anna.*Schnell-${s}`) })
		).toBeVisible();
		await expect(knopf).toHaveAttribute('aria-pressed', 'false');
		await scanneWieScanner(page, buch.frei);
		await expect.poll(() => ausleihen(buch.frei)).toBe('1/1');
	} finally {
		raeumeAuf(s);
	}
});

// Gegenprobe mit denselben Scans ohne den Knopf: Sie zeigt, dass die Probe oben eine Ausleihe
// sähe, wenn die Schnellrückgabe sie nicht verhinderte.
test('ohne Schnellrückgabe geht das freie Buch an den Leser der ersten Rückgabe', async ({
	page
}) => {
	const { s, buch } = await stapel(page);
	try {
		await scanneWieScanner(page, buch.x);
		await expect.poll(() => ausleihen(buch.x)).toBe('1/0');
		await expect(
			page.getByRole('heading', { name: new RegExp(`Anna.*Schnell-${s}`) })
		).toBeVisible();
		await expect.poll(() => fokus(page)).toBe('omnibox-input');

		await scanneWieScanner(page, buch.frei);
		await expect.poll(() => ausleihen(buch.frei)).toBe('1/1');
	} finally {
		raeumeAuf(s);
	}
});
