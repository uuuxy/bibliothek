import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, apiPatch, seedSQL, uniqueSuffix } from './helpers.js';

/**
 * Legt einen Schüler samt ausleihbarem Buch an, sperrt ihn auf die gewünschte Art und
 * öffnet sein Konto in der Ausleihe (Omnibox-Flow). Der Sperr-Dialog erscheint erst beim
 * Buch-Scan.
 *
 * Zwei Arten (entschieden am 24.09.2026): Eine Sperre von Hand hält an der Theke alles auf
 * und lässt sich nur aufheben; eine offene Forderung ist ein Hinweis, den man einmalig
 * übergeht.
 *
 * @param {import('@playwright/test').Page} page
 * @param {string} suffix
 * @param {'von-hand' | 'forderung'} art
 */
async function openBlockedStudent(page, suffix, art) {
	const studentBarcode = `S-${suffix}`;
	const bookBarcode = `B-${suffix}`;
	const bookTitle = `E2E-Sperrbuch-${suffix}`;

	const created = await apiPost(page, '/api/schueler', {
		geburtsdatum: '2012-06-15', // Pflicht seit 21.08.2026: Schlüssel für den LUSD-Abgleich
		vorname: 'E2E',
		nachname: `Gesperrt-${suffix}`,
		klasse: '7B',
		barcode_id: studentBarcode
	});
	expect(created.ok(), `Schüler-Seeding: ${created.status()}`).toBeTruthy();
	const { id: studentId } = await created.json();

	if (art === 'von-hand') {
		// Grund ist beim Sperren Pflicht (Backend-Check + DB-Constraint chk_schueler_block_reason,
		// gespiegelt in StudentLockModal). Ohne reason antwortet der Endpoint mit 400.
		const locked = await apiPatch(page, `/api/admin/students/${studentId}/lock`, {
			is_locked: true,
			reason: 'E2E: manuell gesperrt'
		});
		expect(locked.ok(), `Sperren: ${locked.status()}`).toBeTruthy();
	} else {
		// Eine offene Forderung aus einem verlorenen Buch — kein API-Weg ohne Ausleihe davor.
		seedSQL(`
			WITH t AS (INSERT INTO buecher_titel (titel) VALUES ('E2E-Verloren-${suffix}') RETURNING id),
			     e AS (INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
			           SELECT id, 'B-${suffix}-ALT', false FROM t RETURNING id)
			INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, art)
			SELECT id, '${studentId}', 'E2E: verloren', 12.00, 'nicht_zurueckgegeben' FROM e;
		`);
	}

	// Ausleihbares Buch-Exemplar seeden (kein einfacher API-Weg vorhanden)
	seedSQL(`
        WITH t AS (INSERT INTO buecher_titel (titel) VALUES ('${bookTitle}') RETURNING id)
        INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
        SELECT id, '${bookBarcode}', true FROM t;
    `);

	// Ausleihe: Schüler scannen → Profil öffnet sich
	await page.getByTitle('Ausleihe').click();
	const scanInput = page.getByPlaceholder(/scannen/i).first();
	await scanInput.fill(studentBarcode);
	await scanInput.press('Enter');
	await expect(page.getByText(`Gesperrt-${suffix}`).first()).toBeVisible();

	// Buch scannen → 403 mit Merkmal → Sperr-Dialog
	const bookInput = page.getByPlaceholder(/scannen/i).first();
	await bookInput.fill(bookBarcode);
	await bookInput.press('Enter');
	const dialog = page.getByRole('alertdialog');
	await expect(dialog.getByRole('heading', { name: 'Ausleihe blockiert' })).toBeVisible();
	// Der Handscanner schickt nach jedem Scan ein Enter: Der erste Fokus gehört „Abbrechen".
	await expect(dialog.getByRole('button', { name: 'Abbrechen' })).toBeFocused();

	return { bookTitle, dialog };
}

// Sperre von Hand: Der Dialog bietet nur das Aufheben an — kein „Einmalig ignorieren". Der
// Knopf hebt die Sperre auf (PATCH ans Backend, dieselbe Tür wie in der Akte) und holt die
// abgebrochene Ausleihe nach. Im Dialog gesucht: Hinter ihm steht die Akte des Kindes mit
// ihrem eigenen Knopf „Sperre aufheben".
test('Gesperrter Schüler: nur Aufheben, dann geht das Buch raus', async ({ page }) => {
	await uiLogin(page);
	const { bookTitle, dialog } = await openBlockedStudent(page, uniqueSuffix(), 'von-hand');

	await expect(dialog.getByRole('button', { name: /Einmalig ignorieren/ })).toHaveCount(0);
	await dialog.getByRole('button', { name: 'Sperre aufheben' }).click();
	await expect(page.getByRole('heading', { name: 'Ausleihe blockiert' })).not.toBeVisible();

	await expect(page.getByText(bookTitle).first()).toBeVisible();
});

// Offene Forderung: ein Hinweis. „Einmalig ignorieren" wiederholt den Scan mit override_block
// — die Ausleihe läuft durch, die Forderung bleibt offen.
test('Offene Forderung: Einmalig ignorieren (Override) erlaubt die Ausleihe', async ({ page }) => {
	await uiLogin(page);
	const { bookTitle, dialog } = await openBlockedStudent(page, uniqueSuffix(), 'forderung');

	await expect(dialog.getByRole('button', { name: 'Sperre aufheben' })).toHaveCount(0);
	await dialog.getByRole('button', { name: 'Einmalig ignorieren (Override)' }).click();
	await expect(page.getByRole('heading', { name: 'Ausleihe blockiert' })).not.toBeVisible();

	await expect(page.getByText(bookTitle).first()).toBeVisible();
});
