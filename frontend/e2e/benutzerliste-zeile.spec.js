import { test, expect } from '@playwright/test';
import { uiLogin, gehZu } from './helpers.js';

// In der Benutzerliste brachen „Bearbeiten" und „Löschen" untereinander um und die
// Ausweisnummer hinter dem Bindestrich, sobald Name oder Adresse lang waren. Die festen Teile
// der Zeile brechen nicht um; nachgeben dürfen Name und Adresse. Gemessen wird im Browser,
// weil nur dort entschieden ist, wie breit eine Spalte wird.
test('Benutzerliste: Knöpfe nebeneinander, Ausweisnummer in einer Zeile', async ({ page }) => {
	await page.setViewportSize({ width: 1024, height: 900 });
	await uiLogin(page);
	await gehZu(page, '/berechtigungen');
	const tabelle = page.getByRole('table', { name: 'Benutzerkonten' });
	await expect(tabelle).toBeVisible();

	const befund = await tabelle.locator('tbody tr').evaluateAll((zeilen) => {
		/** @type {string[]} */
		const gestapelt = [];
		/** @type {string[]} */
		const nummerGebrochen = [];
		for (const zeile of zeilen) {
			const name = zeile.children[0]?.textContent?.trim() ?? '';
			const knoepfe = [...zeile.querySelectorAll('button')].map((k) => k.getBoundingClientRect());
			if (knoepfe.length !== 2 || Math.abs(knoepfe[0].top - knoepfe[1].top) > 2)
				gestapelt.push(name);
			const nummer = zeile.children[2]?.firstElementChild;
			if (!nummer || nummer.getClientRects().length !== 1) nummerGebrochen.push(name);
		}
		return { zeilen: zeilen.length, gestapelt, nummerGebrochen };
	});

	expect(befund.zeilen, 'die Liste ist leer, es wurde nichts gemessen').toBeGreaterThan(0);
	expect(befund.gestapelt, 'Zeilen, in denen die Knöpfe untereinander stehen').toEqual([]);
	expect(befund.nummerGebrochen, 'Zeilen, in denen die Ausweisnummer umbricht').toEqual([]);
});
