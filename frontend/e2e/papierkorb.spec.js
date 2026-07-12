import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, csrfToken, seedSQL, uniqueSuffix } from './helpers.js';

// DSGVO-Löschkette: Schüler über die UI archivieren (Tipp-Bestätigung),
// im Papierkorb wiederfinden, wiederherstellen — plus die Schutzregel,
// dass unbezahlte Schadensfälle eine Löschung hart blockieren.
test('Papierkorb: löschen mit Bestätigung, wiederherstellen, Schadensfall blockt', async ({
	page
}) => {
	await uiLogin(page);
	const suffix = uniqueSuffix();

	const created = await apiPost(page, '/api/schueler', {
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
	await page.getByRole('button', { name: 'Papierkorb' }).click();
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
