import { test, expect } from '@playwright/test';
import { uiLogin, gehZu, seedSQL, querySQL, uniqueSuffix, seedFehlbestand } from './helpers.js';

// Die Nachbestell-Liste zählt am Buch: Zwei Auflagen desselben Buchs stehen als EINE Zeile, ihr
// Bestand zählt zusammen, gezeigt wird die neueste Auflage (die wird bestellt), darunter die
// Aufschlüsselung. Reicht der Bestand beider zusammen, verschwindet die Zeile, auch wenn die
// neueste Auflage allein zu wenig hat.
test('Bestellbedarf: Auflagen eines Buchs stehen als eine Zeile mit ihrer Summe', async ({
	page
}) => {
	const marke = `E2E-Buch-${uniqueSuffix()}`;
	const einstellung = (/** @type {string} */ schluessel) =>
		querySQL(`SELECT wert FROM system_einstellungen WHERE schluessel = '${schluessel}'`);
	const vorher = { bestellbedarf_warnung_aktiv: einstellung('bestellbedarf_warnung_aktiv') };
	const setze = (/** @type {string} */ schluessel, /** @type {string} */ wert) =>
		seedSQL(`INSERT INTO system_einstellungen (schluessel, wert) VALUES ('${schluessel}', '${wert}')
		         ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert;`);

	const werk = querySQL(`INSERT INTO werke DEFAULT VALUES RETURNING id`).split('\n')[0];
	const titel = (
		/** @type {string} */ auflage,
		/** @type {number} */ jahr,
		/** @type {string} */ isbn
	) =>
		querySQL(
			`INSERT INTO buecher_titel (titel, auflage, erscheinungsjahr, ist_lernmittel, werk_id, isbn)
			 VALUES ('${marke}', '${auflage}', ${jahr}, true, '${werk}', '${isbn}') RETURNING id`
		).split('\n')[0];
	const ziffern = String(Date.now()).slice(-8);
	const isbnAlt = `97830${ziffern}`;
	const isbnNeu = `97831${ziffern}`;
	const alt = titel('3. Aufl.', 2019, isbnAlt);
	const neu = titel('4. Aufl.', 2023, isbnNeu);
	// Ein Anker mit Fehlbestand hält die Liste gefüllt: Ohne jeden Bedarf zeigt die Seite den
	// Leerzustand statt des Filterfelds, und der letzte Schritt fände nichts zum Tippen.
	const anker = querySQL(
		`INSERT INTO buecher_titel (titel, ist_lernmittel) VALUES ('E2E-Anker-${uniqueSuffix()}', true) RETURNING id`
	).split('\n')[0];
	// Alt 2, neu 1 Exemplar, der Anker 1: Das Buch hat zusammen 3.
	const exemplare = querySQL(`INSERT INTO buecher_exemplare (titel_id, barcode_id)
		VALUES ('${alt}', '${marke}-1'), ('${alt}', '${marke}-2'), ('${neu}', '${marke}-3'), ('${anker}', '${marke}-4')
		RETURNING id`).split('\n');
	const wegFehlbestand = seedFehlbestand(marke, exemplare, 3);

	try {
		setze('bestellbedarf_warnung_aktiv', 'true');

		await uiLogin(page);
		await gehZu(page, '/bestellungen');
		const filter = page.getByRole('searchbox', { name: 'Bestellvorschläge filtern' });
		await filter.fill(marke);
		const zeilen = page.locator('div.group').filter({ hasText: marke });
		await expect(zeilen).toHaveCount(1);
		await expect(zeilen.first()).toContainText(isbnNeu);
		await expect(zeilen.first()).toContainText(
			'Bestand aus 2 Auflagen: 4. Aufl. · 2023 (1), 3. Aufl. · 2019 (2)'
		);

		// Wer die alte Auflage scannt, findet dieselbe Zeile.
		await filter.fill(isbnAlt);
		await expect(zeilen).toHaveCount(1);

		// Die alte Auflage bekommt so viele Exemplare, wie Kinder nachrücken: Zusammen reicht der
		// Bestand, obwohl die neueste Auflage allein weiter nur eines hat.
		const nachruecken = querySQL(`SELECT count(*) FROM schueler WHERE deleted_at IS NULL
			AND NOT ist_abgaenger AND substring(klasse from '^0*([1-9]|1[0-3])(?:\\D|$)') = '2'`);
		seedSQL(`INSERT INTO buecher_exemplare (titel_id, barcode_id)
		         SELECT '${alt}', '${marke}-mehr-' || g FROM generate_series(1, ${nachruecken}) g;`);
		await page.reload();
		await page.getByRole('searchbox', { name: 'Bestellvorschläge filtern' }).fill(marke);
		await expect(page.getByText('Kein Treffer für')).toBeVisible();
		await expect(zeilen).toHaveCount(0);
	} finally {
		wegFehlbestand();
		seedSQL(`
			DELETE FROM buecher_titel WHERE id IN ('${alt}', '${neu}', '${anker}');
			DELETE FROM werke WHERE id = '${werk}';
		`);
		for (const [schluessel, wert] of Object.entries(vorher)) {
			if (wert) setze(schluessel, wert);
			else seedSQL(`DELETE FROM system_einstellungen WHERE schluessel = '${schluessel}';`);
		}
	}
});
