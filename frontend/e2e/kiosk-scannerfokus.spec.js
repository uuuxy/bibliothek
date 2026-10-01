import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

/**
 * Regression: Das Scanfeld muss nach JEDER Aktion den Fokus behalten.
 *
 * submitAction() nimmt ihn bewusst weg (blur, verhindert Doppel-Scans während der
 * Verarbeitung) — es fehlte nur das Gegenstück. Folge am Tresen: Nach dem
 * Schüler-Scan musste man vor jedem Buch erst ins Feld klicken, sonst verpuffte
 * der Scan lautlos. Kein Fehler, keine Meldung, keine Ausleihe.
 *
 * Warum das keiner der bestehenden e2e-Tests gefunden hat: Sie benutzen alle
 * `locator.fill()`, und das fokussiert das Element implizit. Damit testen sie
 * einen Pfad, den ein echter Scanner nie geht.
 *
 * Dieser Test tippt deshalb über `page.keyboard` ins Dokument — genau wie ein
 * Handscanner, der nichts anderes ist als eine Tastatur. Er darf NIEMALS
 * `scanInput.fill()` oder einen Klick ins Feld benutzen, sonst prüft er nichts.
 */
test('Handscanner: Buchscans landen ohne Klick ins Feld', async ({ page }) => {
	await uiLogin(page);

	const suffix = uniqueSuffix();
	const created = await apiPost(page, '/api/schueler', {
		geburtsdatum: '2012-06-15', // Pflicht seit 21.08.2026: Schlüssel für den LUSD-Abgleich
		vorname: 'E2E',
		nachname: `Fokus-${suffix}`,
		klasse: '7A',
		barcode_id: `S-${suffix}`
	});
	expect(created.ok(), `Schüler-Seeding: ${created.status()}`).toBeTruthy();

	seedSQL(`
        WITH t AS (
            INSERT INTO buecher_titel (titel)
            VALUES ('E2E-Fokus1-${suffix}'), ('E2E-Fokus2-${suffix}')
            RETURNING id, titel
        )
        INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
        SELECT id, 'B-' || RIGHT(titel, LENGTH('Fokus1-${suffix}')), true FROM t;
    `);

	// Kein Klick auf „Ausleihe": Nach dem Login IST das der aktive Bildschirm, und ein
	// Klick auf den Menüpunkt zöge den Fokus auf den Nav-Button. Stattdessen wird die
	// Bereitschaft abgewartet — der Kiosk fokussiert das Scanfeld beim Laden selbst.
	await expect(page.getByPlaceholder(/scannen/i).first()).toBeVisible();

	/** Tippt blind ins Dokument und schließt mit Enter ab — wie ein Handscanner. */
	const scanne = async (/** @type {string} */ code) => {
		await page.keyboard.type(code, { delay: 5 });
		await page.keyboard.press('Enter');
	};
	/**
	 * Wartet, bis das Scanfeld den Fokus hat. Bewusst pollend: Die Refokussierung
	 * läuft erst, nachdem Svelte das Profil neu gerendert hat — eine Einmal-Prüfung
	 * direkt nach dem Enter wäre ein Rennen gegen den Renderer.
	 */
	const erwarteFokusImScanfeld = (/** @type {string} */ wann) =>
		expect
			.poll(() => page.evaluate(() => document.activeElement?.id ?? ''), {
				message: `Fokus ${wann} verloren — ein Handscanner tippt danach ins Nichts`,
				timeout: 5000
			})
			.toBe('omnibox-input');

	await erwarteFokusImScanfeld('beim Öffnen des Kiosks');

	// Schüler — danach ist isActive true, und genau dort griff die alte
	// Refokussierung nicht mehr.
	await scanne(`S-${suffix}`);
	await expect(page.getByText(`Fokus-${suffix}`).first()).toBeVisible();
	await erwarteFokusImScanfeld('nach dem Schüler-Scan');

	// Zwei Bücher hintereinander, ohne die Maus anzufassen.
	for (const n of [1, 2]) {
		await scanne(`B-Fokus${n}-${suffix}`);
		await expect
			.poll(
				() =>
					querySQL(
						`SELECT count(*) FROM ausleihen a
                         JOIN buecher_exemplare e ON e.id = a.exemplar_id
                         WHERE a.rueckgabe_am IS NULL AND e.barcode_id = 'B-Fokus${n}-${suffix}';`
					),
				{ message: `Buch ${n} wurde nicht verbucht — Scan ist ins Leere gelaufen` }
			)
			.toBe('1');
		await erwarteFokusImScanfeld(`nach Buch ${n}`);
	}

	// Auch ein Fehlschlag darf das Feld nicht taub zurücklassen: sonst steht die
	// Ausleihe nach einem verschmutzten Etikett still, bis jemand klickt.
	await scanne('B-GIBTESGARNICHT-0000');
	await expect(page.getByText(/nicht gefunden/i).first()).toBeVisible();
	await erwarteFokusImScanfeld('nach einem Fehlscan');
});

