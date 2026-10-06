import { test, expect } from '@playwright/test';
import { uiLogin, gehZu, seedSQL, uniqueSuffix } from './helpers.js';

// Die Druckseite einer Ausweiskarte trägt die Karte und nichts sonst.
//
// Die Karte wird über die Druck-CSS des Browsers auf eine Seite von 85,6 × 53,98 mm gesetzt.
// Was auf dem Bildschirm außerhalb eines .no-print-Bereichs steht, druckt mit: Es schiebt die
// Karte nach unten, und ihr unterer Rand mit dem Strichcode fällt von der Seite.
// ausweis-stapeldruck.spec.js prüft, welche Karten im Druckbereich stehen; hier wird gemessen,
// wo sie auf der Seite liegen.

// Der Druckdialog hält den Test an. Die Attrappe bricht ab, bevor der Auslöser Betriebsart
// und Seitenmaß wieder abräumt: Die Seite steht dann so da wie in dem Moment, in dem der
// Dialog aufginge.
async function druckdialogAnhalten(page) {
	await page.addInitScript(() => {
		window.print = () => {
			throw new Error('Druckdialog im Test angehalten');
		};
	});
}

/**
 * Was im Druck zu sehen ist: die Karten und jeder Text, der nicht auf einer Karte steht.
 * @param {import('@playwright/test').Page} page
 */
async function druckseite(page) {
	await page.emulateMedia({ media: 'print' });
	const lage = await page.evaluate(() => {
		const sichtbar = (/** @type {Element} */ e) => {
			const r = e.getBoundingClientRect();
			// Unsichtbare Beschriftungen für Screenreader messen 1 × 1 px.
			return r.width > 1 && r.height > 1 && getComputedStyle(e).visibility !== 'hidden';
		};
		const fremd = [];
		for (const e of document.querySelectorAll('body *')) {
			if (e.children.length > 0 || !e.textContent?.trim()) continue;
			if (sichtbar(e) && !e.closest('.print-card-box'))
				fremd.push(e.textContent.trim().slice(0, 40));
		}
		const karten = [...document.querySelectorAll('.print-card-box')].filter(sichtbar).map((k) => {
			const r = k.getBoundingClientRect();
			return { links: r.left, oben: r.top, breite: r.width, hoehe: r.height };
		});
		return { betriebsart: document.body.dataset.printMode ?? '', fremd, karten };
	});
	await page.emulateMedia({ media: 'screen' });
	return lage;
}

/** @param {Awaited<ReturnType<typeof druckseite>>} lage @param {string} wo */
function erwarteNurKarten(lage, wo) {
	expect(lage.karten.length, `${wo}: Im Druck steht keine Karte.`).toBeGreaterThan(0);
	expect(lage.fremd, `${wo}: Neben der Karte druckt Text der Oberfläche mit.`).toEqual([]);
	expect(lage.karten[0].oben, `${wo}: Die erste Karte beginnt nicht am Seitenanfang.`).toBeCloseTo(
		0,
		0
	);
	expect(lage.karten[0].links, `${wo}: Die Karte beginnt nicht am linken Rand.`).toBeCloseTo(0, 0);
	// 85,6 × 53,98 mm bei 96 dpi.
	expect(lage.karten[0].breite).toBeCloseTo(323.5, 0);
	expect(lage.karten[0].hoehe).toBeCloseTo(204, 0);
}

test.describe('Ausweis: die Druckseite trägt nur die Karte', () => {
	const s = uniqueSuffix();
	const ausweis = `E2E-ADS-${s}`;
	const vorname = `Seite${s}`;

	test.beforeAll(() => {
		seedSQL(`
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
			VALUES ('${ausweis}', '${vorname}', 'Druckprobe', '08a', EXTRACT(YEAR FROM CURRENT_DATE)::int + 3);
		`);
	});

	test.afterAll(() => {
		seedSQL(`DELETE FROM schueler WHERE barcode_id = '${ausweis}';`);
	});

	test('Stapeldruck aus der Leserdatei', async ({ page }) => {
		await druckdialogAnhalten(page);
		await uiLogin(page);
		await gehZu(page, '/schuelerdatei');
		await page.getByRole('searchbox', { name: 'Leser suchen' }).fill(vorname);
		await expect(page.locator('tbody tr').filter({ hasText: vorname })).toHaveCount(1);
		await page.getByRole('checkbox', { name: /Alle angezeigten Leser/ }).check();
		const balken = page.getByRole('region', { name: /Aktionen für die markierten/ });
		await balken.getByRole('button', { name: 'Ausweise drucken' }).click();

		const lage = await druckseite(page);
		expect(lage.betriebsart).toBe('card');
		erwarteNurKarten(lage, 'Stapeldruck');
	});

	test('Einzeldruck aus der Akte', async ({ page }) => {
		await druckdialogAnhalten(page);
		await uiLogin(page);
		await gehZu(page, '/schuelerdatei');
		await page.getByRole('searchbox', { name: 'Leser suchen' }).fill(vorname);
		await page.getByRole('button', { name: new RegExp(`Profil von ${vorname} `) }).click();
		await expect(page.getByRole('tab', { name: /Ausleihen & Vormerkungen/ })).toBeVisible();
		await page.getByRole('button', { name: 'Ausweis drucken', exact: true }).click();

		const lage = await druckseite(page);
		expect(lage.betriebsart).toBe('card-single');
		erwarteNurKarten(lage, 'Einzeldruck');
	});

	test('Testdruck im Ausweis-Designer', async ({ page }) => {
		await druckdialogAnhalten(page);
		await uiLogin(page);
		await gehZu(page, '/druck-center');
		await page.getByRole('tab', { name: 'Schülerausweise' }).click();
		await expect(page.locator('[data-ausweis-vorschau]')).toBeVisible();
		await page.getByRole('button', { name: 'Testdruck Vorderseite' }).click();

		const lage = await druckseite(page);
		expect(lage.betriebsart).toBe('card');
		erwarteNurKarten(lage, 'Testdruck');
	});
});
