import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix, gehZu } from './helpers.js';

// Die zweite Hälfte von Punkt 1 des Protokolls vom 16.09.2026. Gemessen wird am Draht, was
// die Bibliothekskraft sieht: Zugänge des laufenden Halbjahres, getrennt nach Topf — und der
// Topf kommt aus der BESTELLUNG, nicht aus dem Titel.
test('Zugangsbuch: Zugänge des Halbjahres nach Topf, Lieferant dabei — und als PDF', async ({
	page
}) => {
	const s = uniqueSuffix();
	// Ein Tag, an dem kein anderer Test Exemplare anlegt: Die Seite zeichnet je Topf zuerst 50
	// Zeilen, und die Zugänge anderer Tests von heute schöben die eigenen dahinter.
	const tag = '2003-04-07';

	// Anlegen und Verknüpfen in getrennten Anweisungen: Ein UPDATE, das die Zeilen einer
	// schreibenden CTE derselben Anweisung sucht, trifft lautlos 0 Zeilen.
	seedSQL(`
		INSERT INTO buecher_titel (titel, signatur) VALUES ('E2E-Zugang-Titel ${s}', 'Zug ${s}');

		INSERT INTO bestellungen_verlauf (lieferant_name, lieferant_email, mittel)
		VALUES ('E2E-Haendler ${s}', 'haendler-${s}@example.invalid', 'land');

		INSERT INTO buecher_exemplare (titel_id, barcode_id, erworben_am, bestellung_id)
		SELECT t.id, 'E2E-ZUG-A-${s}', DATE '${tag}', b.id
		FROM buecher_titel t, bestellungen_verlauf b
		WHERE t.titel = 'E2E-Zugang-Titel ${s}' AND b.lieferant_name = 'E2E-Haendler ${s}';

		INSERT INTO buecher_exemplare (titel_id, barcode_id, erworben_am)
		SELECT t.id, 'E2E-ZUG-B-${s}', DATE '${tag}'
		FROM buecher_titel t WHERE t.titel = 'E2E-Zugang-Titel ${s}';
	`);

	try {
		await uiLogin(page);
		// Der Menüpunkt steht unter „Berichte → Bestandsbücher".
		await gehZu(page, '/bestandsbuecher');
		await page.getByRole('tab', { name: 'Zugangsbuch' }).click();

		// 0. Ohne Datum steht das vorige Schulhalbjahr da: Gedruckt wird zu Beginn eines
		//    Halbjahres für das vorige (Leitfaden Lernmittelfreiheit, 11.5). Es endet vor heute;
		//    die angelegten Zugänge zeigt erst der gewählte Tag.
		const heute = querySQL(`SELECT to_char(CURRENT_DATE, 'YYYY-MM-DD')`);
		const vonFeld = page.getByLabel('Von', { exact: true });
		const bisFeld = page.getByLabel('Bis', { exact: true });
		await expect(bisFeld).not.toHaveValue('');
		expect((await bisFeld.inputValue()) < heute, 'das vorige Halbjahr endet vor heute').toBe(true);
		await vonFeld.fill(tag);
		await bisFeld.fill(tag);
		await page.getByRole('button', { name: 'Anzeigen' }).click();

		// 1. Der Zugang aus der Bestellung steht unter dem Topf des Landes, mit Lieferant.
		const land = page.getByRole('table', { name: /Zugangsbuch — Lernmittelfreiheit \(Land\)/ });
		await expect(land.getByText(`E2E-ZUG-A-${s}`)).toBeVisible();
		await expect(land.getByText(`E2E-Haendler ${s}`)).toBeVisible();

		// 2. Der Zugang OHNE Bestellung steht nicht dort, sondern unter „ohne Zuordnung" —
		//    geraten wird der Topf nicht.
		await expect(land.getByText(`E2E-ZUG-B-${s}`)).toHaveCount(0);
		const ohne = page.getByRole('table', { name: /Zugangsbuch — ohne Zuordnung/ });
		await expect(ohne.getByText(`E2E-ZUG-B-${s}`)).toBeVisible();

		// 3. Und die Einschränkung steht dabei, statt Vollständigkeit zu behaupten.
		await expect(page.getByText(/keine Bestellung hinterlegt/)).toBeVisible();

		// 4. Der Ausdruck trägt den Zeitraum, den der Bildschirm zeigt.
		const druck = page.getByRole('link', { name: 'Ausdrucken' });
		const adresse = await druck.getAttribute('href');
		expect(adresse).toMatch(
			/\/api\/bestand\/zugangsbuch\/pdf\?von=\d{4}-\d{2}-\d{2}&bis=\d{4}-\d{2}-\d{2}$/
		);

		// 5. Die Töpfe stehen als Felder mit ihrer Zahl unter dem Zeitraum. Ein Klick zeigt einen
		//    Topf allein, und der Hinweis zu „ohne Zuordnung" geht mit seiner Liste. Der Ausdruck
		//    bleibt das ganze Buch: Die Wahl ändert seine Adresse nicht.
		const feldLand = page
			.getByRole('group', { name: 'Nach Mittelherkunft filtern' })
			.getByRole('button', { name: /^Lernmittelfreiheit \(Land\) · \d/ });
		await feldLand.click();
		await expect(feldLand).toHaveAttribute('aria-pressed', 'true');
		await expect(land.getByText(`E2E-ZUG-A-${s}`)).toBeVisible();
		await expect(ohne).toHaveCount(0);
		await expect(page.getByText(/keine Bestellung hinterlegt/)).toHaveCount(0);
		await expect(druck).toHaveAttribute('href', adresse ?? '');

		// Der zweite Klick zeigt wieder alle.
		await feldLand.click();
		await expect(ohne.getByText(`E2E-ZUG-B-${s}`)).toBeVisible();

		// 6. Der Ausdruck antwortet mit einem PDF.
		const antwort = await page.request.get(adresse ?? '');
		expect(antwort.status()).toBe(200);
		expect(antwort.headers()['content-type']).toContain('application/pdf');
		expect((await antwort.body()).subarray(0, 4).toString()).toBe('%PDF');
	} finally {
		// Ohne das stünden Titel, Bestellung und zwei Exemplare nach jedem Lauf im
		// Zugangsbuch und verschöben jede lokale Zählung. Die Exemplare fallen mit dem
		// Titel (ON DELETE CASCADE).
		seedSQL(`
			DELETE FROM bestellungen_verlauf WHERE lieferant_name = 'E2E-Haendler ${s}';
			DELETE FROM buecher_titel WHERE titel = 'E2E-Zugang-Titel ${s}';
		`);
	}
});

