import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// Klassensätze zählen am Buch (docs/OFFEN.md 4.18, Stufe 5): In der 09Y1 haben vier Kinder die
// 4. und zwei die 3. Auflage desselben Buchs. Je Auflage gezählt wäre das kein Klassensatz
// (vier sind unter der Mindestzahl von fünf), am Buch sind es sechs Leser. Die Kachel zeigt die
// Auflage, die die meisten haben, und darunter, welche Auflagen die Klasse hat.
const KLASSE = '09Y1';

test('Klassensätze: ein Buch in zwei Auflagen zählt einmal und zeigt die Mischung', async ({
	page
}) => {
	const s = uniqueSuffix();
	const titel = `E2E-Klassensatz-Auflagen ${s}`;
	// Reste eines abgebrochenen Laufs zuerst: Sie zählten sonst zur Klassengröße.
	seedSQL(`
		DELETE FROM ausleihen WHERE schueler_id IN (SELECT id FROM leser WHERE barcode_id LIKE 'S-KLAU%');
		DELETE FROM leser WHERE barcode_id LIKE 'S-KLAU%';
		WITH w AS (INSERT INTO werke DEFAULT VALUES RETURNING id),
		alt AS (INSERT INTO buecher_titel (titel, auflage, erscheinungsjahr, ist_lernmittel, werk_id)
			SELECT '${titel}', '3. Aufl.', 2019, true, id FROM w RETURNING id),
		neu AS (INSERT INTO buecher_titel (titel, auflage, erscheinungsjahr, ist_lernmittel, werk_id)
			SELECT '${titel}', '4. Aufl.', 2023, true, id FROM w RETURNING id),
		kinder AS (INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr)
			SELECT 'Kind' || g, 'Auflage-${s}', '${KLASSE}', 'S-KLAU' || g || '${s}', 2030
			FROM generate_series(1, 6) g RETURNING id, barcode_id),
		ex AS (INSERT INTO buecher_exemplare (titel_id, barcode_id)
			SELECT CASE WHEN g <= 4 THEN (SELECT id FROM neu) ELSE (SELECT id FROM alt) END,
			       'B-KLAU' || g || '${s}'
			FROM generate_series(1, 6) g RETURNING id, barcode_id)
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		SELECT ex.id, kinder.id, CURRENT_DATE + 300
		FROM ex JOIN kinder ON substring(ex.barcode_id FROM 7) = substring(kinder.barcode_id FROM 7);
	`);

	try {
		await uiLogin(page);
		await page.goto('/schulklassen');
		await page.getByLabel('Klasse suchen').fill(KLASSE);
		const karte = page.locator('.class-group').filter({ hasText: KLASSE });
		await expect(karte).toHaveCount(1);
		// Bleibt nach dem Filtern eine Klasse übrig, klappt die Übersicht sie selbst auf
		// (KlassenUebersicht) — ein Klick schlösse sie wieder.
		await expect(karte.locator('button[aria-expanded]')).toHaveAttribute('aria-expanded', 'true');

		const kachel = karte.getByRole('button', { name: `${titel} bearbeiten` });
		await expect(kachel).toHaveCount(1);
		await expect(kachel.getByTestId('klassensatz-aus-ausleihen')).toHaveText(
			'aus Ausleihen · 6 Leser'
		);
		await expect(kachel.getByTestId('klassensatz-auflagen').locator('li')).toHaveText([
			'4. Aufl. · 2023: 4 Kinder — diese Auflage',
			'3. Aufl. · 2019: 2 Kinder'
		]);
	} finally {
		seedSQL(`
			DELETE FROM ausleihen WHERE exemplar_id IN (
				SELECT id FROM buecher_exemplare WHERE barcode_id LIKE 'B-KLAU%${s}');
			DELETE FROM buecher_exemplare WHERE barcode_id LIKE 'B-KLAU%${s}';
			DELETE FROM werke WHERE id IN (SELECT werk_id FROM buecher_titel WHERE titel = '${titel}');
			DELETE FROM buecher_titel WHERE titel = '${titel}';
			DELETE FROM leser WHERE barcode_id LIKE 'S-KLAU%${s}';
		`);
	}
});
