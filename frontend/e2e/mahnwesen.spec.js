import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// Mahnwesen: überfällige Ausleihe erscheint in der Übersicht, und „Liste drucken" druckt
// die Liste als Tabelle mit dem Buch.
test('Mahnwesen: überfälliger Schüler erscheint, „Liste drucken" druckt ihn mit Buch', async ({
	page
}) => {
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

	// Das Blatt trägt, was die Liste zeigt: nach der Suche genau dieses Kind, je Buch eine Zeile.
	await page
		.getByRole('searchbox', { name: 'Schüler oder Klasse suchen' })
		.fill(`Saeumig-${suffix}`);
	const fenster = page.waitForEvent('popup');
	await page.getByRole('button', { name: 'Liste drucken' }).click();
	const blatt = await fenster;
	await expect(blatt.locator('h1')).toHaveText('Mahnliste');
	const druckzeile = blatt.locator('tbody tr');
	await expect(druckzeile).toHaveCount(1);
	await expect(druckzeile).toContainText(`E2E Saeumig-${suffix}`);
	await expect(druckzeile).toContainText('8M');
	await expect(druckzeile).toContainText(`E2E-Mahnbuch-${suffix}`);
	await expect(druckzeile).toContainText('noch nicht gemahnt');
	await expect(blatt.locator('p.meta')).toContainText('1 Kind, 1 Buch');
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

	// Ohne Auswahl: „Liste drucken", kein Knopf „Mahnbriefe".
	await expect(page.getByRole('button', { name: 'Liste drucken' })).toBeVisible();
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
