import { test, expect } from '@playwright/test';
import { uiLogin, gehZu, querySQL, seedSQL, uniqueSuffix, scanneWieScanner } from './helpers.js';

// Ein Littera-Etikett trägt im Strichcode nicht die aufgedruckte Nummer, sondern einen EAN-13
// (101634 → 1016340039562). Die Theke rechnet ihn zurück; die Suche im Medienkatalog fand das
// Buch weder über den Scan noch über eine lange LMF-Nummer. Jetzt zeigt sie den Titel des
// Exemplars, mit derselben Erkennung wie die Theke (GET /api/buecher/exemplare/erkennen).

/**
 * Der EAN-13 eines Littera-Etiketts zur Mediennummer: rechts auf 8 Stellen mit Nullen, die
 * Bibliotheksnummer 395, die Stellenzahl, die Prüfziffer (internal/service/littera_etikett.go).
 * @param {string} nummer
 */
function litteraEtikett(nummer) {
	const daten = nummer.padEnd(8, '0') + '395' + nummer.length;
	const summe = [...daten].reduce((s, z, i) => s + Number(z) * (i % 2 ? 3 : 1), 0);
	return daten + ((10 - (summe % 10)) % 10);
}

/** @type {string[]} */
const angelegteTitel = [];

test.afterEach(() => {
	for (const id of angelegteTitel.splice(0)) {
		seedSQL(`
			DELETE FROM buecher_exemplare WHERE titel_id = '${id}';
			DELETE FROM buecher_titel WHERE id = '${id}';
		`);
	}
});

/**
 * Ein Titel mit einem Exemplar, auf Wunsch mit einem alten LMF-Etikett; liefert die Kennung des
 * Titels.
 * @param {string} titel
 * @param {string} barcode
 * @param {string} [altesEtikett]
 */
function legeTitelMitExemplarAn(titel, barcode, altesEtikett) {
	const lmf = altesEtikett ? `'${altesEtikett}'` : 'NULL';
	const id = querySQL(`
		WITH t AS (
			INSERT INTO buecher_titel (titel, autor, medientyp) VALUES ('${titel}', 'E2E', 'Buch')
			RETURNING id
		)
		INSERT INTO buecher_exemplare (titel_id, barcode_id, lmf_barcode) SELECT id, '${barcode}', ${lmf} FROM t
		RETURNING titel_id;
	`)
		.split('\n')[0]
		.trim();
	angelegteTitel.push(id);
	return id;
}

test('Medienkatalog: ein gescanntes Littera-Etikett und ein altes LMF-Etikett zeigen ihren Titel', async ({
	page
}) => {
	const marke = uniqueSuffix();
	const nummer = String(9000000 + Math.floor(Math.random() * 999999));
	const altesEtikett = `LMF-978${String(Date.now()).slice(-10)}-001`;
	const kurz = `B-KE${marke}`;
	const litteraTitel = `E2E Etikettscan ${marke}`;
	const altTitel = `E2E LMF-Etikett ${marke}`;
	legeTitelMitExemplarAn(litteraTitel, nummer);
	legeTitelMitExemplarAn(altTitel, kurz, altesEtikett);

	await uiLogin(page);
	await gehZu(page, '/medienkatalog');
	const feld = page.locator('#katalog-suchfeld');
	// Kein Klick ins Feld: Der Scanner tippt dorthin, wo der Fokus steht.
	await expect(feld).toBeFocused();

	await scanneWieScanner(page, litteraEtikett(nummer));
	await expect(page.getByText(`Exemplar ${nummer} gehört zu „${litteraTitel}“.`)).toBeVisible();
	await expect(page.getByRole('button', { name: litteraTitel, exact: true })).toBeVisible();
	await expect(page.getByText('Keine Bücher gefunden')).toHaveCount(0);

	// Das alte Etikett führt zum Exemplar, genannt wird die Nummer, die es jetzt trägt.
	await feld.fill('');
	await scanneWieScanner(page, altesEtikett);
	await expect(page.getByText(`Exemplar ${kurz} gehört zu „${altTitel}“.`)).toBeVisible();
	await expect(page.getByRole('button', { name: altTitel, exact: true })).toBeVisible();
});
