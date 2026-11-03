import { test, expect } from '@playwright/test';
import { uiLogin, querySQL, seedSQL, uniqueSuffix } from './helpers.js';

// Die Kacheln des Medienkatalogs stehen in einem ruhigen Raster: Alle Kacheln einer Reihe sind
// gleich hoch, und ISBN-Balken und Fußzeile liegen auf einer Linie. Das tragen die festen
// Höhen der Zeilen unter dem Cover (BuchKarte.svelte), auch wenn ein Titel lang ist, keine ISBN
// oder kein Prüfdatum hat oder eine Auflage trägt. Gemessen wird im Browser, weil die Lage erst
// aus Schrift, Umbruch und Raster entsteht.
const s = uniqueSuffix();
/** @type {string[]} */
let titelIds = [];

test.beforeEach(() => {
	const ausgabe = querySQL(`
		WITH t AS (
			INSERT INTO buecher_titel (titel, autor, isbn, last_counted, auflage, signatur, jahrgang_von, jahrgang_bis)
			VALUES
				('E2E-Raster ${s} kurz', 'Rasterprobe ${s}', 'RAS1${s.slice(-12)}', DATE '2026-03-26', NULL, NULL, NULL, NULL),
				('E2E-Raster ${s} Naturwissenschaftsunterricht, ein langer Titel über mehrere Zeilen',
				 'Rasterprobe ${s}', NULL, NULL, '3. Aufl.', NULL, NULL, NULL),
				('E2E-Raster ${s} mittel', 'Rasterprobe ${s}', 'RAS3${s.slice(-12)}', NULL, NULL, 'Spa 7', 7, 7)
			RETURNING id
		),
		ex AS (
			INSERT INTO buecher_exemplare (titel_id, barcode_id)
			SELECT id, 'B-RAST-' || left(id::text, 8) || '-${s}' FROM t
			RETURNING titel_id
		)
		SELECT titel_id FROM ex;
	`);
	titelIds = ausgabe.split('\n').filter(Boolean);
});

test.afterEach(() => {
	if (titelIds.length === 0) return;
	const liste = titelIds.map((id) => `'${id}'`).join(', ');
	seedSQL(`
		DELETE FROM buecher_exemplare WHERE titel_id IN (${liste});
		DELETE FROM buecher_titel WHERE id IN (${liste});
	`);
});

test('Kacheln einer Reihe sind gleich hoch, ISBN-Balken und Fußzeile liegen auf einer Linie', async ({
	page
}) => {
	await uiLogin(page);
	await page.goto('/medienkatalog');
	await page.locator('#katalog-suchfeld').fill(`Rasterprobe ${s}`);
	const koepfe = page.getByRole('heading', { level: 2, name: /^E2E-Raster / });
	await expect(koepfe).toHaveCount(3);
	const karten = page.locator('div.m3-state').filter({ has: koepfe });
	await expect(karten).toHaveCount(3);

	const masse = await karten.evaluateAll((liste) =>
		liste.map((karte) => {
			const oben = (/** @type {Element | null | undefined} */ el) =>
				el ? Math.round(el.getBoundingClientRect().top) : null;
			// Der ISBN-Balken ist der Knopf zum Kopieren oder, ohne ISBN, der Platzhalter.
			const isbn = [...karte.querySelectorAll('button, div')].find(
				(el) => /kopieren|Keine ISBN/.test(el.textContent ?? '') && !el.querySelector('button, div')
			);
			return {
				oben: oben(karte),
				hoehe: Math.round(karte.getBoundingClientRect().height),
				isbn: oben(isbn),
				stift: oben(karte.querySelector('[aria-label="Buch schnell bearbeiten"]'))
			};
		})
	);

	const bericht = JSON.stringify(masse);
	expect(
		masse.every((m) => m.isbn !== null && m.stift !== null),
		`ISBN-Balken und Stift in jeder Kachel: ${bericht}`
	).toBe(true);
	expect(new Set(masse.map((m) => m.oben)).size, `eine Reihe: ${bericht}`).toBe(1);
	expect(new Set(masse.map((m) => m.hoehe)).size, `gleich hoch: ${bericht}`).toBe(1);
	expect(new Set(masse.map((m) => m.isbn)).size, `ISBN-Balken auf einer Linie: ${bericht}`).toBe(1);
	expect(new Set(masse.map((m) => m.stift)).size, `Fußzeile auf einer Linie: ${bericht}`).toBe(1);

	// Ohne Bild ist der Platzhalter die einzige sichtbare Stelle des Titels: Ein langes Wort bricht
	// dort um, statt rechts abgeschnitten zu werden (BuchCover, PLATZHALTER).
	const ohneBild = karten.filter({
		has: page.getByRole('heading', { level: 2, name: /Naturwissenschaftsunterricht/ })
	});
	const ueberlauf = await ohneBild.evaluate((karte) =>
		[...karte.querySelectorAll('h2 button p')].map((p) => p.scrollWidth - p.clientWidth)
	);
	expect(ueberlauf, 'Fach und Titel im Platzhalter').toHaveLength(2);
	expect(Math.max(...ueberlauf), `Überlauf je Zeile in px: ${ueberlauf}`).toBeLessThanOrEqual(0);
});
