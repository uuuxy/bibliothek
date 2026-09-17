import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix, gehZu } from './helpers.js';

// Die zweite Hälfte von Punkt 1 des Protokolls vom 16.09.2026. Gemessen wird am Draht, was
// die Bibliothekskraft sieht: Zugänge des laufenden Halbjahres, getrennt nach Topf — und der
// Topf kommt aus der BESTELLUNG, nicht aus dem Titel.
test('Zugangsbuch: Zugänge des Halbjahres nach Topf, Lieferant dabei — und als PDF', async ({
	page
}) => {
	const s = uniqueSuffix();

	// Anlegen und Verknüpfen in getrennten Anweisungen: Ein UPDATE, das die Zeilen einer
	// schreibenden CTE derselben Anweisung sucht, trifft lautlos 0 Zeilen.
	seedSQL(`
		INSERT INTO buecher_titel (titel, signatur) VALUES ('E2E-Zugang-Titel ${s}', 'Zug ${s}');

		INSERT INTO bestellungen_verlauf (lieferant_name, lieferant_email, mittel)
		VALUES ('E2E-Haendler ${s}', 'haendler-${s}@example.invalid', 'land');

		INSERT INTO buecher_exemplare (titel_id, barcode_id, erworben_am, bestellung_id)
		SELECT t.id, 'E2E-ZUG-A-${s}', CURRENT_DATE, b.id
		FROM buecher_titel t, bestellungen_verlauf b
		WHERE t.titel = 'E2E-Zugang-Titel ${s}' AND b.lieferant_name = 'E2E-Haendler ${s}';

		INSERT INTO buecher_exemplare (titel_id, barcode_id, erworben_am)
		SELECT t.id, 'E2E-ZUG-B-${s}', CURRENT_DATE
		FROM buecher_titel t WHERE t.titel = 'E2E-Zugang-Titel ${s}';
	`);

	await uiLogin(page);
	// Seit dem 17.09.2026 unter „System → Bestandsbücher" statt im Medienkatalog.
	await gehZu(page, '/bestandsbuecher');
	await page.getByRole('tab', { name: 'Zugangsbuch' }).click();

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

	// 4. Der Ausdruck antwortet mit einem PDF.
	const adresse = await page.getByRole('link', { name: 'Ausdrucken' }).getAttribute('href');
	expect(adresse).toMatch(
		/\/api\/bestand\/zugangsbuch\/pdf\?von=\d{4}-\d{2}-\d{2}&bis=\d{4}-\d{2}-\d{2}/
	);
	const antwort = await page.request.get(adresse ?? '');
	expect(antwort.status()).toBe(200);
	expect(antwort.headers()['content-type']).toContain('application/pdf');
	expect((await antwort.body()).subarray(0, 4).toString()).toBe('%PDF');
});
