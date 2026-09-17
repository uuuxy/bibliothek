import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix, gehZu } from './helpers.js';

// Punkt 1 des Protokolls vom 16.09.2026: „Zugangs- und Abgangsbuch fehlen."
//
// Das Abgangsbuch ist ein Nachweis zum Abheften — es muss zwei Dinge können: die Abgänge
// des laufenden Halbjahres nach Topf getrennt zeigen, und sich ausdrucken lassen. Beides
// wird hier am Draht gemessen; eine Liste, die nur auf dem Bildschirm stimmt, hilft der
// Schule am Stichtag nicht.
test('Abgangsbuch: Abgänge des Halbjahres, nach Topf getrennt — und als PDF', async ({ page }) => {
	const s = uniqueSuffix();

	// Zwei Abgänge von heute, einer je Topf. Der Trigger aus Migration 128 setzt das
	// Abgangsdatum selbst — hier wird nur ausgesondert, nicht datiert.
	//
	// Anlegen und Aussondern sind ZWEI Anweisungen: Ein UPDATE, das die Zeilen einer
	// schreibenden CTE derselben Anweisung sucht, findet sie nicht (es liest den
	// Schnappschuss von vorher) und trifft lautlos 0 Zeilen — der Test sähe dann eine
	// leere Liste und gäbe dem Code die Schuld.
	seedSQL(`
		WITH tl AS (INSERT INTO buecher_titel (titel, signatur, ist_lernmittel)
			VALUES ('E2E-Abgang-Lernmittel ${s}', 'Mat ${s}', true) RETURNING id),
		tb AS (INSERT INTO buecher_titel (titel, signatur, ist_lernmittel)
			VALUES ('E2E-Abgang-Buecherei ${s}', 'Jug ${s}', false) RETURNING id)
		INSERT INTO buecher_exemplare (titel_id, barcode_id)
		SELECT id, 'E2E-ABG-L-${s}' FROM tl
		UNION ALL SELECT id, 'E2E-ABG-B-${s}' FROM tb;

		UPDATE buecher_exemplare
		SET ist_ausgesondert = true, ist_ausleihbar = false, aussonderung_grund = 'VERLUST'
		WHERE barcode_id IN ('E2E-ABG-L-${s}', 'E2E-ABG-B-${s}');
	`);

	await uiLogin(page);
	await gehZu(page, '/medienkatalog');
	await page.getByRole('tab', { name: 'Abgangsbuch' }).click();

	// 1. Beide Abschnitte stehen da, jeder mit seiner Stückzahl im Kopf.
	// Die Überschrift kommt vom Server (mittelBeschriftung) — hier steht sie deshalb so,
	// wie sie auch auf dem Ausdruck steht.
	const lernmittel = page.getByRole('table', { name: /Lernmittelfreiheit \(Land\)/ });
	const buecherei = page.getByRole('table', { name: /Schülerbücherei/ });
	await expect(lernmittel.getByText(`E2E-Abgang-Lernmittel ${s}`)).toBeVisible();
	await expect(buecherei.getByText(`E2E-Abgang-Buecherei ${s}`)).toBeVisible();

	// 2. Das Lernmittel steht NICHT im Bücherei-Abschnitt. Ohne diese Gegenprobe bestünde
	//    der Test auch dann, wenn beide Abschnitte dieselbe ungefilterte Liste zeigen.
	await expect(buecherei.getByText(`E2E-Abgang-Lernmittel ${s}`)).toHaveCount(0);

	// 3. Der Grund steht im Klartext, nicht als Schlüssel aus der Datenbank.
	await expect(lernmittel.getByText('Verlust').first()).toBeVisible();
	// Regex, nicht Zeichenkette: getByText('VERLUST') vergleicht ohne Rücksicht auf
	// Groß- und Kleinschreibung und träfe auch „Verlust" — die Gegenprobe wäre wertlos.
	await expect(page.getByText(/^VERLUST$/)).toHaveCount(0);

	// 4. Der Ausdruck: Der Knopf zeigt auf den PDF-Weg, und der liefert ein PDF.
	const druck = page.getByRole('link', { name: 'Ausdrucken' });
	const adresse = await druck.getAttribute('href');
	expect(adresse, 'der Knopf trägt keinen Zeitraum').toMatch(
		/\/api\/bestand\/abgangsbuch\/pdf\?von=\d{4}-\d{2}-\d{2}&bis=\d{4}-\d{2}-\d{2}/
	);
	const antwort = await page.request.get(adresse ?? '');
	expect(antwort.status(), 'der Ausdruck antwortet nicht').toBe(200);
	expect(antwort.headers()['content-type']).toContain('application/pdf');
	expect((await antwort.body()).subarray(0, 4).toString()).toBe('%PDF');
});
