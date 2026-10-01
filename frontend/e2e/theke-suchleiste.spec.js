import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

// Die Suchleiste der Theke: ohne getönte Fläche dahinter, der Inhalt 24 px darunter, und beim
// Scrollen bleibt sie stehen. Gemessen am 01.10.2026 lag sie auf einer Fläche in anderer
// Farbe als die Seite, der Inhalt folgte nach 40 px, und bei zwölf Ausleihen rollte sie mit
// nach oben aus dem Fenster.
const s = uniqueSuffix().slice(0, 8);
const AUSWEIS = `S-TS-${s}`;
const FREIES_BUCH = `B-TS-13-${s}`;

/** Lage und Farben der Leiste und, wenn ein Leser geladen ist, der Abstand zum Inhalt. */
function lage() {
	const feld = document.getElementById('omnibox-input');
	const pille = feld?.closest('form');
	const flaeche = pille?.parentElement;
	const seite = document.querySelector('[data-anwendungsrahmen] > div:last-child');
	if (!pille || !flaeche || !seite) return null;
	const p = pille.getBoundingClientRect();
	const inhalt = flaeche.nextElementSibling?.getBoundingClientRect();
	const vorn = document.elementFromPoint(p.left + p.width / 2, p.top + p.height / 2);
	return {
		oben: Math.round(p.top),
		abstandZumInhalt: inhalt ? Math.round(inhalt.top - p.bottom) : null,
		flaeche: getComputedStyle(flaeche).backgroundColor,
		seite: getComputedStyle(seite).backgroundColor,
		liegtVorn: !!vorn && pille.contains(vorn)
	};
}

/** Tippt blind wie ein Handscanner. @param {import('@playwright/test').Page} page @param {string} code */
async function scanne(page, code) {
	await page.keyboard.type(code, { delay: 5 });
	await page.keyboard.press('Enter');
}

test.describe.serial('Theke: Suchleiste', () => {
	test.beforeAll(() => {
		// Zwölf Lernmittel in der Ausleihe, wie ein Schüler sie hat, und ein dreizehntes im Regal.
		seedSQL(`
			WITH sch AS (
				INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
				VALUES ('${AUSWEIS}', 'Leiste${s}', 'Testschueler', '08a', EXTRACT(YEAR FROM CURRENT_DATE)::int + 3)
				RETURNING id
			),
			t AS (
				INSERT INTO buecher_titel (titel, isbn, ist_lernmittel)
				SELECT 'E2E-Leiste ${s} ' || n, '7' || lpad(n::text, 2, '0') || '${s}', true FROM generate_series(1, 13) n
				RETURNING id, titel
			),
			ex AS (
				INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
				SELECT id, 'B-TS-' || split_part(titel, ' ', 3) || '-${s}', true FROM t
				RETURNING id, barcode_id
			)
			INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
			SELECT ex.id, sch.id, CURRENT_DATE + 20 FROM ex, sch WHERE ex.barcode_id <> '${FREIES_BUCH}';
		`);
	});
	test.afterAll(() => {
		seedSQL(`
			DELETE FROM ausleihen WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE barcode_id LIKE 'B-TS-%-${s}');
			DELETE FROM buecher_exemplare WHERE barcode_id LIKE 'B-TS-%-${s}';
			DELETE FROM buecher_titel WHERE titel LIKE 'E2E-Leiste ${s} %';
			DELETE FROM schueler WHERE barcode_id = '${AUSWEIS}';
		`);
	});

	// Die Startlinie misst e2e/suchpille-einheitlich.spec.js für alle Seiten gemeinsam.
	test('die Fläche hinter der Leiste hat die Farbe der Seite', async ({ page }) => {
		await uiLogin(page);
		await page.locator('#omnibox-input').waitFor();
		const ruhe = await page.evaluate(lage);
		expect(ruhe, 'Suchleiste der Theke nicht gefunden').not.toBeNull();
		expect(
			ruhe?.flaeche,
			'hinter der Leiste liegt eine Fläche in anderer Farbe als die Seite'
		).toBe(ruhe?.seite);
	});

	test('zwölf Ausleihen: der Inhalt folgt 24 px unter der Leiste, sie bleibt beim Scrollen stehen, und ein Scan bucht', async ({
		page
	}) => {
		await page.setViewportSize({ width: 1280, height: 900 });
		await uiLogin(page);
		await expect
			.poll(() => page.evaluate(() => document.activeElement?.id ?? ''), { timeout: 5000 })
			.toBe('omnibox-input');
		await scanne(page, AUSWEIS);
		await expect(page.getByText('Entliehene Bücher (12)')).toBeVisible();

		const vorher = await page.evaluate(lage);
		expect(vorher?.abstandZumInhalt, 'Abstand von der Leiste zum Inhalt').toBe(24);

		// Ganz nach unten rollen: Der Bereich, der die Theke scrollt, ist der erste überlaufende
		// Vorfahre der Leiste.
		const gerollt = await page.evaluate(() => {
			for (
				let e = document.getElementById('omnibox-input')?.parentElement;
				e;
				e = e.parentElement
			) {
				if (
					/(auto|scroll)/.test(getComputedStyle(e).overflowY) &&
					e.scrollHeight > e.clientHeight + 1
				) {
					e.scrollTop = e.scrollHeight;
					return e.scrollTop;
				}
			}
			return 0;
		});
		expect(
			gerollt,
			'zwölf Ausleihen laufen bei 1280 × 900 nicht über — der Test misst nichts'
		).toBeGreaterThan(100);
		const unten = await page.evaluate(lage);
		expect(unten?.oben, 'die Leiste ist mit dem Inhalt weggerollt').toBe(vorher?.oben);
		expect(unten?.liegtVorn, 'die Liste liegt über der Leiste').toBe(true);

		// Ein Scan ohne Klick ins Feld bucht, auch in der gerollten Lage.
		await scanne(page, FREIES_BUCH);
		await expect
			.poll(
				() =>
					querySQL(`
						SELECT count(*) FROM ausleihen a JOIN buecher_exemplare e ON e.id = a.exemplar_id
						WHERE e.barcode_id = '${FREIES_BUCH}' AND a.rueckgabe_am IS NULL`),
				{ timeout: 5000 }
			)
			.toBe('1');
	});
});
