import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, seedBenutzer, uniqueSuffix } from './helpers.js';

test('Klassensatz-Reservierung "erledigen"', async ({ page }) => {
	// 1. Seed a book title and a reservation
	const s = uniqueSuffix();
	seedSQL(`
        INSERT INTO buecher_titel (id, isbn, titel, autor)
        VALUES (gen_random_uuid(), '978-${s}', 'E2E Klassensatz Buch ${s}', 'Test Autor');

        INSERT INTO klassensatz_reservierungen (titel_id, klasse, anzahl, notiz, angefordert_von)
        VALUES ((SELECT id FROM buecher_titel WHERE isbn = '978-${s}'), 'k8b${s}', 25, 'E2E Test Notiz', NULL);
    `);

	// 2. Login
	await uiLogin(page);

	// 3. Navigation zu Bestellwesen -> Klassensätze
	await page.getByTitle('Bestellungen').click();

	// Über die Rolle „tab", nicht „button": Seit dem 09.08.2026 trägt die Reiterzeile
	// role=tablist/tab (vorher nackte <button>, ein Screenreader hörte sechs
	// zusammenhanglose Knöpfe). Das löst zugleich die Mehrdeutigkeit, die der Kommentar
	// hier vorher beschrieb — der Badge in der Seitenleiste ist kein Reiter.
	await page.getByRole('tab', { name: /Klassensatz-Reservierungen/i }).click();

	// 4. Verifikation des Renderns der Reservierung
	await expect(page.getByText(`E2E Klassensatz Buch ${s}`).first()).toBeVisible();
	await expect(page.getByText(`k8b${s}`).first()).toBeVisible();
	await expect(page.getByText('25').first()).toBeVisible();

	// 5. Reservierung abschließen
	const reservierungZeile = page
		.locator('li')
		.filter({ hasText: `E2E Klassensatz Buch ${s}` })
		.first();
	await reservierungZeile.getByRole('button', { name: 'Abschließen' }).click();

	// Bestätigung klicken
	await reservierungZeile.getByRole('button', { name: 'Wirklich abschließen?' }).click();

	// 6. Verifikation: Die Reservierung verschwindet aus der UI
	await expect(reservierungZeile).not.toBeVisible();
});

// Das Warteschlangen-Modell (16.08.2026) am ganzen Weg: Reservieren sperrt nichts —
// die zweite Lehrkraft sieht VOR dem Klick, wer schon in der Schlange steht, und
// erfährt nach dem Absenden, hinter wem ihr Satz an der Reihe ist.
test('Klassensatz-Warteschlange: Chip vor dem Klick, Vordermann nach dem Absenden', async ({
	page
}) => {
	const s = uniqueSuffix();
	// Ein Titel mit drei Exemplaren und einer bestehenden Reservierung der 08a.
	seedSQL(`
        INSERT INTO buecher_titel (id, isbn, titel, autor)
        VALUES (gen_random_uuid(), '979-${s}', 'E2E KSQ Buch ${s}', 'Queue Autor');

        INSERT INTO buecher_exemplare (titel_id, barcode_id)
        SELECT id, 'B-KSQ-' || '${s}' || '-' || n
        FROM buecher_titel, generate_series(1, 3) AS n WHERE isbn = '979-${s}';

        INSERT INTO klassensatz_reservierungen (titel_id, klasse, anzahl, erstellt_am)
        VALUES ((SELECT id FROM buecher_titel WHERE isbn = '979-${s}'), 'k8a${s}', 3, now() - interval '1 hour');
    `);

	const LEHRKRAFT = `e2e-ksq-${s}@test.local`;
	seedBenutzer(LEHRKRAFT, 'kollegium');
	await uiLogin(page, LEHRKRAFT);
	await page.getByTitle('Mein Portal').click();

	await page
		.getByRole('textbox', { name: 'Bücher für einen Klassensatz suchen' })
		.fill(`E2E KSQ Buch ${s}`);

	// Der Chip steht am Treffer, BEVOR irgendetwas angeklickt wird.
	await expect(page.getByText(`3 reserviert für k8a${s}`)).toBeVisible();
	// Verrechnet mit dem Regal: 3 vorhanden, 3 vorgemerkt → rechnerisch nichts frei.
	await expect(page.getByText('3 vorgemerkt · 0 rechnerisch frei')).toBeVisible();

	// Trotzdem reservieren: erlaubt — die Warnung steht im Formular VOR dem Absenden,
	// und die Bestätigung nennt den Vordermann.
	await page.getByRole('button', { name: 'Klassensatz reservieren' }).click();
	await expect(page.getByRole('status')).toContainText(`du stellst dich hinter k8a${s} an`);
	await page.getByLabel('Klasse *').fill(`k9b${s}`);
	await page.getByRole('button', { name: /Anfrage senden/ }).click();
	await expect(
		page.getByTitle(new RegExp(`dein Satz ist nach k8a${s} an der Reihe`))
	).toBeVisible();

	// Und die eigene Reservierung erscheint sofort als zweiter Chip in der Schlange.
	await expect(page.getByText(`reserviert für k9b${s}`)).toBeVisible();
});

