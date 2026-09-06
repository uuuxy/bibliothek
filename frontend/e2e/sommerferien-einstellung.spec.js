import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, gehZu, einstellungsKategorie, csrfToken } from './helpers.js';

// Die Sommerferien als Einstellung (Peter, 06.09.2026: „das muss doch dann irgendwo
// eingestellt werden"): Ein Jahr über der Programmtabelle wird in Einstellungen → LUSD &
// Versetzung eingetragen, landet in Normalform in system_einstellungen, und die
// Selbstprüfung (Betriebsbereitschaft) rechnet mit ihm — die Warnung zeigt auf diese
// Einstellung, nicht auf ein Programm-Update. Zum Schluss: Der Server lehnt Unlesbares ab.
const SCHLUESSEL = 'sommerferien';

test.afterEach(() => {
	seedSQL(`DELETE FROM system_einstellungen WHERE schluessel = '${SCHLUESSEL}';`);
});

test('Sommerferien: eigenes Jahr eintragen, gespeichert, in der Selbstprüfung sichtbar', async ({
	page
}) => {
	seedSQL(`DELETE FROM system_einstellungen WHERE schluessel = '${SCHLUESSEL}';`);
	await uiLogin(page);
	await gehZu(page, '/einstellungen');
	await einstellungsKategorie(page, 'LUSD & Versetzung').click();

	const bereich = page.getByTestId('sommerferien');
	await expect(bereich.getByRole('row').filter({ hasText: 'Programm' }).first()).toBeVisible();
	const vorher = await bereich.getByRole('row').count();

	await bereich.getByLabel('Beginn').fill('2031-07-07');
	await bereich.getByLabel('Ende').fill('2031-08-15');
	await bereich.getByRole('button', { name: '2031 aufnehmen' }).click();
	await expect(bereich.getByRole('row')).toHaveCount(vorher + 1);
	await expect(bereich.getByRole('row').filter({ hasText: '2031' })).toContainText(
		'eigener Eintrag'
	);

	await bereich.getByRole('button', { name: 'Sommerferien speichern' }).click();
	await expect(page.getByText('Gespeichert.')).toBeVisible();
	expect(querySQL(`SELECT wert FROM system_einstellungen WHERE schluessel = '${SCHLUESSEL}'`)).toBe(
		'[{"jahr":2031,"von":"2031-07-07","bis":"2031-08-15"}]'
	);

	// Die Selbstprüfung sieht das eingestellte Jahr.
	const lage = await (await page.request.get('/api/admin/system/betriebsbereitschaft')).json();
	const befund = lage.befunde.find((/** @type {any} */ b) => b.bereich === 'Ferientabelle');
	expect(befund?.befund, 'Befund Ferientabelle').toContain('2031');

	// Nach dem Neuladen steht der Eintrag wieder da — und lässt sich entfernen.
	await page.reload();
	await einstellungsKategorie(page, 'LUSD & Versetzung').click();
	await bereich.getByRole('button', { name: 'Sommerferien 2031 entfernen' }).click();
	await bereich.getByRole('button', { name: 'Sommerferien speichern' }).click();
	await expect(page.getByText('Gespeichert.')).toBeVisible();
	expect(querySQL(`SELECT wert FROM system_einstellungen WHERE schluessel = '${SCHLUESSEL}'`)).toBe(
		''
	);

	// Unlesbares lehnt der Server ab — mit dem Grund, nicht mit einer stillen 200.
	const antwort = await page.request.put('/api/einstellungen', {
		data: { sommerferien: '[{"jahr":2031,"von":"2031-08-15","bis":"2031-07-07"}]' },
		headers: { 'X-CSRF-Token': await csrfToken(page) }
	});
	expect(antwort.status(), 'Ende vor Beginn').toBe(400);
	expect((await antwort.json()).error ?? '').toContain('Sommerferien 2031');
});
