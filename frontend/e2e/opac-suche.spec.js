import { test, expect } from '@playwright/test';
import { seedSQL, uiLogin, uniqueSuffix } from './helpers.js';

// Öffentlicher Medienkatalog (/katalog): die einzige komplett anonyme Route.
// Muss OHNE Login erreichbar sein und darf keine Ausleiher-Daten leaken (DSGVO).

test('OPAC: öffentliche Suche ohne Login zeigt Verfügbarkeit, leakt keine Personendaten', async ({
	page
}) => {
	const s = uniqueSuffix();

	seedSQL(`
        WITH t AS (
            INSERT INTO buecher_titel (isbn, titel, autor)
            VALUES ('979-${s}', 'E2E Opactitel ${s}', 'Opac Autor')
            RETURNING id
        )
        INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
        SELECT id, 'OPAC-${s}', true FROM t;
    `);

	// Direkt und anonym — kein uiLogin!
	await page.goto('/katalog');
	await expect(page.getByText('Öffentlicher Medienkatalog')).toBeVisible();
	await expect(page.getByText('DSGVO-konform')).toBeVisible();

	// Suche (debounced) findet den geseedeten Titel mit Verfügbarkeits-Badge
	await page.getByPlaceholder('Titel, Autor oder ISBN eingeben …').fill(`E2E Opactitel ${s}`);
	await expect(page.getByText(`E2E Opactitel ${s}`).first()).toBeVisible();
	await expect(page.getByText('✓ Verfügbar').first()).toBeVisible();

	// DSGVO-Check auf API-Ebene: die öffentliche Antwort enthält keine
	// Ausleiher-/Personenfelder.
	const suchbegriff = encodeURIComponent(`E2E Opactitel ${s}`);
	const res = await page.request.get(`/api/public/opac/suche?q=${suchbegriff}`);
	expect(res.status()).toBe(200);
	const body = JSON.stringify(await res.json()).toLowerCase();
	for (const feld of ['vorname', 'nachname', 'schueler', 'klasse', 'barcode_id', 'eltern_email']) {
		expect(body, `öffentliche OPAC-Antwort enthält "${feld}"`).not.toContain(feld);
	}
});

// Schulbücher der Lernmittelfreiheit werden klassensatzweise zugeteilt, nicht
// recherchiert. Im öffentlichen Katalog würden sie die Treffer der Freihand-
// Bibliothek zuschütten — bei Klassensatzstärke sind das hunderte Titel.
test('OPAC: LMF-Schulbücher bleiben aus dem öffentlichen Katalog heraus', async ({ page }) => {
	const s = uniqueSuffix();

	// Lernmittel ist seit Migration 093 ein Feld — der Titeltext ist beliebig. Die
	// alten „LMF"-Schreibweisen bleiben als Titel stehen, damit der Test weiter
	// dokumentiert, was 2026 zweimal durch die Text-Erkennung fiel.
	seedSQL(`
        WITH t AS (
            INSERT INTO buecher_titel (titel, autor, ist_lernmittel)
            VALUES ('LMF-Biologie ${s}', 'Schulbuchverlag', true),
                   ('LMF - Deutsch ${s}', 'Schulbuchverlag', true),
                   ('Freihand Roman ${s}', 'Romanautor', false)
            RETURNING id, titel
        )
        INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
        SELECT id, 'LMFOPAC-' || substr(md5(titel), 1, 8) || '-${s}', true FROM t;
    `);

	// Gesucht wird nach dem gemeinsamen Suffix: Ohne Filter kämen alle drei zurück.
	const res = await page.request.get(`/api/public/opac/suche?q=${encodeURIComponent(s)}`);
	expect(res.status()).toBe(200);
	const titel = (await res.json()).map((/** @type {any} */ t) => t.titel);

	expect(titel, 'Freihand-Titel muss im Katalog stehen').toContain(`Freihand Roman ${s}`);
	expect(titel, 'LMF-Titel mit Bindestrich').not.toContain(`LMF-Biologie ${s}`);
	expect(titel, 'LMF-Titel mit Leerzeichen').not.toContain(`LMF - Deutsch ${s}`);

	// Und derselbe Titel bleibt für die Verwaltung auffindbar — der Filter gehört
	// ausschließlich in die öffentliche Suche.
	await uiLogin(page);
	const intern = await page.request.get(`/api/books?q=${encodeURIComponent('LMF-Biologie ' + s)}`);
	expect(intern.status(), 'interne Buchsuche').toBe(200);
	expect(JSON.stringify(await intern.json()), 'LMF-Titel muss intern auffindbar bleiben').toContain(
		`LMF-Biologie ${s}`
	);
});
