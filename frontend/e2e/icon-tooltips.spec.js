// Gate gegen den Icon-Button ohne Erklärung.
//
// Ein Button, der nur ein Symbol trägt, hat keine sichtbare Beschriftung. Vor der
// Normierung hatte genau die Hälfte gar keine Erklärung, die andere Hälfte den nativen
// title-Tooltip des Browsers — Optik des Betriebssystems, rund eine Sekunde Verzögerung,
// bei Tastaturbedienung überhaupt nichts.
//
// Geprüft wird das VERHALTEN, nicht das Attribut: Der Test fährt jedes Symbol an und
// verlangt eine sichtbare Blase mit Text. Ein `data-tip=""` oder eine Blase, die im
// overflow-Container der Bestellhistorie abgeschnitten wird, bestünde eine reine
// Attribut-Prüfung — hier fällt beides durch.
import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

const SCREENS = [
	['Bestellungen', '/bestellungen'],
	['Mahnwesen', '/mahnwesen'],
	['Schülerdatei', '/schuelerdatei'],
	['Druck-Center', '/druck-center'],
	['Einstellungen', '/einstellungen']
];

/** Sichtbare Buttons, deren Inhalt praktisch nur ein Symbol ist. */
const SYMBOL_BUTTONS =
	'button:has(svg), [role="button"]:has(svg)';

/**
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<number>} Anzahl der geprüften Symbole
 */
async function pruefeBildschirm(page, name, fehler) {
	const kandidaten = await page.locator(SYMBOL_BUTTONS).all();
	let geprueft = 0;

	for (const b of kandidaten) {
		if (!(await b.isVisible().catch(() => false))) continue;
		const text = ((await b.textContent()) || '').trim();
		if (text.length > 3) continue; // beschriftet — braucht keine Blase
		const box = await b.boundingBox();
		if (!box) continue;

		// Bewusste Ausnahme: Elemente, die sich beim Überfahren selbst erklären (die
		// Cover-Vorschau zeigt das Cover). Die Begründung steht als Wert des Attributs an
		// der Fundstelle — nicht als Liste hier im Test, wo sie niemand liest.
		if (await b.getAttribute('data-tip-eigen')) continue;

		const tip = await b.getAttribute('data-tip');
		const aria = await b.getAttribute('aria-label');
		geprueft++;

		if (!tip) {
			fehler.push(`${name}: "${aria || '(ohne Label)'}" hat kein data-tip`);
			continue;
		}

		// Verhalten: Die Blase muss erscheinen UND lesbaren Text tragen.
		await b.hover();
		const blase = page.locator('[data-tooltip-blase]');
		try {
			await expect(blase).toBeVisible({ timeout: 2000 });
			await expect(blase).toHaveText(tip, { timeout: 2000 });
		} catch {
			fehler.push(`${name}: "${aria}" zeigt beim Überfahren keine Blase mit "${tip}"`);
		}
		// Wegfahren, sonst bleibt die Blase für das nächste Symbol stehen.
		await page.mouse.move(0, 0);
		await expect(blase).toBeHidden({ timeout: 2000 }).catch(() => {});
	}
	return geprueft;
}

test('Jedes Symbol erklärt sich beim Überfahren', async ({ page }) => {
	test.setTimeout(180_000);
	await uiLogin(page);

	/** @type {string[]} */
	const fehler = [];
	let geprueft = 0;

	for (const [name, pfad] of SCREENS) {
		await page.goto(pfad);
		await page.waitForLoadState('domcontentloaded');
		// Nicht auf <main> warten: Die Anwendung rendert zwei davon (Rahmen + Inhalt).
		// Der Abmelden-Knopf steht auf jedem Bildschirm und erst, wenn die App steht.
		await page.getByRole('button', { name: 'Abmelden' }).waitFor();
		geprueft += await pruefeBildschirm(page, name, fehler);
	}

	// Die Bestellhistorie ausdrücklich: Ihre Symbole stecken in einer zugeklappten Zeile
	// UND in einem overflow-x-auto-Container — genau dort würde eine normal positionierte
	// Blase abgeschnitten. Ohne stille Wächter: Findet der Test den Messpunkt nicht,
	// gehört er rot (ein übersprungener Messpunkt sieht aus wie ein bestandener).
	await page.goto('/bestellungen');
	await page.getByRole('button', { name: 'Bestellhistorie' }).click();
	const zeilen = page.locator('tbody tr');
	await zeilen.first().waitFor();
	await zeilen.first().click();
	await page.getByRole('columnheader', { name: 'ISBN' }).waitFor();
	geprueft += await pruefeBildschirm(page, 'Bestellhistorie (aufgeklappt)', fehler);

	expect(
		fehler,
		'Symbole ohne Erklärung.\nBehebung: data-tip="…" an das Element schreiben ' +
			'(wirkt auch an der Button-Komponente, sie reicht Rest-Props durch).\n\n' +
			fehler.join('\n')
	).toEqual([]);

	// Selbstschutz: Bricht eine Navigation, prüft der Test still nichts mehr.
	expect(geprueft, 'Der Test hat fast keine Symbole gefunden').toBeGreaterThan(5);
});
