import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix, ADMIN_PASSWORD } from './helpers.js';

// Die Sperre nach Inaktivität baute die Bildschirme ab; was getippt und nicht gespeichert
// war, war nach dem Aufschließen weg (docs/OFFEN.md 5.44). Jetzt bleibt die Anwendung hinter
// dem Sperrbildschirm stehen — ausgeblendet und träge. Die drei Gründe, aus denen sie am
// 22.08.2026 abgebaut wurde, gelten weiter und stehen hier als Prüfung: Die Druckvorschau
// zeigt sie nicht, Tab erreicht sie nicht, Screenreader lesen sie nicht.
const s = uniqueSuffix().slice(0, 6);
const kern = ('978' + String(Date.now()).slice(-9)).slice(0, 12);
const ISBN =
	kern + ((10 - ([...kern].reduce((a, d, i) => a + Number(d) * (i % 2 ? 3 : 1), 0) % 10)) % 10);
const TITEL = `Sperrprobe ${s}`;

/** Anmelden mit gestellter Uhr; die Fristen der Sperre sind geladen.
 * @param {import('@playwright/test').Page} page */
async function anmelden(page) {
	await page.clock.install();
	const fristen = page.waitForResponse(
		(r) => new URL(r.url()).pathname === '/api/einstellungen/sitzung'
	);
	await uiLogin(page);
	await fristen;
	// Ohne Neuladen in die Titel-Verwaltung: Ein Neuladen stellte die Uhr der Sperre neu.
	await page.getByTitle('Medienkatalog').click();
	await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
}

/** @param {import('@playwright/test').Page} page */
async function sperre(page) {
	await page.clock.fastForward('15:10');
	const bildschirm = page.getByTestId('sperrbildschirm');
	await expect(bildschirm).toBeVisible();
	return bildschirm;
}

/** @param {import('@playwright/test').Page} page */
async function schliesseAuf(page) {
	await page.locator('#sperre-passwort').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Entsperren' }).click();
	await expect(page.getByTestId('sperrbildschirm')).toHaveCount(0);
}

/** Der Text, den die Seite zeigt — und druckt, und den ein Screenreader liest.
 * @param {import('@playwright/test').Page} page */
const gezeigterText = (page) => page.evaluate(() => document.body.innerText);

test.describe('Sperre: Ungespeichertes bleibt', () => {
	test.afterAll(() => {
		seedSQL(`
			DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE isbn = '${ISBN}');
			DELETE FROM buecher_titel WHERE isbn = '${ISBN}';`);
	});

	test('eine halb ausgefüllte Maske steht nach dem Aufschließen noch da; gesperrt ist sie weder zu sehen noch zu erreichen noch zu drucken', async ({
		page
	}) => {
		await anmelden(page);
		await page.getByRole('button', { name: 'Neues Buch' }).first().click();
		await page.locator('#buch-titel').fill(`Ungespeichert ${s}`);
		await page.locator('#buch-signatur').fill('UNG 1');

		const bildschirm = await sperre(page);

		// Nicht zu sehen — auf dem Bildschirm und in der Druckvorschau.
		await expect(page.locator('#buch-titel')).toBeHidden();
		for (const medium of /** @type {const} */ (['screen', 'print'])) {
			await page.emulateMedia({ media: medium });
			const text = await gezeigterText(page);
			expect(text, `${medium}: die Maske hinter der Sperre`).not.toContain('Neues Buch');
			expect(text, `${medium}: die Seitenleiste hinter der Sperre`).not.toContain('Leserdatei');
			expect(text, medium).toContain('Gesperrt wegen Inaktivität');
		}
		await page.emulateMedia({ media: 'screen' });

		// Nicht für Screenreader: Im Zugänglichkeitsbaum steht nur der Sperrbildschirm.
		await expect(page.getByRole('navigation')).toHaveCount(0);
		await expect(page.getByRole('textbox')).toHaveCount(1);
		await expect(page.getByRole('heading', { name: 'Neues Buch' })).toHaveCount(0);

		// Nicht mit der Tastatur zu erreichen: Tab bleibt im Sperrbildschirm.
		for (let i = 0; i < 8; i++) {
			await page.keyboard.press('Tab');
			expect(
				await page.evaluate(
					() => !!document.activeElement?.closest('[data-testid="sperrbildschirm"]')
				),
				`nach ${i + 1}× Tab steht der Fokus außerhalb des Sperrbildschirms`
			).toBe(true);
		}

		// Tasten und die Zurück-Taste des Browsers erreichen die Maske dahinter nicht: Escape
		// führte sonst an die Theke. Einmal aus dem Passwortfeld, einmal ohne Fokus in einem Feld.
		await page.locator('#sperre-passwort').focus();
		await page.keyboard.press('Escape');
		await bildschirm.click({ position: { x: 5, y: 5 } });
		await page.keyboard.press('Escape');
		await page.goBack();
		await expect(bildschirm).toBeVisible();

		await schliesseAuf(page);
		await expect(page.getByRole('heading', { name: 'Neues Buch' })).toBeVisible();
		await expect(page.locator('#buch-titel')).toHaveValue(`Ungespeichert ${s}`);
		await expect(page.locator('#buch-signatur')).toHaveValue('UNG 1');
		await expect(page).toHaveURL(/\/medienkatalog/);
		// Der Fokus steht wieder in dem Feld, in dem getippt wurde.
		await expect(page.locator('#buch-signatur')).toBeFocused();
	});

	test('eine offene Rückfrage ist hinter der Sperre nicht zu sehen und steht danach wieder da', async ({
		page
	}) => {
		seedSQL(`
			INSERT INTO buecher_titel (titel, autor, isbn, signatur, medientyp, cover_url)
			VALUES ('${TITEL}', 'E2E', '${ISBN}', 'Spe 1', 'Buch', '/covers/e2e-dummy.jpg');
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
			SELECT id, 'B-SP-' || g || '-${s}', true FROM buecher_titel, generate_series(1, 3) g WHERE isbn = '${ISBN}';`);
		await anmelden(page);
		await page.getByRole('searchbox', { name: 'Bücher durchsuchen' }).fill(TITEL);
		await page.getByText(TITEL).first().click();
		await expect(page.locator('#buch-bestand')).toHaveValue('3');
		await page.locator('#buch-bestand').fill('1');
		await page.getByRole('button', { name: 'Speichern' }).click();
		const frage = page.getByText('Bestand von 3 auf 1 verringern?');
		await expect(frage).toBeVisible();

		await sperre(page);
		await expect(frage).toBeHidden();
		expect(await gezeigterText(page)).not.toContain('verringern');

		await schliesseAuf(page);
		await expect(frage).toBeVisible();
		await page
			.getByRole('dialog')
			.filter({ hasText: 'verringern?' })
			.getByRole('button', { name: 'Abbrechen' })
			.click();
		await expect(page.locator('#buch-bestand')).toHaveValue('1');
		expect(
			querySQL(
				`SELECT count(*) FROM buecher_exemplare e JOIN buecher_titel t ON t.id = e.titel_id WHERE t.isbn = '${ISBN}' AND NOT e.ist_ausgesondert`
			)
		).toBe('3');
	});
});
