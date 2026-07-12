import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

// LMF-Massenverlängerung (/lmf-aktionen): kritisches Massen-Update — verlängert
// alle offenen LMF-Ausleihen einer Klasse auf ein fixes Datum. Der Handler matcht
// per Projekt-Konvention über das Titel-Präfix "LMF-" (mit Bindestrich!).
test('LMF-Massenverlängerung: global extend verlängert genau die Klassen-Ausleihen', async ({
	page
}) => {
	const s = uniqueSuffix();
	const klasse = `1e${s.slice(-2)}`; // eigene Wegwerf-Klasse, kollidiert nicht mit echten Daten

	seedSQL(`
        WITH bt AS (
            INSERT INTO buecher_titel (isbn, titel, autor)
            VALUES ('978x${s}', 'LMF-Extend Testbuch ${s}', 'Autor')
            RETURNING id
        ),
        st AS (
            INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr)
            VALUES ('LMF', 'Ext1-${s}', '${klasse}', 'S-lmf1-${s}', 2030),
                   ('LMF', 'Ext2-${s}', '${klasse}', 'S-lmf2-${s}', 2030)
            RETURNING id
        ),
        ex AS (
            INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
            SELECT bt.id, 'B-lmf-${s}-' || st.id, true
            FROM bt, st
            RETURNING id, barcode_id
        )
        INSERT INTO ausleihen (exemplar_id, schueler_id, bearbeiter_id, ausgeliehen_am, rueckgabe_frist)
        SELECT ex.id, st.id, (SELECT id FROM benutzer ORDER BY id LIMIT 1), NOW(), NOW() - INTERVAL '10 days'
        FROM ex
        JOIN st ON ex.barcode_id = 'B-lmf-${s}-' || st.id;
    `);

	await uiLogin(page);
	await page.goto('/lmf-aktionen');

	await expect(page.getByRole('heading', { name: 'LMF-Massenverlängerung' })).toBeVisible();

	await page.getByLabel(/Klasse/i).fill(klasse);

	const futureDate = new Date();
	futureDate.setFullYear(futureDate.getFullYear() + 1);
	const dateStr = futureDate.toISOString().split('T')[0];
	await page.locator('input[type="date"]').fill(dateStr);

	const dialogMessages = [];
	page.on('dialog', async (dialog) => {
		dialogMessages.push(dialog.message());
		await dialog.accept();
	});

	await page.getByRole('button', { name: /verlängern/i }).click();

	// Der Erfolgs-Alert nennt die Anzahl — "Erfolgreich" allein würde auch bei
	// 0 Treffern erscheinen und wäre als Assertion wertlos.
	await expect.poll(() => dialogMessages.join(' ')).toContain('2 Ausleihen');

	// Harte DB-Verifikation: BEIDE Fristen stehen auf dem neuen Datum (23:59:59).
	const fristen = querySQL(`
        SELECT count(*) FROM ausleihen a
        JOIN schueler st ON st.id = a.schueler_id
        WHERE st.klasse = '${klasse}'
          AND a.rueckgabe_am IS NULL
          AND a.rueckgabe_frist::date = '${dateStr}'
    `);
	expect(fristen).toBe('2');
});
