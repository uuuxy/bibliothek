// Ein Ausdruck über den Browser trägt alle Zeilen. Der Rahmen der Anwendung hat am Bildschirm
// die Höhe des Fensters; behält er sie auf Papier, endet der Fehlbestandsbericht nach der
// ersten Seite, und wer mit dem Zettel ins Regal geht, sucht nur einen Teil.

import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

const ZEILEN = 150;

test('Der Fehlbestandsbericht druckt alle Zeilen', async ({ page }) => {
	await uiLogin(page);

	// Der Bericht kommt aus einer Attrappe: Gemessen wird der Ausdruck, nicht die Inventur.
	await page.route('**/api/inventur/abgeschlossen', (route) =>
		route.fulfill({
			json: [
				{
					session_id: 'druck-probe',
					label: 'Druckprobe',
					abgeschlossen_am: '2026-10-01T10:00:00Z',
					erfasst: 10,
					verluste: ZEILEN,
					verworfen: false
				}
			]
		})
	);
	await page.route('**/api/inventur/fehlbestand*', (route) =>
		route.fulfill({
			json: Array.from({ length: ZEILEN }, (_, i) => ({
				exemplar_id: `00000000-0000-4000-8000-${String(i).padStart(12, '0')}`,
				barcode_id: `DRUCK-${String(i + 1).padStart(3, '0')}`,
				signatur: `SIG ${String(i + 1).padStart(3, '0')}`,
				titel: `Drucktitel Nummer ${i + 1}`,
				autor: 'Autor'
			}))
		})
	);

	await page.goto('/inventur');
	await page.getByRole('button', { name: 'Fehlbestand' }).click();
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