// Die Bibliotheksseite derselben Schlange: älteste zuerst, und je Zeile der
// Regal-Blick — mit einem verliehenen Exemplar zeigt der Titel "2 verfügbar".
test('Klassensatz-Warteschlange: Reihenfolge und Regal-Blick im Erledigen-Tab', async ({
	page
}) => {
	const s = uniqueSuffix();
	seedSQL(`
        INSERT INTO buecher_titel (id, isbn, titel, autor)
        VALUES (gen_random_uuid(), '977-${s}', 'E2E KSQ Tab ${s}', 'Queue Autor');

        INSERT INTO buecher_exemplare (titel_id, barcode_id)
        SELECT id, 'B-KST-' || '${s}' || '-' || n
        FROM buecher_titel, generate_series(1, 3) AS n WHERE isbn = '977-${s}';

        INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr)
        VALUES ('E2E', 'KSQ-Leser', 'k8a${s}', 'S-KSQ-${s}', EXTRACT(YEAR FROM NOW())::int + 2);

        INSERT INTO ausleihen (exemplar_id, schueler_id, bearbeiter_id, ausgeliehen_am, rueckgabe_frist)
        SELECT e.id, (SELECT id FROM schueler WHERE barcode_id = 'S-KSQ-${s}'),
               (SELECT id FROM benutzer ORDER BY erstellt_am LIMIT 1), NOW(), NOW() + INTERVAL '14 days'
        FROM buecher_exemplare e JOIN buecher_titel t ON e.titel_id = t.id
        WHERE t.isbn = '977-${s}' LIMIT 1;

        INSERT INTO klassensatz_reservierungen (titel_id, klasse, anzahl, erstellt_am)
        VALUES ((SELECT id FROM buecher_titel WHERE isbn = '977-${s}'), 'k8a${s}', 3, now() - interval '2 hours'),
               ((SELECT id FROM buecher_titel WHERE isbn = '977-${s}'), 'k9b${s}', 2, now() - interval '1 hour');
    `);

	await uiLogin(page);
	await page.getByTitle('Bestellungen').click();
	await page.getByRole('tab', { name: /Klassensatz-Reservierungen/i }).click();

	const zeilen = page.locator('li').filter({ hasText: `E2E KSQ Tab ${s}` });
	// Warteschlange: die ältere Klasse steht VOR der jüngeren. Klassennamen sind
	// suffix-eindeutig: Das Klassen-Vokabular (Migration 079) kanonisiert bekannte
	// Schreibweisen — feste Namen wie '08a' würden je nach DB-Altbestand zu '8A'.
	await expect(zeilen.nth(0)).toContainText(`k8a${s}`);
	await expect(zeilen.nth(1)).toContainText(`k9b${s}`);
	// Regal-Blick: ein Exemplar ist verliehen, zwei stehen im Regal.
	await expect(zeilen.nth(0)).toContainText('2 verfügbar');
});
