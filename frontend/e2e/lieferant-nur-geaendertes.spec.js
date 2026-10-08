import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix, einstellungsKategorie } from './helpers.js';

// Die Zeile der Lieferantentabelle füllt ihre Maske aus der Liste, und die lädt beim Öffnen der
// Seite. Macht ein anderer Platz inzwischen einen anderen Händler zum Hauptlieferanten, nahm das
// Korrigieren einer E-Mail ihm das Merkmal wieder: Die Maske schickte ihren alten Stand mit.
// Sie schickt nur, was sie seit dem Öffnen geändert hat. Den Hauptlieferanten von vor dem Lauf
// stellt der Teardown zurück (global-teardown.js); die Namen beginnen deshalb mit ZZZ.
test('Lieferant bearbeiten: das Speichern nimmt einem anderen Händler nicht das Merkmal Hauptlieferant', async ({
	page
}) => {
	const s = uniqueSuffix();
	const alt = `ZZZ-NG-Alt-${s}`;
	const neu = `ZZZ-NG-Neu-${s}`;
	seedSQL(`
		UPDATE lieferanten SET ist_hauptlieferant = false WHERE ist_hauptlieferant;
		INSERT INTO lieferanten (name, email, kundennummer, ist_hauptlieferant)
		VALUES ('${alt}', 'alt${s}@example.invalid', 'K-A-${s}', true),
		       ('${neu}', 'neu${s}@example.invalid', 'K-N-${s}', false);
	`);
	const stand = () =>
		querySQL(
			`SELECT string_agg(name || '|' || email || '|' || ist_hauptlieferant, ' ; ' ORDER BY name) FROM lieferanten WHERE name IN ('${alt}', '${neu}')`
		).trim();

	await uiLogin(page);
	await page.goto('/einstellungen');
	await einstellungsKategorie(page, 'Lieferanten').click();
	await page.getByRole('heading', { name: 'Neuer Lieferant' }).waitFor();
	await page.locator('tr', { hasText: alt }).getByRole('button', { name: 'Bearbeiten' }).click();
	const maske = page.locator('tr', { hasText: /Speichern/ });
	const mail = maske.locator('input[type="email"]');
	await expect(mail).toHaveValue(`alt${s}@example.invalid`);

	// Ein anderer Platz macht den zweiten Händler zum Hauptlieferanten, während die Maske offen ist.
	seedSQL(`
		UPDATE lieferanten SET ist_hauptlieferant = false WHERE ist_hauptlieferant;
		UPDATE lieferanten SET ist_hauptlieferant = true WHERE name = '${neu}';
	`);

	const gesendet = page.waitForRequest(
		(r) => r.method() === 'PUT' && /\/api\/lieferanten\//.test(r.url())
	);
	await mail.fill(`korrigiert${s}@example.invalid`);
	await maske.getByRole('button', { name: 'Speichern' }).click();
	expect((await gesendet).postDataJSON(), 'Die Maske schickt mehr als das geänderte Feld').toEqual({
		email: `korrigiert${s}@example.invalid`
	});
	await expect(page.getByText('Lieferant aktualisiert.')).toBeVisible();

	expect(stand(), 'Das Korrigieren der E-Mail hat das Merkmal Hauptlieferant zurückgeholt').toBe(
		`${alt}|korrigiert${s}@example.invalid|false ; ${neu}|neu${s}@example.invalid|true`
	);
	// Die neu geladene Liste zeigt den Stand des Servers.
	await expect(
		page.locator('tr', { hasText: neu }).getByText('Hauptlieferant', { exact: true })
	).toBeVisible();
});
