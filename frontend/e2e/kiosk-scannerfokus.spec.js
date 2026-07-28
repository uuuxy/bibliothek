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
