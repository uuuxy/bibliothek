import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// Punkt 4 des Protokolls vom 16.09.2026: „Bücher, zu denen es keine Exemplare gibt,
// tauchen in der Trefferliste auf."
//
// Entschieden ist, sie NICHT zu verstecken — ein Titel ohne Exemplare ist ein legitimer
// Zustand (angelegt ohne Bestandsangabe, Altbestand aus Littera). Die Liste muss es
// SAGEN, sonst läuft jemand ins Regal und sucht etwas, das es dort nie gab.
//
// Der Test misst beide Fälle an EINER Trefferliste, weil der Unterschied der Punkt ist:
// „Keine Exemplare" ist eine Sackgasse, „0 von 2 verfügbar" ist der Normalfall im
// Schuljahr. Sähen beide gleich aus, wäre nichts gewonnen.
test('Theke: die Trefferliste sagt den Bestand — und nennt einen Titel ohne Exemplare', async ({
	page
}) => {
	const s = uniqueSuffix();

	seedSQL(`
		INSERT INTO buecher_titel (titel) VALUES ('E2E-Bestand-Ohne ${s}');

		WITH t AS (INSERT INTO buecher_titel (titel) VALUES ('E2E-Bestand-Mit ${s}') RETURNING id),
		ex AS (INSERT INTO buecher_exemplare (titel_id, barcode_id)
			SELECT id, 'E2E-BST-A-${s}' FROM t
			UNION ALL SELECT id, 'E2E-BST-B-${s}' FROM t
			RETURNING id, barcode_id),
		sch AS (INSERT INTO leser (barcode_id, vorname, nachname, klasse, art, abgaenger_jahr)
			VALUES ('E2E-BST-S-${s}', 'Bestand${s}', 'Testleser', '7a', 'schueler',
				EXTRACT(YEAR FROM CURRENT_DATE)::int + 1) RETURNING id)
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		SELECT ex.id, sch.id, CURRENT_DATE + 14 FROM ex, sch WHERE ex.barcode_id = 'E2E-BST-A-${s}';
	`);

	await uiLogin(page);

	// Blind tippen statt fill(): Der Kiosk lebt vom Tastaturfokus (siehe
	// kiosk-scannerfokus.spec.js) — fill() setzte ihn implizit und verdeckte einen Fehler.
	await page.locator('#omnibox-input').click();
	await page.keyboard.type(`E2E-Bestand ${s}`);

	const liste = page.locator('#omnibox-dropdown');
	const ohne = liste.getByRole('option', { name: new RegExp(`E2E-Bestand-Ohne ${s}`) });
	const mit = liste.getByRole('option', { name: new RegExp(`E2E-Bestand-Mit ${s}`) });

	// 1. Beide Titel stehen in der Liste — der ohne Exemplare wird nicht versteckt.
	await expect(ohne).toBeVisible();
	await expect(mit).toBeVisible();

	// 2. Der Titel ohne Exemplare sagt es. Bis zum 17.09.2026 stand hier nur der Name.
	await expect(ohne).toContainText('Keine Exemplare');

	// 3. Der andere nennt die Zahlen: zwei Exemplare, eines verliehen.
	await expect(mit).toContainText('1 von 2 verfügbar');
	await expect(mit).not.toContainText('Keine Exemplare');

	// 4. Und der Screenreader hört dasselbe — die Zeile trägt es im Namen, nicht nur
	//    als Farbe (BITV: eine Auszeichnung allein über die Farbe zählt nicht).
	await expect(ohne).toHaveAttribute('aria-label', /Keine Exemplare/);
});