// Ein Zeitraum kann Zehntausende Zugänge haben, etwa am Tag eines Listenimports, und sie alle zu
// zeichnen hält die Seite Sekunden an. Ein Topf zeigt deshalb zuerst 50 Zeilen; die Zahl in
// Feld und Überschrift nennt alle.
test('Zugangsbuch: Ein Topf mit mehr als 50 Zugängen zeigt 50 Zeilen und „Mehr laden"', async ({
	page
}) => {
	const s = uniqueSuffix();
	const tag = '2003-04-08';
	seedSQL(`
		INSERT INTO buecher_titel (titel, signatur) VALUES ('E2E-Zugang-Viele ${s}', 'Zug ${s}');

		INSERT INTO buecher_exemplare (titel_id, barcode_id, erworben_am)
		SELECT t.id, 'E2E-ZUG-V-${s}-' || lpad(n::text, 2, '0'), DATE '${tag}'
		FROM buecher_titel t, generate_series(1, 55) n
		WHERE t.titel = 'E2E-Zugang-Viele ${s}';
	`);

	try {
		await uiLogin(page);
		await gehZu(page, '/bestandsbuecher');

		// Erst wenn der Vorschlag des Servers in den Feldern steht, überschreibt er die Eingabe
		// nicht mehr.
		const bisFeld = page.getByLabel('Bis', { exact: true });
		await expect(bisFeld).not.toHaveValue('');
		await page.getByLabel('Von', { exact: true }).fill(tag);
		await bisFeld.fill(tag);
		await page.getByRole('button', { name: 'Anzeigen' }).click();

		const felder = page.getByRole('group', { name: 'Nach Mittelherkunft filtern' });
		await expect(felder.getByRole('button', { name: 'ohne Zuordnung · 55' })).toBeVisible();
		await expect(
			page.getByRole('heading', { name: 'ohne Zuordnung · 55 Exemplare' })
		).toBeVisible();
		const ohne = page.getByRole('table', { name: 'Zugangsbuch — ohne Zuordnung' });
		await expect(ohne.locator('tbody tr')).toHaveCount(50);

		await page.getByRole('button', { name: 'Mehr laden (5 weitere)' }).click();
		await expect(ohne.locator('tbody tr')).toHaveCount(55);
		await expect(page.getByRole('button', { name: /^Mehr laden/ })).toHaveCount(0);
	} finally {
		seedSQL(`DELETE FROM buecher_titel WHERE titel = 'E2E-Zugang-Viele ${s}';`);
	}
});
