import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// Die Suche unter „Suche & Filter" im Medienkatalog findet einen Titel über sein Schlagwort,
// auch angefangen, und über einen Verweis auf das Schlagwort. Gefiltert wird im Browser an den
// Suchwörtern, die /api/books je Titel mitliefert (repository.ladeSuchwoerterDerTitel).
test.describe('Medienkatalog: Suche über Schlagworte', () => {
	const s = uniqueSuffix().slice(0, 6);
	const WORT = `Drachenkunde${s}`;
	const VERWEIS = `Lindwurmlehre${s}`;
	const MIT = `Das Feuerbuch ${s}`;
	const OHNE = `Das Wasserbuch ${s}`;

	test.beforeAll(() => {
		seedSQL(`
			INSERT INTO buecher_titel (isbn, titel, autor) VALUES
				('978sm${s}', '${MIT}', 'Katalog Autor'),
				('978so${s}', '${OHNE}', 'Katalog Autor');
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
			SELECT id, 'SW-' || isbn, true FROM buecher_titel WHERE isbn IN ('978sm${s}', '978so${s}');
			INSERT INTO schlagworte (wort) VALUES ('${WORT}');
			INSERT INTO schlagworte (wort, verweis_auf) SELECT '${VERWEIS}', id FROM schlagworte WHERE wort = '${WORT}';
			INSERT INTO titel_schlagworte (titel_id, schlagwort_id)
			SELECT t.id, w.id FROM buecher_titel t, schlagworte w
			WHERE t.isbn = '978sm${s}' AND w.wort = '${WORT}';
		`);
	});

	test.afterAll(() => {
		seedSQL(`
			DELETE FROM buecher_titel WHERE isbn IN ('978sm${s}', '978so${s}');
			DELETE FROM schlagworte WHERE wort = '${VERWEIS}';
			DELETE FROM schlagworte WHERE wort = '${WORT}';
		`);
	});

	test('findet den Titel über das Schlagwort, ein Stück davon und den Verweis', async ({
		page
	}) => {
		await uiLogin(page);
		await page.goto('/medienkatalog');
		await page.getByRole('tab', { name: 'Suche & Filter' }).click();
		const suche = page.getByRole('searchbox', {
			name: 'Suchen nach Titel, Fach, Klasse oder Autor'
		});

		for (const begriff of [WORT, `kunde${s}`, VERWEIS]) {
			await suche.fill(begriff);
			await expect(page.getByText(MIT).first(), `Suche nach „${begriff}"`).toBeVisible({
				timeout: 15000
			});
			await expect(page.getByText(OHNE), `Suche nach „${begriff}"`).toHaveCount(0);
		}
	});
});
