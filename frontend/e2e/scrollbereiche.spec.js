import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix, oeffneSchuelerProfil, gehZu } from './helpers.js';

// Eine Liste hält alle ihre Zeilen; gescrollt wird der Bereich, in dem auch ihre Überschrift
// steht. Ein Kasten um die Liste fällt erst mit Mengen wie an der Schule auf, acht bis
// achtzehn Ausleihen je Leser.

/**
 * Die Scrollbereiche um eine Liste, von innen nach außen. Die Liste ist die Tabelle mit
 * dieser Beschriftung oder, wo sie keine Tabelle ist, das Element mit dem Text einer Zeile.
 * @param {{ beschriftung?: string, zeile?: string, ueberschrift: string }} suche
 */
function scrollLage({ beschriftung, zeile, ueberschrift }) {
	const tabellen = [...document.querySelectorAll('table')].filter(
		(t) => beschriftung && t.querySelector('caption')?.textContent?.includes(beschriftung)
	);
	const liste = zeile
		? [...document.querySelectorAll('span')].find((e) => e.textContent?.trim() === zeile)
		: tabellen[0];
	const kopf = [...document.querySelectorAll('h2, h3')].find((h) =>
		h.textContent?.includes(ueberschrift)
	);
	const seite = document.scrollingElement;
	if (!liste || !kopf || !seite) return null;
	const bereiche = [];
	for (let e = liste.parentElement; e; e = e.parentElement) {
		if (e === seite || /(auto|scroll)/.test(getComputedStyle(e).overflowY)) bereiche.push(e);
	}
	return {
		tabellen: tabellen.length,
		ueberschriftImSelbenBereich: bereiche[0].contains(kopf),
		ueberlaufend: bereiche.filter((e) => e.scrollHeight > e.clientHeight + 1).length,
		seiteLaeuftUeber: seite.scrollHeight > window.innerHeight + 1
	};
}

test('Leserakte: zwölf Ausleihen stehen in einer Liste ohne eigenen Scrollkasten', async ({
	page
}) => {
	const s = uniqueSuffix();
	seedSQL(`
		WITH sch AS (
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
			VALUES ('E2E-SCR-S-${s}', 'Scroll${s}', 'Testschueler', '08a', EXTRACT(YEAR FROM CURRENT_DATE)::int + 3)
			RETURNING id
		),
		t AS (
			INSERT INTO buecher_titel (titel, isbn)
			SELECT 'E2E-Scrollbuch ${s} ' || n, '9' || n || '${s}' FROM generate_series(1, 12) n
			RETURNING id
		),
		ex AS (
			INSERT INTO buecher_exemplare (titel_id, barcode_id)
			SELECT id, 'E2E-SCR-B-${s}-' || row_number() OVER () FROM t
			RETURNING id
		)
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		SELECT ex.id, sch.id, CURRENT_DATE + 20 FROM ex, sch;
	`);

	await uiLogin(page);
	await oeffneSchuelerProfil(page, `Scroll${s}`);
	await expect(page.getByText('Entliehene Bücher (12)')).toBeVisible();

	const lage = await page.evaluate(scrollLage, {
		beschriftung: 'Ausgeliehene Bücher',
		ueberschrift: 'Entliehene Bücher'
	});
	expect(lage, 'Tabelle oder Überschrift der Ausleihliste nicht gefunden').not.toBeNull();
	expect(
		lage?.ueberschriftImSelbenBereich,
		'Die Ausleihliste scrollt in einem eigenen Kasten, getrennt von ihrer Überschrift.'
	).toBe(true);
	expect(
		lage?.ueberlaufend,
		'Um die Ausleihliste scrollen mehrere Bereiche ineinander.'
	).toBeLessThanOrEqual(1);
});

