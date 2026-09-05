import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix, gehZu } from './helpers.js';

// Zwei Suchfelder außerhalb der Theke riefen bis 05.09.2026 die Theken-Suche GET /api/search
// (perform_actions) und lasen je eine Hälfte der Antwort: die Etiketten-Titelsuche nur die
// Titel (und holte Schüler-Kiosk-Daten mit, die sie nie zeigte), die Vormerkungs-Schülersuche
// nur die Schüler. Entzog ein Admin einer Rolle das Theken-Recht, meldeten beide „Suche nicht
// möglich". Seit dem 05.09.2026 fragen sie ihre eigene Tür — dieses Gate misst am Draht, WELCHE
// Tür der Browser wirklich ruft, nicht nur, dass ein Treffer erscheint (die Quelltext-Ratsche
// frontend-hygiene-action-endpunkt.test.js sieht nur den Text, nicht den Lauf).

test('Etiketten-Titelsuche im Druck-Center fragt die Titel-Tür, nicht die Theken-Suche', async ({
	page
}) => {
	const s = uniqueSuffix();
	seedSQL(`
        INSERT INTO buecher_titel (isbn, titel, autor, verlag)
        VALUES ('978t${s}', 'E2E Tuersuche ${s}', 'Tuer Autor', 'Tuerverlag');
    `);

	await uiLogin(page);
	await gehZu(page, '/druck-center');
	await page.getByText('Buch-Etiketten', { exact: true }).first().click();

	/** @type {string[]} */
	const anfragen = [];
	page.on('request', (r) => anfragen.push(r.url()));

	await page.getByRole('searchbox', { name: 'Buchtitel im Katalog suchen' }).fill(`Tuersuche ${s}`);
	await expect(page.getByRole('button', { name: new RegExp(`E2E Tuersuche ${s}`) })).toBeVisible();

	expect(
		anfragen.filter((u) => u.includes('/api/buecher/titel/suche?q=')),
		'die Titel-Tür wurde gerufen'
	).not.toEqual([]);
	expect(
		anfragen.filter((u) => u.includes('/api/search')),
		'die Theken-Suche darf der Druckbildschirm nicht rufen'
	).toEqual([]);
});

test('Vormerkungs-Schülersuche in der Buchakte fragt die Schülerdatei, nicht die Theken-Suche', async ({
	page
}) => {
	const s = uniqueSuffix();
	seedSQL(`
        INSERT INTO buecher_titel (isbn, titel) VALUES ('978v${s}', 'E2E Vormerkbuch ${s}');
        INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr)
        VALUES ('Tuerchen', 'Vormerk${s}', '07A', 'S-VM-${s}', 2031);
    `);
	const titelId = querySQL(`SELECT id FROM buecher_titel WHERE isbn = '978v${s}'`);

	await uiLogin(page);
	await gehZu(page, `/medienkatalog/buch/${titelId}`);
	await page
		.getByText(/^Vormerkungen \(/)
		.first()
		.click();
	await page.getByRole('button', { name: '+ Schüler vormerken' }).click();

	/** @type {string[]} */
	const anfragen = [];
	page.on('request', (r) => anfragen.push(r.url()));

	await page.locator('#student-search-input').fill(`Vormerk${s}`);
	await page.getByRole('button', { name: 'Suchen', exact: true }).click();
	await expect(page.getByText(`Tuerchen Vormerk${s}`)).toBeVisible();

	expect(
		anfragen.filter((u) => u.includes('/api/schueler?q=')),
		'die Schülerdatei wurde gerufen'
	).not.toEqual([]);
	expect(
		anfragen.filter((u) => u.includes('/api/search')),
		'die Theken-Suche darf die Buchakte nicht rufen'
	).toEqual([]);
});
