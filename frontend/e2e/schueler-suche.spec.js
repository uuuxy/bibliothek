import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// Die Schülersuche fand Leute nicht — je nachdem, wie ihre Klasse heißt.
//
// Die Schülerdatei filterte im BROWSER über die Liste, die sie geladen hatte, und die ist
// serverseitig bei 500 Zeilen gekappt (sortiert nach Klasse). Bei 875 Schülern waren 375
// über die Suche nicht erreichbar. Für den Benutzer sah das nach Zufall aus: Ein Name ging,
// der nächste nicht, ohne erkennbaren Unterschied.
//
// Der Test legt einen Schüler in einer Klasse an, die garantiert hinter der Kappungsgrenze
// sortiert, und sucht ihn. Ohne Serversuche kann er nicht gefunden werden.
test('Schülerdatei findet auch Schüler hinter der 500er-Grenze', async ({ page }) => {
	const s = uniqueSuffix();
	const nachname = `Zzunsichtbar${s}`;

	seedSQL(`
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ('E2E-SUCH-${s}', 'Testine', '${nachname}', 'ZZZ-${s}', 2030);
	`);

	await uiLogin(page);
	await page.getByTitle('Schülerdatei').click();

	const suchfeld = page.getByLabel('Schüler suchen');
	await suchfeld.click();
	await suchfeld.fill(nachname);

	// BEWEIS: Der Schüler steht in der Liste. Vorher lieferte die Suche hier nichts,
	// weil sein Datensatz gar nicht erst im Browser angekommen war.
	await expect(page.getByText(`Testine ${nachname}`).first()).toBeVisible();
});

// Zweiter Teil derselben Beschwerde: Die Serversuche kann mehr als der frühere
// Browser-Filter (Teilstring über "vorname nachname"). Sie ist dieselbe wie an der Theke.
test('Schülersuche: Reihenfolge und Schreibweise des Namens sind egal', async ({ page }) => {
	const s = uniqueSuffix();

	seedSQL(`
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ('E2E-SUCH2-${s}', 'Jörg', 'Müllermann${s}', '8c', 2030);
	`);

	await uiLogin(page);
	await page.getByTitle('Schülerdatei').click();
	const suchfeld = page.getByLabel('Schüler suchen');
	const treffer = page.getByText(`Jörg Müllermann${s}`).first();

	for (const eingabe of [
		`Müllermann${s}`, // Nachname allein
		`Müllermann${s} Jörg`, // Nachname zuerst — der Browser-Filter verglich "Vorname Nachname"
		`Muellermann${s}`, // deutsche Ersatzschreibung
		`Mullermann${s}` // ohne Umlautpunkte, wie an der Theke getippt
	]) {
		await suchfeld.fill(eingabe);
		await expect(treffer, `Eingabe "${eingabe}" fand den Schüler nicht`).toBeVisible();
		// Zusätzlich die Trefferzahl: Ohne sie bestünde der Test auch dann, wenn die
		// Suche gar nicht filtert und einfach die ganze Liste stehen lässt.
		await expect(
			page.getByText('Treffer: 1'),
			`Eingabe "${eingabe}" hat nicht gefiltert`
		).toBeVisible();
	}
});

// Die Schülerdatei konnte immer schon nach der Klasse suchen. Beim Umzug der Suche auf
// den Server wäre das beinahe stillschweigend verschwunden — die Suche an der Theke
// kennt nur Namen und Barcodes, und beide teilen sich dasselbe Prädikat.
test('Schülerdatei sucht auch nach Klasse — allein und zusammen mit einem Namen', async ({
	page
}) => {
	const s = uniqueSuffix();
	const klasse = `9z${s}`.slice(0, 20);

	seedSQL(`
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr) VALUES
			('E2E-KL1-${s}', 'Anton', 'Klassenkind${s}', '${klasse}', 2030),
			('E2E-KL2-${s}', 'Berta', 'Andersname${s}', '${klasse}', 2030);
	`);

	await uiLogin(page);
	await page.getByTitle('Schülerdatei').click();
	const suchfeld = page.getByLabel('Schüler suchen');

	// Klasse allein: beide Kinder.
	await suchfeld.fill(klasse);
	await expect(page.getByText('Treffer: 2')).toBeVisible();

	// Klasse UND Name: nur das eine. Jedes Token wird einzeln geprüft, deshalb wirkt die
	// Kombination als Und-Verknüpfung und nicht als "alles aus der Klasse".
	await suchfeld.fill(`${klasse} Andersname${s}`);
	await expect(page.getByText('Treffer: 1')).toBeVisible();
	await expect(page.getByText(`Berta Andersname${s}`).first()).toBeVisible();
});

// In der Ausleihe stand der gesuchte Schüler im Vorschlag — aber Enter schickte den rohen
// Text an /api/action, und der Weg dort endet für eine Eingabe ohne Präfix in der
// TITELsuche. Ein getippter Nachname lief deshalb zuverlässig ins Leere.
test('Ausleihe: Enter auf einen eindeutigen Namen öffnet das Schülerkonto', async ({ page }) => {
	const s = uniqueSuffix();
	const nachname = `Eindeutig${s}`;

	seedSQL(`
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ('E2E-OMNI-${s}', 'Paula', '${nachname}', '7a', 2030);
	`);

	await uiLogin(page);

	// Blind tippen statt fill(): Der Kiosk lebt vom Tastaturfokus, und fill() würde ihn
	// implizit setzen und einen Fokusfehler verdecken (siehe kiosk-scannerfokus.spec.js).
	await page.locator('#omnibox-input').click();
	await page.keyboard.type(nachname);

	// Auf den Vorschlag warten (300 ms Entprellung), dann Enter — ohne vorher mit den
	// Pfeiltasten auszuwählen. Genau das war der kaputte Weg.
	await expect(page.getByText(`Paula ${nachname}`).first()).toBeVisible();
	await page.keyboard.press('Enter');

	// BEWEIS: Das Konto ist offen. Vorher erschien hier "nichts gefunden".
	await expect(page.getByRole('heading', { name: `Paula ${nachname}` })).toBeVisible();
});
