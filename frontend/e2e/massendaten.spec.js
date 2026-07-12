import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL } from './helpers.js';

// Massendaten: realistische Obergrenze der Schule × 1,5 (2.000 Schüler) plus
// der echte Wachstumstreiber — 50.000 Ausleihen Historie über die Jahre.
// Schülerdatei, Suche und Mahnwesen müssen bedienbar bleiben.
// Alle Daten tragen das MASS-Präfix und werden am Ende wieder entfernt.

const CLEANUP = `
    DELETE FROM ausleihen WHERE exemplar_id IN
        (SELECT id FROM buecher_exemplare WHERE barcode_id LIKE 'MASS-B-%');
    DELETE FROM buecher_exemplare WHERE barcode_id LIKE 'MASS-B-%';
    DELETE FROM buecher_titel WHERE titel LIKE 'MASS-Titel%';
    DELETE FROM schueler WHERE barcode_id LIKE 'MASS-S-%';
`;

test('Massendaten: 2.000 Schüler + 50.000 Ausleihen — UI bleibt bedienbar', async ({ page }) => {
	test.setTimeout(120000);
	seedSQL(CLEANUP); // Reste eines abgebrochenen Laufs wegräumen

	try {
		seedSQL(`
            INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr)
            SELECT 'Mass', 'Schueler' || i, lpad((5 + i % 9)::text, 2, '0') || chr(97 + i % 4),
                   'MASS-S-' || i, 2030
            FROM generate_series(1, 2000) AS i;

            INSERT INTO buecher_titel (titel) VALUES ('MASS-Titel');

            INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
            SELECT (SELECT id FROM buecher_titel WHERE titel = 'MASS-Titel'),
                   'MASS-B-' || i, true
            FROM generate_series(1, 300) AS i;

            -- 50.000 abgeschlossene Ausleihen (Historie)
            WITH e AS (SELECT id, row_number() OVER (ORDER BY barcode_id) rn
                       FROM buecher_exemplare WHERE barcode_id LIKE 'MASS-B-%'),
                 s AS (SELECT id, row_number() OVER (ORDER BY barcode_id) rn
                       FROM schueler WHERE barcode_id LIKE 'MASS-S-%'),
                 b AS (SELECT id FROM benutzer ORDER BY erstellt_am LIMIT 1)
            INSERT INTO ausleihen (exemplar_id, schueler_id, bearbeiter_id,
                                   ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
            SELECT e.id, s.id, b.id,
                   NOW() - (g.i % 700 + 30) * INTERVAL '1 day',
                   NOW() - (g.i % 700 + 9) * INTERVAL '1 day',
                   NOW() - (g.i % 700) * INTERVAL '1 day'
            FROM generate_series(1, 50000) AS g(i)
            JOIN e ON e.rn = 1 + g.i % 300
            JOIN s ON s.rn = 1 + g.i % 2000
            CROSS JOIN b;

            -- 100 aktive überfällige Ausleihen für den Mahnlauf
            -- (je eigenes Exemplar — Migration 033 erlaubt nur 1 aktive pro Exemplar)
            WITH e AS (SELECT id, row_number() OVER (ORDER BY barcode_id) rn
                       FROM buecher_exemplare WHERE barcode_id LIKE 'MASS-B-%'),
                 s AS (SELECT id, row_number() OVER (ORDER BY barcode_id) rn
                       FROM schueler WHERE barcode_id LIKE 'MASS-S-%'),
                 b AS (SELECT id FROM benutzer ORDER BY erstellt_am LIMIT 1)
            INSERT INTO ausleihen (exemplar_id, schueler_id, bearbeiter_id,
                                   ausgeliehen_am, rueckgabe_frist)
            SELECT e.id, s.id, b.id, NOW() - INTERVAL '30 days', NOW() - INTERVAL '10 days'
            FROM generate_series(1, 100) AS g(i)
            JOIN e ON e.rn = g.i
            JOIN s ON s.rn = g.i
            CROSS JOIN b;
        `);

		await uiLogin(page);

		// Schülerdatei: öffnet und die Suche findet einen konkreten Schüler
		await page.getByTitle('Schülerdatei').click();
		const suche = page.getByPlaceholder('Nach Name, Klasse oder Barcode filtern...');
		await expect(suche).toBeVisible({ timeout: 15000 });
		await suche.fill('Schueler1234');
		await expect(page.getByText('Schueler1234').first()).toBeVisible({ timeout: 15000 });

		// Mahnwesen: lädt trotz 50k-Historie und zeigt die überfälligen Schüler
		await page.getByTitle('Mahnwesen').click();
		await expect(page.getByRole('row', { name: /Mass Schueler/ }).first()).toBeVisible({
			timeout: 15000
		});
	} finally {
		seedSQL(CLEANUP);
	}
});
