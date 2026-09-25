import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// Gemischte Auflagen an der Theke (docs/OFFEN.md 4.18, Stufe 5): Ein Kind der 7B hat die
// 3. Auflage, ein zweites bekommt die 4. Auflage desselben Buchs. Die Ausleihe geht durch, und
// über dem Konto steht die Zeile, welche Auflage die Klasse schon hat — entschieden am
// 25.09.2026: wie die Fremdrückgabe, kein Dialog.
test('Theke: Hinweis, wenn eine Klasse gemischte Auflagen bekommt', async ({ page }) => {
	const s = uniqueSuffix();
	seedSQL(`
		WITH w AS (INSERT INTO werke DEFAULT VALUES RETURNING id),
		alt AS (INSERT INTO buecher_titel (titel, auflage, erscheinungsjahr, ist_lernmittel, werk_id)
			SELECT 'E2E-Auflagen-Theke ${s}', '3. Aufl.', 2019, true, id FROM w RETURNING id),
		neu AS (INSERT INTO buecher_titel (titel, auflage, erscheinungsjahr, ist_lernmittel, werk_id)
			SELECT 'E2E-Auflagen-Theke ${s}', '4. Aufl.', 2023, true, id FROM w RETURNING id),
		ex AS (INSERT INTO buecher_exemplare (titel_id, barcode_id)
			SELECT id, 'B-AUFA${s}' FROM alt UNION ALL SELECT id, 'B-AUFN${s}' FROM neu
			RETURNING id, barcode_id),
		ida AS (INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr)
			VALUES ('Ida', 'Auflage-${s}', '7B', 'S-AUFI${s}', 2031) RETURNING id)
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		SELECT ex.id, ida.id, CURRENT_DATE + 300 FROM ex, ida WHERE ex.barcode_id = 'B-AUFA${s}';

		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr)
		VALUES ('Ben', 'Auflage-${s}', '7B', 'S-AUFB${s}', 2031);
	`);

	try {
		await uiLogin(page);
		await page.getByTitle('Ausleihe').click();
		const scan = page.getByPlaceholder(/scannen/i).first();
		await scan.fill(`S-AUFB${s}`);
		await scan.press('Enter');
		await expect(page.getByText(`Auflage-${s}`).first()).toBeVisible();

		await scan.fill(`B-AUFN${s}`);
		await scan.press('Enter');
		await expect(page.getByText(`„E2E-Auflagen-Theke ${s}" ausgeliehen an Ben.`)).toBeVisible();
		await expect(page.getByRole('status').filter({ hasText: 'Andere Auflage' })).toHaveText(
			'Andere Auflage in der 07B: 1 Kind hat 3. Aufl. · 2019 — dieses Exemplar ist 4. Aufl. · 2023.'
		);
	} finally {
		seedSQL(`
			DELETE FROM ausleihen WHERE exemplar_id IN (
				SELECT id FROM buecher_exemplare WHERE barcode_id IN ('B-AUFA${s}', 'B-AUFN${s}'));
			DELETE FROM werke WHERE id IN (
				SELECT werk_id FROM buecher_titel WHERE titel = 'E2E-Auflagen-Theke ${s}');
			DELETE FROM buecher_titel WHERE titel = 'E2E-Auflagen-Theke ${s}';
			DELETE FROM leser WHERE barcode_id IN ('S-AUFI${s}', 'S-AUFB${s}');
		`);
	}
});
