import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, seedSQL, uniqueSuffix } from './helpers.js';

// Hardware-Dauerfeuer: Barcode-Scanner tippen nicht wie Menschen — sie
// feuern Zeichenkette + Enter in Millisekunden. Drei Bücher direkt
// hintereinander (ohne auf Toasts zu warten) müssen alle sauber verbucht
// werden; das Backend ist über Idempotenz + Unique-Constraint (033)
// geschützt, hier geht es um das UI-Verhalten.
test('Scan-Dauerfeuer: drei Bücher in schneller Folge werden alle verbucht', async ({ page }) => {
	await uiLogin(page);
	const suffix = uniqueSuffix();

	const created = await apiPost(page, '/api/schueler', {
		vorname: 'E2E',
		nachname: `Feuer-${suffix}`,
		klasse: '7A',
		barcode_id: `S-${suffix}`
	});
	expect(created.ok(), `Schüler-Seeding: ${created.status()}`).toBeTruthy();

	seedSQL(`
        WITH t AS (
            INSERT INTO buecher_titel (titel)
            VALUES ('E2E-Feuer1-${suffix}'), ('E2E-Feuer2-${suffix}'), ('E2E-Feuer3-${suffix}')
            RETURNING id, titel
        )
        INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
        SELECT id, 'B-' || RIGHT(titel, LENGTH('Feuer1-${suffix}')), true FROM t;
    `);

	await page.getByTitle('Ausleihe').click();
	const scanInput = page.getByPlaceholder(/scannen/i).first();
	await scanInput.fill(`S-${suffix}`);
	await scanInput.press('Enter');
	await expect(page.getByText(`Feuer-${suffix}`).first()).toBeVisible();

	// Dauerfeuer: kein Warten zwischen den Scans
	for (const n of [1, 2, 3]) {
		await scanInput.fill(`B-Feuer${n}-${suffix}`);
		await scanInput.press('Enter');
	}

	// Alle drei Ausleihen müssen ankommen — Profil zählt mit
	await expect(page.getByText(`ENTLIEHENE BÜCHER (3)`)).toBeVisible({ timeout: 15000 });
	for (const n of [1, 2, 3]) {
		await expect(page.getByText(`E2E-Feuer${n}-${suffix}`).first()).toBeVisible();
	}
});
