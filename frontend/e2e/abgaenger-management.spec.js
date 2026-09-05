import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// Abgänger-Ansicht (/abgaenger): die Abschlussklassen (9H, 10R, 13) mit offenen
// Ausleihen — noch an der Schule, zum Einsammeln vor der Entlassung. Wer nichts mehr
// schuldet, verschwindet; wer laut LUSD schon weg ist, steht hier nie (Mahnwesen).
//
// Die Liste hat eine Saison (01.05.–31.07.), die der Server bestimmt. Dieser Test kann
// den Kalender nicht stellen, prüft aber in JEDEM Zustand etwas Echtes: in der Saison
// die Zeilen und den PDF-Export, außerhalb den Hinweis — und dass ein Schüler der 9H1
// mit offenem Buch dann trotzdem NICHT erscheint. Beide Zustände deterministisch:
// api/graduates_pg_test.go mit gestellter Uhr.
test('Abgänger: Abschlussklasse mit offenem Buch — Saison entscheidet über Liste und PDF', async ({
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
            INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr)
            VALUES ('Schuldet', 'Noch-${s}', '9H1', 'S-abg1-${s}', 2030)
            RETURNING id
        ),
        sB AS (
            INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr)
            VALUES ('Ist', 'Entlastet-${s}', '9H1', 'S-abg2-${s}', 2030)
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
	const antwort = await (await page.request.get('/api/abgaenger')).json();
	await page.goto('/schuljahr/abgaenger');

	if (!antwort.fenster.offen) {
		// Außerhalb der Saison: Hinweis statt Liste — und der 9H1-Schüler mit Buch fehlt.
		await expect(page.getByText('Abschlussklassen erscheinen hier ab Mai')).toBeVisible();
		await expect(page.getByText(`Schuldet Noch-${s}`)).toHaveCount(0);
		expect(antwort.abgaenger).toEqual([]);
		expect((await page.request.get('/api/abgaenger/pdf')).status()).toBe(404);
		return;
	}

	// Schüler A (offene Ausleihe) erscheint …
	await expect(page.getByText(`Schuldet Noch-${s}`)).toBeVisible();
	// … Schüler B (entlastet, keine Ausleihe) NICHT. Erst nach dem sichtbaren
	// A-Eintrag prüfen, damit die Liste sicher fertig geladen ist.
	await expect(page.getByText(`Ist Entlastet-${s}`)).not.toBeVisible();

	// Kontoauszug-PDF (Smoke): Der frühere „Laufzettel" ist längst der Kontoauszug mit
	// Freigabezeile — jetzt heißt auch der Knopf so.
	const downloadPromise = page.waitForEvent('download');
	await page.getByRole('button', { name: /Kontoauszüge drucken/i }).click();
	const download = await downloadPromise;
	expect(download.suggestedFilename()).toBe('Kontoauszuege_Abgaenger.pdf');
});
