import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// Punkt 4 des Protokolls vom 16.09.2026: „Bücher, zu denen es keine Exemplare gibt,
// tauchen in der Trefferliste auf."
//
// Am 17.09.2026 war entschieden, sie NICHT zu verstecken, sondern den Bestand zu sagen. Die
// Schule hat am 22.09.2026 anders entschieden (docs/OFFEN.md 9.4): Ein Titel ohne Exemplar
// steht in keinem Katalog und in keiner Trefferliste — die Verwaltung erreicht ihn über die
// Aufräumsicht der Titel-Verwaltung. Der Titel mit Exemplaren sagt weiter seinen Bestand:
// „0 von 2 verfügbar" ist im Schuljahr der Normalfall und darf nicht wie eine Sackgasse
// aussehen.
//
// Rot gesehen am 22.09.2026 an der alten Erwartung (der Titel ohne Exemplar stand in der
// Liste) — dieselbe Spec, andere Erwartung.
test('Theke: die Trefferliste sagt den Bestand — und zeigt keinen Titel ohne Exemplare', async ({
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

	// 1. Der Titel mit Exemplaren steht in der Liste und nennt die Zahlen: zwei Exemplare,
	//    eines verliehen.
	await expect(mit).toBeVisible();
	await expect(mit).toContainText('1 von 2 verfügbar');
	await expect(mit).not.toContainText('Keine Exemplare');

	// 2. Der Titel ohne Exemplar steht nicht darin — dieselbe Suche, dieselbe Liste. Erst
	//    geprüft, nachdem die Liste da ist (Schritt 1), sonst zählte eine leere Liste als
	//    Beweis.
	await expect(ohne).toHaveCount(0);
});
