// Gate gegen Barrieren, die nur VERHALTEN zeigt — axe sieht sie nicht:
//   1. Ein Dialog nimmt den Fokus, hält ihn (Tab kreist), und gibt ihn beim Schließen an
//      den Auslöser zurück (WCAG 2.4.3, 2.1.2). Gemessen am 08.09.2026: EIN Tab verließ
//      den Dialog; 22 von 24 Dialogen hängen an Dialog.svelte, der Fix sitzt dort.
//   2. Tabellen tragen eine Beschriftung und ihre Kopfzellen einen scope (1.3.1) — im
//      Bauteil ui/Tabelle, nicht in 147 einzelnen <th>.
//   3. Wer im System „Bewegung reduzieren" wählt, bekommt keine Animation (2.3.3) —
//      eine Regel in app.css für alle 107 Animationen.
//
// Dieses Gate wurde vor dem ersten Fix ROT gesehen (09.09.2026).
import { test, expect } from '@playwright/test';
import { uiLogin, gehZu, seedSQL, uniqueSuffix } from './helpers.js';

/** Liegt der Fokus innerhalb des Dialogs? */
const fokusImDialog = (/** @type {import('@playwright/test').Page} */ page) =>
	page.evaluate(() => {
		const dialog = document.querySelector('[role="dialog"]');
		return !!dialog && dialog.contains(document.activeElement);
	});

test('Dialog: Fokus wandert hinein, kreist darin und kehrt zum Auslöser zurück', async ({
	page
}) => {
	await uiLogin(page);
	await gehZu(page, '/schuelerdatei');
	const ausloeser = page.getByRole('button', { name: 'Neuer Leser', exact: true });
	await ausloeser.click();
	await expect(page.getByRole('dialog')).toBeVisible();

	expect(await fokusImDialog(page), 'Fokus nach dem Öffnen').toBe(true);
	for (let i = 0; i < 25; i++) {
		await page.keyboard.press('Tab');
		expect(await fokusImDialog(page), `Fokus nach ${i + 1}× Tab`).toBe(true);
	}
	for (let i = 0; i < 5; i++) {
		await page.keyboard.press('Shift+Tab');
		expect(await fokusImDialog(page), `Fokus nach ${i + 1}× Shift+Tab`).toBe(true);
	}

	await page.keyboard.press('Escape');
	await expect(page.getByRole('dialog')).toHaveCount(0);
	await expect(ausloeser, 'Fokus kehrt zum Auslöser zurück').toBeFocused();
});

test('Tabellen: Beschriftung an der Tabelle, scope an jeder Kopfzelle', async ({ page }) => {
	// Leserdatei und Mahnwesen zeigen ohne Zeile keine Tabelle: Der Test legt einen Leser mit einem
	// überfälligen Buch selbst an, sonst hinge er an den Resten früherer Tests.
	const s = uniqueSuffix();
	seedSQL(`
		WITH t AS (INSERT INTO buecher_titel (titel) VALUES ('E2E-Tabelle ${s}') RETURNING id),
		ex AS (INSERT INTO buecher_exemplare (titel_id, barcode_id) SELECT id, 'B-TAB-${s}' FROM t RETURNING id),
		l AS (INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		      VALUES ('S-TAB-${s}', 'Tabelle', 'Probe-${s}', '07T', 2031) RETURNING id)
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		SELECT ex.id, l.id, CURRENT_DATE - 30 FROM ex, l;
	`);
	try {
		await uiLogin(page);
		/** @type {string[]} */
		const befunde = [];
		for (const [name, pfad] of [
			['Leserdatei', '/schuelerdatei'],
			['Mahnwesen', '/mahnwesen'],
			// Nicht der Medienkatalog: Der ist ein Kachelraster ohne Tabelle.
			['Benutzer & Rechte', '/berechtigungen']
		]) {
			await gehZu(page, pfad);
			await page.locator('table').first().waitFor();
			const fund = await page.evaluate(() => {
				const tabellen = [...document.querySelectorAll('table')];
				const ohneName = tabellen.filter(
					(t) =>
						!t.querySelector('caption') &&
						!t.getAttribute('aria-label') &&
						!t.getAttribute('aria-labelledby')
				).length;
				const ohneScope = [...document.querySelectorAll('table th')].filter(
					(th) => !th.getAttribute('scope')
				).length;
				return { tabellen: tabellen.length, ohneName, ohneScope };
			});
			if (fund.ohneName)
				befunde.push(`${name}: ${fund.ohneName}/${fund.tabellen} Tabellen ohne Beschriftung`);
			if (fund.ohneScope) befunde.push(`${name}: ${fund.ohneScope} <th> ohne scope`);
		}
		expect(befunde).toEqual([]);
	} finally {
		seedSQL(`
			DELETE FROM ausleihen WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE barcode_id = 'B-TAB-${s}');
			DELETE FROM buecher_exemplare WHERE barcode_id = 'B-TAB-${s}';
			DELETE FROM buecher_titel WHERE titel = 'E2E-Tabelle ${s}';
			DELETE FROM leser WHERE barcode_id = 'S-TAB-${s}';
		`);
	}
});

test('Bewegung reduzieren: keine Animation, wenn das System es verlangt', async ({ page }) => {
	await page.emulateMedia({ reducedMotion: 'reduce' });
	await uiLogin(page);
	await gehZu(page, '/kiosk');
	const el = page.locator('.animate-fade-in').first();
	await el.waitFor();
	const dauer = await el.evaluate((n) => getComputedStyle(n).animationDuration);
	// Nicht „animation: none" verlangen — 0s genügt und lässt Endzustände stehen.
	expect(dauer, 'animationDuration bei prefers-reduced-motion').toBe('0s');
});
