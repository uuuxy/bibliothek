// Zwei Wege aufs Papier. Der Druck des Browsers (Strg+P) trägt alle Zeilen: Der Rahmen der
// Anwendung hat am Bildschirm die Höhe des Fensters; behält er sie auf Papier, endet eine lange
// Seite nach dem ersten Blatt. „Liste drucken" im Fehlbestandsbericht öffnet ein eigenes Blatt
// mit dem, was noch zu suchen ist.

import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

const ZEILEN = 150;
const LANG =
	'Elemente der Mathematik 7 – Gymnasium G9 Hessen: Schülerband mit Lösungen zu den Kontrollaufgaben';

/** @param {number} i */
const kennung = (i) => `00000000-0000-4000-8000-${String(i).padStart(12, '0')}`;

/**
 * Läuft im Browser: über wie viele Zeilen der Text des Elements geht, ob davon etwas
 * abgeschnitten ist und ob die Seite breiter ist als ihr Fenster.
 * @param {Element} el
 */
function lage(el) {
	const bereich = document.createRange();
	bereich.selectNodeContents(el);
	const seite = document.documentElement;
	return {
		zeilen: new Set([...bereich.getClientRects()].map((r) => Math.round(r.top))).size,
		abgeschnitten: el.scrollWidth > el.clientWidth,
		ragtHinaus: seite.scrollWidth > seite.clientWidth
	};
}

/**
 * Öffnet den Fehlbestandsbericht einer früheren Inventur. Er kommt aus einer Attrappe:
 * Gemessen wird der Ausdruck, nicht die Inventur.
 * @param {import('@playwright/test').Page} page
 * @param {any[]} eintraege
 */
async function oeffneBericht(page, eintraege) {
	await page.route('**/api/inventur/abgeschlossen', (route) =>
		route.fulfill({
			json: [
				{
					session_id: 'druck-probe',
					label: 'Druckprobe',
					abgeschlossen_am: '2026-10-01T10:00:00Z',
					erfasst: 10,
					verluste: eintraege.length,
					verworfen: false
				}
			]
		})
	);
	await page.route('**/api/inventur/fehlbestand*', (route) => route.fulfill({ json: eintraege }));
	await page.goto('/inventur');
	await page.getByRole('button', { name: 'Fehlbestand' }).click();
}

test('Der Fehlbestandsbericht druckt alle Zeilen', async ({ page }) => {
	await uiLogin(page);
	await oeffneBericht(
		page,
		Array.from({ length: ZEILEN }, (_, i) => ({
			exemplar_id: kennung(i),
			barcode_id: `DRUCK-${String(i + 1).padStart(3, '0')}`,
			signatur: `SIG ${String(i + 1).padStart(3, '0')}`,
			titel: `Drucktitel Nummer ${i + 1}`,
			autor: 'Autor'
		}))
	);
	const letzteZeile = page.getByText(`Drucktitel Nummer ${ZEILEN}`, { exact: true });
	await letzteZeile.waitFor({ state: 'attached' });

	await page.emulateMedia({ media: 'print' });

	// Kein Vorfahr schneidet die letzte Zeile ab.
	const schneidetAb = await letzteZeile.evaluate((zeile) => {
		const unten = zeile.getBoundingClientRect().bottom;
		const treffer = [];
		for (let el = zeile.parentElement; el; el = el.parentElement) {
			if (getComputedStyle(el).overflowY === 'visible') continue;
			if (unten > el.getBoundingClientRect().bottom + 1) {
				treffer.push(`${el.tagName.toLowerCase()}.${String(el.className).slice(0, 60)}`);
			}
		}
		return treffer;
	});
	expect(schneidetAb, 'Vorfahren, die die letzte Zeile im Druck abschneiden').toEqual([]);

	// Und der Ausdruck selbst: 150 Zeilen passen nicht auf drei Seiten.
	const pdf = await page.pdf({ format: 'A4' });
	const seiten = (pdf.toString('latin1').match(/\/Type\s*\/Page\b(?!s)/g) ?? []).length;
	expect(seiten, 'Seiten im Ausdruck').toBeGreaterThan(3);
});

