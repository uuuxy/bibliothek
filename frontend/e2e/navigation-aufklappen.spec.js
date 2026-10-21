// Die eingeklappte Navigation (64 px) klappt wie bei Gmail auf, solange die Maus über ihr steht,
// und liegt dabei über dem Inhalt. Gemessen wird, was nur der Browser zeigt: Kein Symbol wechselt
// seinen Platz, der Inhalt rückt nicht, die Fläche liegt oben und trägt Erhebung statt Rahmen
// (e2e/m3-bauform.spec.js). Nach der Wahl eines Ziels ist der Inhalt wieder frei.
import { test, expect } from '@playwright/test';
import { uiLogin, gehZu, menuepunkt } from './helpers.js';

/** Lage jedes Navigationsziels: Oberkante und Mitte des Symbols. Läuft im Browser. */
const LAGE = () =>
	[...document.querySelectorAll('aside nav button')]
		.filter((b) => b.querySelector('svg'))
		.map((b) => {
			const symbol = /** @type {SVGElement} */ (b.querySelector('svg')).getBoundingClientRect();
			return `${Math.round(b.getBoundingClientRect().y)}/${Math.round(symbol.x + symbol.width / 2)}`;
		});

test('Eingeklappte Navigation: klappt unter der Maus über dem Inhalt auf, ohne dass etwas rückt', async ({
	page
}) => {
	// Unter 1280 px beginnt die Navigation eingeklappt.
	await page.setViewportSize({ width: 1100, height: 900 });
	await uiLogin(page);
	await gehZu(page, '/schuelerdatei');
	const leiste = page.locator('aside', { has: page.locator('nav') });
	const suche = page.getByRole('searchbox', { name: 'Leser suchen' });
	await expect(leiste.getByRole('button', { name: 'Navigation ausklappen' })).toBeVisible();
	await expect(leiste.getByText('Mahnwesen', { exact: true })).toHaveCount(0);

	const vorher = await page.evaluate(LAGE);
	const sucheVorher = await suche.boundingBox();
	expect(vorher.length, 'keine Navigationsziele gefunden').toBeGreaterThan(10);

	const ziel = await menuepunkt(page, 'Mahnwesen').boundingBox();
	if (!ziel) throw new Error('Mahnwesen hat keine Fläche');
	await page.mouse.move(ziel.x + ziel.width / 2, ziel.y + ziel.height / 2);
	const name = leiste.getByText('Mahnwesen', { exact: true });
	await expect(name).toBeVisible();
	// Die Breite wächst über 300 ms; gemessen wird danach.
	await expect.poll(async () => (await leiste.locator('> div').boundingBox())?.width).toBe(256);

	expect(await page.evaluate(LAGE), 'Symbole nach dem Aufklappen').toEqual(vorher);
	expect(await suche.boundingBox(), 'der Inhalt darf nicht rücken').toEqual(sucheVorher);
	expect((await leiste.boundingBox())?.width, 'Platz der Leiste im Seitenaufbau').toBe(64);

	const flaeche = await leiste.locator('> div').evaluate((e) => {
		const stil = getComputedStyle(e);
		const r = e.getBoundingClientRect();
		const oben = document.elementFromPoint(r.right - 20, r.y + r.height / 2);
		return {
			rahmen: stil.borderRightWidth,
			schatten: stil.boxShadow,
			obenAuf: !!oben && e.contains(oben)
		};
	});
	expect(flaeche.rahmen, 'über dem Inhalt kein Rahmen (M3: Rahmen oder Erhebung)').toBe('0px');
	expect(flaeche.schatten, 'über dem Inhalt mit Erhebung').not.toBe('none');
	expect(flaeche.obenAuf, 'die Fläche liegt über dem Inhalt').toBe(true);

	await name.click();
	await expect(page).toHaveURL(/\/mahnwesen$/);
	await expect(name, 'nach der Wahl wieder eingeklappt').toHaveCount(0);

	// Weg und wieder hin: Sie klappt erneut auf.
	await page.mouse.move(700, 450);
	await page.mouse.move(ziel.x + ziel.width / 2, ziel.y + ziel.height / 2);
	await expect(leiste.getByText('Mahnwesen', { exact: true })).toBeVisible();
	await page.mouse.move(700, 450);
	await expect(leiste.getByText('Mahnwesen', { exact: true })).toHaveCount(0);
});

// Ein Dialog liegt über der Navigation, eingeklappt wie ausgeklappt: Über den Inhalt hebt sich
// die Leiste nur, solange sie unter der Maus aufgeklappt ist. Viele Dialoge stehen im Inhalt
// in einem eigenen Stapelkontext; läge die Leiste immer oben, verdeckte sie deren linken Rand.
for (const breite of [1100, 1710]) {
	test(`Ein Dialog liegt über der Navigation (${breite} px)`, async ({ page }) => {
		await page.setViewportSize({ width: breite, height: 900 });
		await uiLogin(page);
		await gehZu(page, '/schulklassen');
		await page.getByRole('button', { name: 'Klasse hinzufügen' }).click();
		await expect(page.locator('#book-search-field')).toBeVisible();
		const leisteOben = await page.evaluate(() => {
			const leiste = document.querySelector('aside:has(nav)');
			const punkt = document.elementFromPoint(32, 400);
			return !!leiste && !!punkt && leiste.contains(punkt);
		});
		expect(leisteOben, 'die Navigation liegt über dem Schleier des Dialogs').toBe(false);
	});
}
