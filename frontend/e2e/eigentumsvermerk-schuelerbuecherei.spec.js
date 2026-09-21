import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, gehZu, einstellungsKategorie } from './helpers.js';

// Die zweite Einstellung „Eigentumsvermerk Schülerbücherei" (OFFEN.md 4.21, entschieden am
// 21.09.2026) — der Weg vom Feld bis in die Tabelle. Was der Wert auf dem Etikett bewirkt,
// prüfen die Go-Tests am fertigen PDF (api/etiketten_topf_pg_test.go); hier steht, was nur
// der Browser zeigen kann: Das Feld ist da, es speichert unter SEINEM Schlüssel, und es
// lässt den allgemeinen Vermerk daneben unangetastet.
const SCHLUESSEL = 'etikett_eigentumsvermerk_schuelerbuecherei';
const wert = (schluessel) =>
	querySQL(`SELECT wert FROM system_einstellungen WHERE schluessel = '${schluessel}';`).trim();

// Die Einstellungen sind global und die Suite teilt sich eine Datenbank.
test.afterEach(() => {
	seedSQL(`DELETE FROM system_einstellungen WHERE schluessel = '${SCHLUESSEL}';`);
});

test('Eigentumsvermerk Schülerbücherei: speichert unter eigenem Schlüssel, leer heißt leer', async ({
	page
}) => {
	const allgemeinVorher = wert('etikett_eigentumsvermerk');

	await uiLogin(page);
	await gehZu(page, '/einstellungen');
	await einstellungsKategorie(page, 'Schule').click();

	// Zwei Felder, deren Beschriftung gleich beginnt — beide müssen einzeln greifbar sein.
	await expect(page.getByLabel('Eigentumsvermerk', { exact: true })).toBeVisible();
	const feld = page.getByLabel('Eigentumsvermerk Schülerbücherei', { exact: true });
	await expect(feld).toHaveValue('');

	await feld.fill('Eigentum der Stadt Friedrichsdorf');
	await page.getByRole('button', { name: 'Schule speichern' }).click();
	await expect(page.getByText('Gespeichert.')).toBeVisible();
	expect(wert(SCHLUESSEL)).toBe('Eigentum der Stadt Friedrichsdorf');
	expect(wert('etikett_eigentumsvermerk'), 'der allgemeine Vermerk wurde mitverändert').toBe(
		allgemeinVorher
	);

	// Nach dem Neuladen steht der Wert im Feld — er kam also auch wieder aus dem Server.
	await page.reload();
	await einstellungsKategorie(page, 'Schule').click();
	await expect(page.getByLabel('Eigentumsvermerk Schülerbücherei', { exact: true })).toHaveValue(
		'Eigentum der Stadt Friedrichsdorf'
	);

	// Leer heißt leer: kein Vermerk, keine Vorgabe.
	await page.getByLabel('Eigentumsvermerk Schülerbücherei', { exact: true }).fill('');
	await page.getByRole('button', { name: 'Schule speichern' }).click();
	await expect(page.getByText('Gespeichert.')).toBeVisible();
	expect(wert(SCHLUESSEL)).toBe('');
});