// „Liste drucken" öffnet ein eigenes Blatt mit dem, was noch zu suchen ist. Ein langer Titel
// steht dort ganz: Auf dem Papier führt kein Weg zum abgeschnittenen Rest.
test('„Liste drucken" im Fehlbestandsbericht: offene Exemplare, ganzer Titel, Kästchen', async ({
	page
}) => {
	await uiLogin(page);
	await oeffneBericht(page, [
		{
			exemplar_id: kennung(1),
			barcode_id: 'DRUCK-001',
			signatur: 'Ma 7.1',
			titel: LANG,
			autor: 'Griesel, Heinz'
		},
		{
			exemplar_id: kennung(2),
			barcode_id: 'DRUCK-002',
			signatur: 'Bel SCH',
			titel: 'Der Vorleser',
			autor: 'Schlink, Bernhard',
			gefunden_am: '2026-10-05T09:00:00+02:00'
		},
		// Ohne exemplar_id: schon endgültig gelöscht, zu suchen gibt es nichts mehr.
		{ barcode_id: 'DRUCK-003', signatur: 'Ek 0.1', titel: 'Diercke Weltatlas', autor: '' }
	]);

	// Am Bildschirm gehört die übrige Breite dem Titel, nicht Signatur und Barcode.
	const tabelle = page.getByRole('table', { name: 'Fehlbestand' });
	/** @param {string} name */
	const breite = async (name) =>
		(await tabelle.getByRole('columnheader', { name, exact: true }).boundingBox())?.width ?? 0;
	expect(await breite('Titel'), 'Breite der Titelspalte').toBeGreaterThan(
		(await breite('Signatur')) + (await breite('Barcode'))
	);
	// Der lange Titel läuft um und steht ganz da: Gekürzt wäre der Rest ohne Maus nicht zu lesen.
	const amSchirm = await tabelle.getByText(LANG, { exact: true }).evaluate(lage);
	expect(amSchirm.zeilen, 'Zeilen des langen Titels am Bildschirm').toBeGreaterThan(1);
	expect(amSchirm.abgeschnitten, 'der Titel ist am Bildschirm abgeschnitten').toBe(false);

	const fenster = page.waitForEvent('popup');
	await page.getByRole('button', { name: 'Liste drucken' }).click();
	const blatt = await fenster;
	await expect(blatt.locator('h1')).toHaveText('Fehlbestand — Druckprobe');
	await expect(blatt.locator('p.meta')).toContainText(
		'Als Verlust gebucht: 3 | bereits geklärt: 2 | noch offen: 1'
	);
	const zeile = blatt.locator('tbody tr');
	await expect(zeile).toHaveCount(1);
	await expect(zeile).toContainText('DRUCK-001');

	// Auf einem schmalen Blatt läuft der Titel um und ragt nicht über den Rand.
	await blatt.setViewportSize({ width: 560, height: 800 });
	const titel = zeile.locator('td').nth(1);
	await expect(titel).toHaveText(LANG);
	const aufPapier = await titel.evaluate(lage);
	expect(aufPapier.zeilen, 'Zeilen des langen Titels auf dem Blatt').toBeGreaterThan(1);
	expect(aufPapier.abgeschnitten, 'der Titel ist auf dem Blatt abgeschnitten').toBe(false);
	expect(aufPapier.ragtHinaus, 'das Blatt ragt über den Rand').toBe(false);

	// Das Kästchen zum Abhaken ist gezeichnet, die Zelle ist nicht nur leer.
	const kaestchen = await zeile
		.locator('td')
		.last()
		.evaluate((zelle) => {
			const stil = getComputedStyle(zelle, '::before');
			return {
				rahmen: stil.borderTopStyle,
				breite: parseFloat(stil.width),
				hoehe: parseFloat(stil.height)
			};
		});
	expect(kaestchen.rahmen).toBe('solid');
	expect(kaestchen.breite).toBeGreaterThan(10);
	expect(kaestchen.hoehe).toBeGreaterThan(10);
});
