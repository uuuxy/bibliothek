import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, seedSQL, uniqueSuffix } from './helpers.js';

// Fund 31.08.2026 nachts (Duc-Bauer-Spur, flasch3): ListStudentsWithStats filterte
// nur deleted_at — ein als Abgänger markierter Schüler stand zusätzlich mitten in
// „Aktive Schüler", als „Gesperrt" ohne erkennbaren Grund. Nach dem Schuljahres-
// wechsel wären das ~100 Karteileichen zwischen den Aktiven (später sogar mit
// anonymisiertem Namen). Der Reiter heißt AKTIVE Schüler; Abgänger haben ihren
// eigenen Reiter samt eigenem Endpunkt (/api/abgaenger).
test('Aktive Schüler zeigt keine Abgänger — die stehen im Archiv-Reiter', async ({ page }) => {
	await uiLogin(page);
	const suffix = uniqueSuffix();

	const res = await apiPost(page, '/api/schueler', {
		geburtsdatum: '2010-05-05',
		vorname: 'E2E',
		nachname: `Weggegangen-${suffix}`,
		klasse: '10A',
		barcode_id: `ABG-${suffix}`
	});
	expect(res.ok()).toBeTruthy();
	const { id } = await res.json();
	// So hinterlässt ihn der Abgänger-Pfad: Flag + Systemsperre + Grund — und ein
	// noch offenes Buch, denn genau die stehen im Abgänger-Reiter (Arbeitsliste
	// „schuldet noch Bücher": JOIN auf offene Ausleihen in queryGraduatesBasic).
	seedSQL(`
        UPDATE schueler SET ist_abgaenger = true, ist_gesperrt = true, block_reason = 'Abgänger' WHERE id = '${id}';
        WITH t AS (
            INSERT INTO buecher_titel (titel) VALUES ('E2E Abgaengerbuch ${suffix}') RETURNING id
        ), ex AS (
            INSERT INTO buecher_exemplare (titel_id, barcode_id) SELECT id, 'B-ABG-${suffix}' FROM t RETURNING id
        )
        INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
        SELECT ex.id, '${id}', NOW() + INTERVAL '10 days' FROM ex;
    `);

	await page.getByTitle('Schülerdatei').click();
	const suche = page.getByPlaceholder(/Name, Klasse oder Barcode/);
	await suche.fill(`Weggegangen-${suffix}`);

	// Der Leerzustand ist die Aussage: kein Treffer unter den Aktiven. Auf ihn
	// warten (statt sofort zu prüfen), damit die Suche sicher geantwortet hat.
	await expect(page.getByText('Keine Schüler im Verzeichnis gefunden.')).toBeVisible();
	await expect(page.locator('tr').filter({ hasText: `Weggegangen-${suffix}` })).toHaveCount(0);

	// Gegenprobe, damit der Filter nicht zu viel löscht: Im Abgänger-Reiter steht er.
	await page.getByRole('tab', { name: /Abgänger/ }).click();
	await expect(page.getByText(`Weggegangen-${suffix}`).first()).toBeVisible();
});
