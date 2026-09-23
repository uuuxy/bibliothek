import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// „Mein Portal → Suchen & Reservieren": Die Schlagworte, die die Pflegeseite als Filter
// markiert, stehen als Filter-Chips unter der Suche (docs/OFFEN.md 4.20). Über den Draht
// und mit der Rolle kollegium, weil das Portal über den öffentlichen Katalog sucht und
// Lehrkräfte kein view_books haben — ein Test am Handler sähe weder die Tür noch die Rolle.
const LEHRER_EMAIL = 'e2e-lehrer-filter@test.local';

test.describe('Mein Portal: Filter nach Schlagwort', () => {
	const s = uniqueSuffix().slice(0, 6);
	const WORT = `E2E-Weltraum ${s}`;
	const MIT = `Mondflug ${s}`;
	const OHNE = `Mondkalender ${s}`;

	test.beforeAll(() => {
		seedSQL(`
			INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
			VALUES ('E2E', 'Filter', '${LEHRER_EMAIL}', 'kollegium', true)
			ON CONFLICT (email) DO UPDATE SET aktiv = true;
			INSERT INTO buecher_titel (isbn, titel, autor) VALUES
				('978fm${s}', '${MIT}', 'Portal Autor'),
				('978fo${s}', '${OHNE}', 'Portal Autor');
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
			SELECT id, 'FI-' || isbn, true FROM buecher_titel WHERE isbn IN ('978fm${s}', '978fo${s}');
			INSERT INTO schlagworte (wort, ist_filter) VALUES ('${WORT}', true);
			INSERT INTO titel_schlagworte (titel_id, schlagwort_id)
			SELECT t.id, w.id FROM buecher_titel t, schlagworte w
			WHERE t.isbn = '978fm${s}' AND w.wort = '${WORT}';
		`);
	});

	test.afterAll(() => {
		seedSQL(`
			DELETE FROM buecher_titel WHERE isbn IN ('978fm${s}', '978fo${s}');
			DELETE FROM schlagworte WHERE wort = '${WORT}';
		`);
	});

	test('Chip wählen zeigt die Titel des Worts, zurücknehmen den Überblick, Text grenzt ein', async ({
		page
	}) => {
		await uiLogin(page, LEHRER_EMAIL);
		await page.getByTitle('Mein Portal').click();

		const filter = page.getByRole('group', { name: 'Nach Schlagwort filtern' });
		const chip = filter.getByRole('button', { name: WORT });
		const treffer = page.locator('h3');
		await expect(chip).toHaveAttribute('aria-pressed', 'false');

		// Ohne Suchtext: alle Titel des Worts — und nur sie.
		await chip.click();
		await expect(chip).toHaveAttribute('aria-pressed', 'true');
		await expect(treffer.filter({ hasText: MIT })).toHaveCount(1);
		await expect(treffer.filter({ hasText: OHNE })).toHaveCount(0);

		// Mit Suchtext grenzt der Filter ein: Die Kennung des Laufs steht in beiden Titeln,
		// der Filter lässt einen übrig.
		const suche = page.getByLabel('Bücher für einen Klassensatz suchen');
		await suche.fill(s);
		await expect(treffer.filter({ hasText: MIT })).toHaveCount(1);
		await expect(treffer.filter({ hasText: OHNE })).toHaveCount(0);

		// Zurücknehmen: Der Suchtext gilt wieder allein — beide Titel.
		await chip.click();
		await expect(chip).toHaveAttribute('aria-pressed', 'false');
		await expect(treffer.filter({ hasText: OHNE })).toHaveCount(1);

		// Weder Text noch Filter: der Überblick (die Lehrkraft hat keine Reservierung).
		await suche.fill('');
		await expect(page.getByText(/Zurzeit wartet keine Reservierung/)).toBeVisible();
	});
});