test('Wareneingang: die Positionen scrollen mit der Seite, und nur ein Bereich scrollt', async ({
	page
}) => {
	const s = uniqueSuffix();
	// Vier Lieferanten, also vier Tabellen untereinander: Nur eine Tabelle unterhalb des
	// Fensterrands zeigt, ob ihre unsichtbare Beschriftung die Seite verlängert.
	seedSQL(`
		WITH t AS (
			INSERT INTO buecher_titel (isbn, titel, autor)
			SELECT '8' || n || '${s}', 'E2E Scroll-Zulauf ${s} ' || n, 'Zulauf Autor'
			FROM generate_series(1, 8) n
			RETURNING id, titel
		)
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, etikett_gedruckt, zustand_notiz, bestellstatus)
		SELECT id, 'B-scr-${s}-' || right(titel, 1), false, false,
		       'Im Zulauf - E2E-Scroll-Lieferant ${s}-' || (right(titel, 1)::int + 1) / 2, 'im_zulauf'
		FROM t;
	`);

	await uiLogin(page);
	await gehZu(page, '/bestellungen');
	await page.getByRole('tab', { name: /Wareneingang/ }).click();
	await expect(page.getByRole('heading', { name: 'Wareneingang bearbeiten' })).toBeVisible();
	await expect(page.getByText(`E2E Scroll-Zulauf ${s} 8`)).toBeAttached();

	const lage = await page.evaluate(scrollLage, {
		beschriftung: 'Bestellte Exemplare im Zulauf',
		ueberschrift: 'Wareneingang bearbeiten'
	});
	expect(lage, 'Tabelle oder Überschrift des Wareneingangs nicht gefunden').not.toBeNull();
	expect(lage?.tabellen, 'Die Testdaten ergeben keine vier Lieferanten.').toBeGreaterThanOrEqual(4);
	expect(
		lage?.ueberschriftImSelbenBereich,
		'Die Positionen scrollen in einem eigenen Kasten, getrennt von ihrer Überschrift.'
	).toBe(true);
	expect(
		lage?.seiteLaeuftUeber,
		'Die unsichtbare Beschriftung einer Tabelle verlängert die Seite über das Fenster hinaus.'
	).toBe(false);
	expect(
		lage?.ueberlaufend,
		'Im Wareneingang scrollen mehrere Bereiche ineinander.'
	).toBeLessThanOrEqual(1);

	// Einbuchen steht in der Leiste unter der Liste und bleibt beim Scrollen im Fenster. Der
	// Knopf im Kopf der Seite war bei einer langen Lieferung aus dem Bild.
	await expect(page.getByRole('button', { name: 'Einbuchen', exact: true })).toHaveCount(0);
	await page.getByRole('checkbox', { name: `E2E Scroll-Zulauf ${s} 1 auswählen` }).check();
	const leiste = page.getByRole('region', { name: 'Aktionen für die markierten Positionen' });
	await expect(leiste).toContainText('1 Exemplar markiert');
	await page.getByText(`E2E Scroll-Zulauf ${s} 8`).scrollIntoViewIfNeeded();
	await expect(leiste.getByRole('button', { name: 'Einbuchen', exact: true })).toBeInViewport();
	await page.getByRole('heading', { name: 'Wareneingang bearbeiten' }).scrollIntoViewIfNeeded();
	await expect(leiste.getByRole('button', { name: 'Einbuchen', exact: true })).toBeInViewport();
	await leiste.getByRole('button', { name: 'Markierung aufheben' }).click();
	await expect(leiste).toHaveCount(0);
});

test('Buchmaske: zwanzig Exemplare stehen in einer Liste ohne eigenen Scrollkasten', async ({
	page
}) => {
	const s = uniqueSuffix();
	seedSQL(`
		WITH t AS (
			INSERT INTO buecher_titel (isbn, titel, autor)
			VALUES ('978s${s}', 'E2E Scrollmaske ${s}', 'Scroll Autor')
			RETURNING id
		)
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
		SELECT id, 'B-scm-${s}-' || lpad(n::text, 2, '0'), true FROM t, generate_series(1, 20) n;
	`);

	try {
		await uiLogin(page);
		await gehZu(page, '/medienkatalog');
		await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
		await page.getByRole('searchbox', { name: 'Bücher durchsuchen' }).fill(`E2E Scrollmaske ${s}`);
		await page.getByText(`E2E Scrollmaske ${s}`).first().click();
		await expect(page.getByRole('heading', { name: 'Exemplare (20)' })).toBeVisible();

		const lage = await page.evaluate(scrollLage, {
			zeile: `B-scm-${s}-01`,
			ueberschrift: 'Exemplare (20)'
		});
		expect(lage, 'Liste oder Überschrift der Exemplare nicht gefunden').not.toBeNull();
		expect(
			lage?.ueberschriftImSelbenBereich,
			'Die Exemplare scrollen in einem eigenen Kasten, getrennt von ihrer Überschrift.'
		).toBe(true);
		expect(
			lage?.ueberlaufend,
			'In der Buchmaske scrollen mehrere Bereiche ineinander.'
		).toBeLessThanOrEqual(1);
	} finally {
		// Zwanzig Exemplare ohne gedrucktes Etikett stünden sonst nach jedem Lauf im Druck-Center.
		seedSQL(`
			DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE isbn = '978s${s}');
			DELETE FROM buecher_titel WHERE isbn = '978s${s}';
		`);
	}
});
