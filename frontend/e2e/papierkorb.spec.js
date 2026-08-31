import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, csrfToken, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

// DSGVO-Löschkette: Schüler über die UI archivieren (Tipp-Bestätigung),
// im Papierkorb wiederfinden, wiederherstellen — plus die Schutzregel,
// dass unbezahlte Schadensfälle eine Löschung hart blockieren.
test('Papierkorb: löschen mit Bestätigung, wiederherstellen, Schadensfall blockt', async ({
	page
}) => {
	await uiLogin(page);
	const suffix = uniqueSuffix();

	const created = await apiPost(page, '/api/schueler', {
		geburtsdatum: '2012-06-15', // Pflicht seit 21.08.2026: Schlüssel für den LUSD-Abgleich
		vorname: 'E2E',
		nachname: `Korb-${suffix}`,
		klasse: '8A',
		barcode_id: `S-${suffix}`
	});
	expect(created.ok(), `Schüler-Seeding: ${created.status()}`).toBeTruthy();
	const { id: studentId } = await created.json();

	// Konto öffnen → Stammdaten-Tab → Gefahrenzone
	await page.getByTitle('Ausleihe').click();
	const scanInput = page.getByPlaceholder(/scannen/i).first();
	await scanInput.fill(`S-${suffix}`);
	await scanInput.press('Enter');
	await expect(page.getByText(`Korb-${suffix}`).first()).toBeVisible();

	await page.getByRole('button', { name: 'Stammdaten & Adresse' }).click();
	await page.getByRole('button', { name: 'Schüler archivieren / löschen' }).click();

	// Tipp-Bestätigung: exakter Name als Sicherung gegen Versehen
	await page.locator('#confirm-name').fill(`E2E Korb-${suffix}`);
	await page.getByRole('button', { name: 'Endgültig archivieren/löschen' }).click();

	// Papierkorb zeigt den Gelöschten, Wiederherstellen bringt ihn zurück
	await page.getByTitle('Schülerdatei').click();
	await page.getByRole('tab', { name: 'Papierkorb' }).click();
	const zeile = page.getByRole('row', { name: new RegExp(`Korb-${suffix}`) });
	await expect(zeile).toBeVisible();
	await zeile.getByTitle('Wiederherstellen').click();
	await expect(zeile).not.toBeVisible();

	// Wiederhergestellt: Konto per Scan wieder erreichbar
	await page.getByTitle('Ausleihe').click();
	await scanInput.fill(`S-${suffix}`);
	await scanInput.press('Enter');
	await expect(page.getByText(`Korb-${suffix}`).first()).toBeVisible();

	// DSGVO-Schutzregel: unbezahlter Schadensfall blockiert die Löschung (400).
	// check_damage_item verlangt einen Exemplar-/Geräte-Bezug.
	seedSQL(`
        WITH t AS (
            INSERT INTO buecher_titel (titel) VALUES ('E2E-Korbschaden-${suffix}') RETURNING id
        ), e AS (
            INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
            SELECT id, 'B-KORB-${suffix}', false FROM t RETURNING id
        )
        INSERT INTO schadensfaelle (schueler_id, exemplar_id, beschreibung, betrag, ist_bezahlt)
        SELECT '${studentId}', e.id, 'E2E offener Schaden', 9.99, false FROM e;
    `);
	const token = await csrfToken(page);
	const del = await page.request.delete(`/api/schueler/${studentId}`, {
		headers: { 'X-CSRF-Token': token }
	});
	expect(del.status(), 'Löschung trotz offener Forderung').toBe(400);
	expect(await del.text()).toContain('unbezahlte Schadensfälle');
});

// Art.-17-Weg über die Oberfläche: Der Papierkorb bekam am 31.08.2026 den Knopf
// „Endgültig löschen" (Backend-Route existierte seit dem 24.08. ohne Aufrufer).
// Beweis in beide Richtungen: Löschung wirkt bis in die DB, und eine Server-
// Blockade (unbezahlter Schaden) erreicht den Bildschirm statt zu verschwinden.
test('Papierkorb: Endgültig löschen mit Rückfrage — und Blockade zeigt den Servertext', async ({
	page
}) => {
	await uiLogin(page);
	const suffix = uniqueSuffix();

	const created = await apiPost(page, '/api/schueler', {
		geburtsdatum: '2011-03-02',
		vorname: 'E2E',
		nachname: `Purge-${suffix}`,
		klasse: '8A',
		barcode_id: `SP-${suffix}`
	});
	expect(created.ok(), `Schüler-Seeding: ${created.status()}`).toBeTruthy();
	const { id: studentId } = await created.json();

	// In den Papierkorb (Soft-Delete) per API — der UI-Weg dorthin ist oben getestet.
	const token = await csrfToken(page);
	const del = await page.request.delete(`/api/schueler/${studentId}`, {
		headers: { 'X-CSRF-Token': token }
	});
	expect(del.ok(), `Archivieren: ${del.status()}`).toBeTruthy();

	// Blockade zuerst: unbezahlter Schaden entsteht NACH dem Archivieren (vorher
	// hätte er schon den Soft-Delete verhindert, siehe Test oben).
	seedSQL(`
        WITH t AS (
            INSERT INTO buecher_titel (titel) VALUES ('E2E-Purgeschaden-${suffix}') RETURNING id
        ), e AS (
            INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
            SELECT id, 'B-PURGE-${suffix}', false FROM t RETURNING id
        )
        INSERT INTO schadensfaelle (schueler_id, exemplar_id, beschreibung, betrag, ist_bezahlt)
        SELECT '${studentId}', e.id, 'E2E offener Schaden', 4.50, false FROM e;
    `);

	await page.getByTitle('Schülerdatei').click();
	await page.getByRole('tab', { name: 'Papierkorb' }).click();
	const zeile = page.getByRole('row', { name: new RegExp(`Purge-${suffix}`) });
	await expect(zeile).toBeVisible();

	await zeile.getByTitle('Endgültig löschen').click();
	const dialog = page.getByRole('dialog');
	await expect(dialog.getByText(`E2E Purge-${suffix}`)).toBeVisible();
	await dialog.getByRole('button', { name: 'Endgültig löschen' }).click();

	// Servertext der 409-Blockade erreicht den Bildschirm, die Zeile bleibt.
	await expect(page.getByText(/Löschen blockiert: .*unbezahlte/)).toBeVisible();
	await expect(zeile).toBeVisible();

	// Blockade auflösen, dann der echte Löschweg.
	seedSQL(`UPDATE schadensfaelle SET ist_bezahlt = true WHERE schueler_id = '${studentId}';`);
	await zeile.getByTitle('Endgültig löschen').click();
	await page.getByRole('dialog').getByRole('button', { name: 'Endgültig löschen' }).click();

	await expect(page.getByText('Schüler endgültig gelöscht.')).toBeVisible();
	await expect(zeile).not.toBeVisible();

	// Beleg an der DB: der Datensatz ist weg, nicht nur die Bildschirmzeile.
	const rest = querySQL(`SELECT count(*) FROM schueler WHERE id = '${studentId}';`).trim();
	expect(rest, 'Schülerzeile nach Purge').toBe('0');
});
