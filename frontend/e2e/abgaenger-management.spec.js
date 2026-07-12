import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// Abgänger-Management (/abgaenger): Die Liste zeigt NUR Abgänger mit offenen
// Ausleihen ("nicht entlastet"); wer nichts mehr schuldet, verschwindet.
// Der Laufzettel-PDF-Export filtert serverseitig auf die Klassen 9h/10r/13.
test('Abgänger: listet nur Abgänger mit offenen Ausleihen, erlaubt PDF-Export', async ({
	page
}) => {
	const s = uniqueSuffix();

	seedSQL(`
        WITH bt AS (
            INSERT INTO buecher_titel (isbn, titel, autor)
            VALUES ('978a${s}', 'Abgänger Buch ${s}', 'Autor')
            RETURNING id
        ),
        sA AS (
            INSERT INTO schueler (vorname, nachname, klasse, ist_abgaenger, barcode_id, abgaenger_jahr)
            VALUES ('Schuldet', 'Noch-${s}', '13', true, 'S-abg1-${s}', 2030)
            RETURNING id
        ),
        sB AS (
            INSERT INTO schueler (vorname, nachname, klasse, ist_abgaenger, barcode_id, abgaenger_jahr)
            VALUES ('Ist', 'Entlastet-${s}', '13', true, 'S-abg2-${s}', 2030)
            RETURNING id
        ),
        ex AS (
            INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
            SELECT bt.id, 'B-abg-${s}', true
            FROM bt
            RETURNING id
        )
        INSERT INTO ausleihen (exemplar_id, schueler_id, bearbeiter_id, ausgeliehen_am, rueckgabe_frist)
        SELECT ex.id, sA.id, (SELECT id FROM benutzer ORDER BY id LIMIT 1), NOW(), NOW() + INTERVAL '10 days'
        FROM ex, sA;
    `);

	await uiLogin(page);
	await page.goto('/abgaenger');

	// Schüler A (offene Ausleihe) erscheint …
	await expect(page.getByText(`Schuldet Noch-${s}`)).toBeVisible();
	// … Schüler B (entlastet, keine Ausleihe) NICHT. Erst nach dem sichtbaren
	// A-Eintrag prüfen, damit die Liste sicher fertig geladen ist.
	await expect(page.getByText(`Ist Entlastet-${s}`)).not.toBeVisible();

	// Laufzettel-PDF (Smoke): Download startet und heißt Laufzettel.pdf
	const downloadPromise = page.waitForEvent('download');
	await page.getByRole('button', { name: /Laufzettel drucken/i }).click();
	const download = await downloadPromise;
	expect(download.suggestedFilename()).toBe('Laufzettel.pdf');
});
