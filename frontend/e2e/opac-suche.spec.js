import { test, expect } from '@playwright/test';
import { seedSQL, uniqueSuffix } from './helpers.js';

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
	const res = await page.request.get(
		`/api/public/opac/suche?q=${encodeURIComponent(`E2E Opactitel ${s}`)}`
	);
	expect(res.status()).toBe(200);
	const body = JSON.stringify(await res.json()).toLowerCase();
	for (const feld of ['vorname', 'nachname', 'schueler', 'klasse', 'barcode_id', 'eltern_email']) {
		expect(body, `öffentliche OPAC-Antwort enthält "${feld}"`).not.toContain(feld);
	}
});
