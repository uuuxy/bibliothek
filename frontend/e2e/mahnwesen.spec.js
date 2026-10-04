import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// Mahnwesen: überfällige Ausleihe erscheint in der Übersicht, und die
// Mahnliste kommt als echtes PDF (Smoke-Assert statt visueller Prüfung).
test('Mahnwesen: überfälliger Schüler erscheint, Mahnliste-PDF antwortet', async ({ page }) => {
	await uiLogin(page);
	const suffix = uniqueSuffix();

	seedSQL(`
        WITH s AS (
            INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr)
            VALUES ('E2E', 'Saeumig-${suffix}', '8M', 'S-${suffix}', 2030) RETURNING id
        ), t AS (
            INSERT INTO buecher_titel (titel) VALUES ('E2E-Mahnbuch-${suffix}') RETURNING id
        ), e AS (
            INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
            SELECT id, 'B-${suffix}', true FROM t RETURNING id
        )
        INSERT INTO ausleihen (exemplar_id, schueler_id, bearbeiter_id, ausgeliehen_am, rueckgabe_frist)
        SELECT e.id, s.id, (SELECT id FROM benutzer ORDER BY erstellt_am LIMIT 1), NOW() - INTERVAL '30 days', NOW() - INTERVAL '10 days' FROM e, s;
    `);

	await page.getByTitle('Mahnwesen').click();
	// Die Tabelle listet Schüler (Medien nur als Zähler, keine Titel)
	const zeile = page.getByRole('row', { name: new RegExp(`Saeumig-${suffix}`) });
	await expect(zeile).toBeVisible();
	await expect(zeile).toContainText('8M');

	// PDF-Smoke: Status, Content-Type und nicht-leerer Body genügen —
	// visuelle PDF-Prüfung lohnt den Wartungsaufwand nicht.
	const pdf = await page.request.get('/api/mahnwesen/pdf');
	expect(pdf.status(), 'Mahnliste-PDF Status').toBe(200);
	expect(pdf.headers()['content-type']).toContain('application/pdf');
	expect((await pdf.body()).length).toBeGreaterThan(1000);
});

// Der Weg der Mahnbriefe: Kind anhaken, „Mahnbriefe drucken", danach nennt die Liste die
// Mahnung. Ohne Auswahl steht kein Knopf da, der Briefe für alle druckt.
test('Mahnbrief: aus der Auswahl gedruckt, danach steht die Mahnung in der Liste', async ({
	page
}) => {
	await uiLogin(page);
	const suffix = uniqueSuffix();

	seedSQL(`
        WITH s AS (
            INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr)
            VALUES ('E2E', 'Briefkind-${suffix}', '8M', 'SB-${suffix}', 2030) RETURNING id
        ), t AS (
            INSERT INTO buecher_titel (titel) VALUES ('E2E-Briefbuch-${suffix}') RETURNING id
        ), e AS (
            INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
            SELECT id, 'BB-${suffix}', true FROM t RETURNING id
        )
        INSERT INTO ausleihen (exemplar_id, schueler_id, bearbeiter_id, ausgeliehen_am, rueckgabe_frist)
        SELECT e.id, s.id, (SELECT id FROM benutzer ORDER BY erstellt_am LIMIT 1), NOW() - INTERVAL '30 days', NOW() - INTERVAL '10 days' FROM e, s;
    `);

	await page.getByTitle('Mahnwesen').click();
	const zeile = page.getByRole('row', { name: new RegExp(`Briefkind-${suffix}`) });
	await expect(zeile).toContainText('noch nicht gemahnt');
	await expect(zeile).toContainText('10 Tage überfällig');

	// Ohne Auswahl: das Druck-Menü hinter dem Drucker-Knopf, kein Knopf „Mahnbriefe".
	await expect(
		page.getByRole('button', { name: 'Weitere Druck- und Export-Optionen' })
	).toBeVisible();
	await expect(page.getByRole('button', { name: 'Mahnbriefe', exact: true })).toHaveCount(0);

	await page.getByLabel(`E2E Briefkind-${suffix} auswählen`).click();
	const download = page.waitForEvent('download');
	await page.getByRole('button', { name: 'Mahnbriefe drucken' }).click();
	const datei = await download;
	expect(datei.suggestedFilename()).toMatch(/^mahnbriefe_\d{4}-\d{2}-\d{2}\.pdf$/);
	const pfad = await datei.path();
	expect(readFileSync(pfad).subarray(0, 4).toString(), 'der Druck ist ein PDF').toBe('%PDF');

	// Die Liste lädt nach dem Druck neu und nennt die Mahnung mit dem heutigen Tag.
	const heute = new Date().toLocaleDateString('de-DE', {
		day: '2-digit',
		month: '2-digit',
		year: 'numeric',
		timeZone: 'Europe/Berlin'
	});
	await expect(zeile).toContainText(`1× gemahnt, zuletzt ${heute}`);
});