// Ein Klick in der Akte lässt den Fokus auf dem Reiter, auf dem Knopf oder nirgends stehen; der
// nächste Scan verpuffte dann wie oben. Die Theke lenkt das getippte Zeichen deshalb ins
// Scanfeld (scanOhneFokus.js). Belegt wird es an der Ausleihe in der Datenbank.
test.describe('Handscanner: der Fokus steht woanders', () => {
	// Leser mit einer offenen Forderung und drei Lernmitteln; der Leser ist gescannt. Lernmittel,
	// weil die offene Forderung eine Bücherei-Ausleihe anhält — der Scan soll hier buchen.
	async function thekeMitLeser(page) {
		await uiLogin(page);
		const suffix = uniqueSuffix();
		const created = await apiPost(page, '/api/schueler', {
			geburtsdatum: '2012-06-15',
			vorname: 'E2E',
			nachname: `Reiter-${suffix}`,
			klasse: '7A',
			barcode_id: `S-${suffix}`
		});
		expect(created.ok(), `Schüler-Seeding: ${created.status()}`).toBeTruthy();
		const { id } = await created.json();
		seedSQL(`
			WITH t AS (
				INSERT INTO buecher_titel (titel, ist_lernmittel)
				VALUES ('E2E-Reiter1-${suffix}', true), ('E2E-Reiter2-${suffix}', true),
				       ('E2E-Reiter3-${suffix}', true)
				RETURNING id, titel
			)
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
			SELECT id, 'B-' || RIGHT(titel, LENGTH('Reiter1-${suffix}')), true FROM t;
			WITH t AS (
				INSERT INTO buecher_titel (titel) VALUES ('E2E-Reiter-Schaden-${suffix}') RETURNING id
			), e AS (
				INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
				SELECT id, 'B-RS${suffix}', true FROM t RETURNING id
			)
			INSERT INTO schadensfaelle (schueler_id, exemplar_id, beschreibung, betrag)
			SELECT '${id}', e.id, 'E2E Schaden', 12.00 FROM e;
		`);

		await expect(page.getByPlaceholder(/scannen/i).first()).toBeVisible();
		await expect.poll(() => fokus(page)).toBe('omnibox-input');
		await page.keyboard.type(`S-${suffix}`, { delay: 5 });
		await page.keyboard.press('Enter');
		await expect(page.getByText(`Reiter-${suffix}`).first()).toBeVisible();
		await page.getByRole('tab', { name: /Gebühren & Schäden/ }).click();
		return suffix;
	}

	const fokus = (page) => page.evaluate(() => document.activeElement?.id ?? '');
	const verbucht = (suffix, n) =>
		querySQL(
			`SELECT count(*) FROM ausleihen a
			 JOIN buecher_exemplare e ON e.id = a.exemplar_id
			 WHERE a.rueckgabe_am IS NULL AND e.barcode_id = 'B-Reiter${n}-${suffix}';`
		);

	test('der Scan wird verbucht: auf dem Reiter, nach „Bezahlt", nach einem Klick ins Leere', async ({
		page
	}) => {
		const suffix = await thekeMitLeser(page);

		/** Blind scannen. Vorher steht der Fokus nicht im Scanfeld — sonst prüfte der Schritt nichts. */
		const scanne = async (n, wo) => {
			expect(await fokus(page), `Fokus ${wo}`).not.toBe('omnibox-input');
			await page.keyboard.type(`B-Reiter${n}-${suffix}`, { delay: 5 });
			await page.keyboard.press('Enter');
			await expect
				.poll(() => verbucht(suffix, n), { message: `Der Scan ${wo} wurde nicht verbucht` })
				.toBe('1');
			// Nach der Buchung gibt die Theke dem Scanfeld den Fokus zurück; erst danach der
			// nächste Klick, damit ihr Zeitgeber ihn nicht nachträglich verdeckt.
			await expect.poll(() => fokus(page)).toBe('omnibox-input');
		};

		await scanne(1, 'auf dem Reiter');

		// Der Knopf verschwindet mit der Zahlung, der Fokus fällt auf die Seite.
		await page.getByRole('button', { name: 'Bezahlt' }).click();
		await expect(page.getByText('Zahlung verbucht.')).toBeVisible();
		await scanne(2, 'nach „Bezahlt"');

		await page.getByRole('heading', { name: new RegExp(`Reiter-${suffix}`) }).click();
		await scanne(3, 'nach einem Klick neben die Knöpfe');
	});

	// Die Gegenrichtung: Ein offener Dialog liegt über dem Scanfeld und behält seine Eingabe.
	// Buchte der Scan dahinter, entstünde eine Ausleihe, während jemand einen Storno begründet.
	test('ein offener Dialog der Akte behält die Tastatur', async ({ page }) => {
		const suffix = await thekeMitLeser(page);
		await page.getByRole('button', { name: 'Stornieren' }).click();
		await expect(page.getByRole('dialog', { name: 'Gebühr wirklich stornieren?' })).toBeVisible();

		await page.keyboard.type(`B-Reiter1-${suffix}`, { delay: 5 });
		await page.keyboard.press('Enter');
		await page.waitForTimeout(1500);
		expect(await page.locator('#omnibox-input').inputValue()).toBe('');
		expect(verbucht(suffix, 1), 'Der Scan wurde hinter dem Dialog verbucht').toBe('0');
	});
});
