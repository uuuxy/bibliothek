import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, apiPatch, seedSQL, uniqueSuffix } from './helpers.js';

/**
 * Legt einen manuell gesperrten Schüler samt ausleihbarem Buch an und öffnet
 * dessen Konto in der Ausleihe (Omnibox-Flow). Der Block-Alert erscheint erst
 * beim Buch-Scan — der Server antwortet dann mit 403 „Manuelle Sperre".
 */
async function openBlockedStudent(page, suffix) {
	const studentBarcode = `S-${suffix}`;
	const bookBarcode = `B-${suffix}`;
	const bookTitle = `E2E-Sperrbuch-${suffix}`;

	// Schüler anlegen und manuell sperren (gleicher Endpoint wie StudentLockModal)
	const created = await apiPost(page, '/api/schueler', {
		vorname: 'E2E',
		nachname: `Gesperrt-${suffix}`,
		klasse: '7B',
		barcode_id: studentBarcode
	});
	expect(created.ok(), `Schüler-Seeding: ${created.status()}`).toBeTruthy();
	const { id: studentId } = await created.json();

	// Grund ist beim Sperren Pflicht (Backend-Check + DB-Constraint chk_schueler_block_reason,
	// gespiegelt in StudentLockModal). Ohne reason antwortet der Endpoint mit 400.
	const locked = await apiPatch(page, `/api/admin/students/${studentId}/lock`, {
		is_locked: true,
		reason: 'E2E: manuell gesperrt'
	});
	expect(locked.ok(), `Sperren: ${locked.status()}`).toBeTruthy();

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

	// Buch scannen → 403 „Manuelle Sperre" → Block-Alert-Modal
	const bookInput = page.getByPlaceholder(/scannen/i).first();
	await bookInput.fill(bookBarcode);
	await bookInput.press('Enter');
	await expect(page.getByRole('heading', { name: 'Ausleihe blockiert' })).toBeVisible();

	return { bookTitle };
}

// Smoke-Flow Sperre: „Sperre dauerhaft aufheben" entsperrt (PATCH ans Backend)
// und holt die abgebrochene Ausleihe automatisch nach. Der Button erscheint
// nur, wenn is_manually_blocked am aktiven Schüler ankommt.
test('Gesperrter Schüler: Block-Alert und Sperre aufheben', async ({ page }) => {
	await uiLogin(page);
	const { bookTitle } = await openBlockedStudent(page, uniqueSuffix());

	await page.getByRole('button', { name: 'Sperre dauerhaft aufheben' }).click();
	await expect(page.getByRole('heading', { name: 'Ausleihe blockiert' })).not.toBeVisible();

	await expect(page.getByText(bookTitle).first()).toBeVisible();
});

// Override-Pfad: „Einmalig ignorieren" lässt die Sperre bestehen, wiederholt
// den Scan aber mit override_block — die Ausleihe läuft durch.
test('Gesperrter Schüler: Einmalig ignorieren (Override) erlaubt die Ausleihe', async ({
	page
}) => {
	await uiLogin(page);
	const { bookTitle } = await openBlockedStudent(page, uniqueSuffix());

	await page.getByRole('button', { name: 'Einmalig ignorieren (Override)' }).click();
	await expect(page.getByRole('heading', { name: 'Ausleihe blockiert' })).not.toBeVisible();

	await expect(page.getByText(bookTitle).first()).toBeVisible();
});
