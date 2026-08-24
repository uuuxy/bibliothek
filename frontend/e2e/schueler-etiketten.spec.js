import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix, csrfToken } from './helpers.js';

// Schüler-Etiketten (Betreiber-Entscheidung 24.08.2026): Der A4-Kartenbogen ist
// abgeschafft, an seiner Stelle steht ein Klebebogen mit Name, Klasse und Barcode.
//
// Warum das hier und nicht nur als Unit-Test steht: Die Kette hat vier Glieder, die
// jeweils woanders liegen — der Umschalter im Ausweis-Designer, das zentral
// gespeicherte Design, der Knopf in der Schülerdatei und der PDF-Endpunkt. Jedes
// Glied ist einzeln geprüft; dass sie zusammenhängen, sagt nur dieser Durchlauf.
// Genau an so einer Stelle stand der Schalter monatelang ins Leere: Er schrieb in
// lokalen Zustand, während die Schülerdatei den Store las.
//
// Der Umschalter ändert ZENTRALEN Zustand (Auto-Save nach /api/ausweis-layout). Diese
// Spec räumt ihn deshalb hinterher wieder auf — ein liegengelassenes 'etikett' würde
// den Ausweis-Stapeldruck aller späteren Läufe umstellen, und genau diese Sorte
// Teardown-Schaden hat hier schon einmal den Hauptlieferanten gekostet.
test.describe('Schüler-Etiketten', () => {
	const s = uniqueSuffix().slice(0, 6);

	test.beforeEach(() => {
		seedSQL(`
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
			VALUES ('ETIK-A-${s}', 'Ayse', 'Etikett${s}', '8G2', 2031),
			       ('ETIK-B-${s}', 'Bela',  'Etikett${s}', '8G2', 2031);
		`);
	});

	test.afterEach(() => {
		seedSQL(`
			DELETE FROM schueler WHERE barcode_id LIKE 'ETIK-%-${s}';
			-- Betriebsart zurück auf Kartendrucker, ohne den Rest des Designs anzufassen.
			UPDATE system_einstellungen
			   SET wert = (wert::jsonb || '{"printMode":"card"}'::jsonb)::text
			 WHERE schluessel = 'ausweis_layout' AND wert <> '' AND wert::jsonb ? 'printMode';
		`);
	});

	test('Umschalter im Designer steuert den Stapeldruck der Schülerdatei', async ({ page }) => {
		await uiLogin(page);

		// ── Ausweis-Designer: der Schalter, wo früher „A4-Bogen" stand ──
		await page.getByTitle('Druck-Center').click();
		await page.getByRole('tab', { name: 'Schülerausweise' }).click();

		const kartendrucker = page.getByRole('button', { name: 'Kartendrucker', exact: true });
		const etikettenbogen = page.getByRole('button', { name: 'Etikettenbogen', exact: true });
		await expect(kartendrucker).toBeVisible();
		await expect(etikettenbogen).toBeVisible();
		// Der abgeschaffte Modus darf hier nicht mehr auftauchen.
		await expect(page.getByRole('button', { name: 'A4-Bogen', exact: true })).toHaveCount(0);

		// Im Kartenmodus gibt es kein Bogenformat zu wählen.
		// Select.svelte ist kein natives <select>, sondern button[role=combobox] —
		// getByLabel().toHaveValue() liefe hier auf „Not an input element".
		const format = page.getByRole('combobox', { name: 'Etikettenformat' });
		await expect(format).toHaveCount(0);
		await expect(page.getByRole('button', { name: /Testdruck Vorderseite/ })).toBeVisible();

		await etikettenbogen.click();
		await expect(format).toBeVisible();
		await expect(format).toHaveText(/Zweckform L4760/);

		// Der Testdruck holt DASSELBE PDF wie der echte Bogen, nur mit Muster-Schüler.
		const musterAntwort = page.waitForResponse(
			(r) => r.url().includes('/api/print/schueler-etiketten') && r.request().method() === 'POST'
		);
		await page.getByRole('button', { name: 'Muster-Etikett drucken' }).click();
		const muster = await musterAntwort;
		expect(muster.status()).toBe(200);
		expect(muster.headers()['content-type']).toContain('application/pdf');
		expect(JSON.parse(muster.request().postData() ?? '{}').muster).toBe(true);

		// Der Auto-Save braucht 800 ms Ruhe; ohne ihn läse die Schülerdatei den alten Stand.
		await expect(page.getByText('Zentral gespeichert')).toBeVisible();

		// ── Schülerdatei: derselbe Zustand, anderer Bildschirm ──
		await page.getByTitle('Schülerdatei').click();
		await page.getByLabel('Schüler suchen').fill(`Etikett${s}`);
		await expect(page.locator('tbody tr').filter({ hasText: `Etikett${s}` })).toHaveCount(2);
		await page.getByRole('checkbox', { name: /Alle angezeigten Schüler/ }).check();

		const balken = page.getByRole('region', { name: /Aktionen für die markierten/ });
		await expect(balken.getByRole('button', { name: 'Etiketten drucken' })).toBeVisible();
		await expect(balken.getByRole('spinbutton')).toHaveValue('1');

		// Kein versteckter Kartensatz im DOM: Im Etikettenmodus zöge jede Karte ein
		// Barcode-Bild über die Leitung, das niemand zu sehen bekommt.
		await expect(page.locator('.print-section-card')).toHaveCount(0);

		// Angebrochener Bogen: ab Feld 5 drucken.
		await balken.getByRole('spinbutton').fill('5');

		const bogenAntwort = page.waitForResponse(
			(r) => r.url().includes('/api/print/schueler-etiketten') && r.request().method() === 'POST'
		);
		await balken.getByRole('button', { name: 'Etiketten drucken' }).click();
		const bogen = await bogenAntwort;
		expect(bogen.status()).toBe(200);
		expect(bogen.headers()['content-type']).toContain('application/pdf');

		const gesendet = JSON.parse(bogen.request().postData() ?? '{}');
		expect(gesendet.startPosition).toBe(5);
		expect(gesendet.schuelerIds).toHaveLength(2);
		// Namen dürfen NICHT mitgeschickt werden — die holt der Server selbst.
		expect(JSON.stringify(gesendet)).not.toContain(`Etikett${s}`);

		// Die Bytes der Antwort oben sind nicht mehr lesbar — die Seite hat den Blob
		// bereits verbraucht (window.open). Deshalb derselbe Auftrag noch einmal direkt:
		// Nur so ist belegt, dass wirklich ein PDF herauskommt und nicht eine leere
		// 200-Antwort mit dem richtigen Content-Type.
		const direkt = await page.request.post('/api/print/schueler-etiketten', {
			headers: { 'X-CSRF-Token': await csrfToken(page) },
			data: gesendet
		});
		expect(direkt.status()).toBe(200);
		const bytes = await direkt.body();
		expect(bytes.length).toBeGreaterThan(1000);
		expect(bytes.subarray(0, 5).toString()).toBe('%PDF-');
	});

	// Der sichtbare Einstieg (24.08.2026, auf Peters Ansage): Der Weg über Suche +
	// Kopf-Kästchen funktioniert, ist aber für neue Benutzer unauffindbar. Der Block
	// im Druck-Center druckt bewusst nicht selbst — er landet VOR der Aktionsleiste,
	// damit deren Warnungen auch auf diesem Weg vor dem Stapel stehen.
	test('Druck-Center: „Klassenweise drucken" springt mit fertig markierter Klasse in die Schülerdatei', async ({
		page
	}) => {
		await uiLogin(page);
		await page.getByTitle('Druck-Center').click();
		await page.getByRole('tab', { name: 'Klassenweise drucken' }).click();

		await page.getByRole('combobox', { name: 'Klasse' }).click();
		await page.getByRole('option', { name: '8G2', exact: true }).click();
		await page.getByRole('button', { name: 'Klasse zum Druck markieren' }).click();

		// Gelandet in der Schülerdatei: Suche vorbefüllt, die Aktionsleiste steht ohne
		// weiteres Zutun da, und JEDE angezeigte Zeile trägt den Haken.
		await expect(page.getByLabel('Schüler suchen')).toHaveValue('8G2');
		await expect(page.getByRole('region', { name: /Aktionen für die markierten/ })).toBeVisible();

		const zeilen = page.locator('tbody tr');
		await expect(zeilen.filter({ hasText: `Etikett${s}` })).toHaveCount(2);
		const kaestchen = zeilen.getByRole('checkbox');
		for (const box of await kaestchen.all()) {
			await expect(box).toBeChecked();
		}
	});

	test('Kartenmodus bleibt der Kartenmodus — der Stapel liegt weiter im DOM', async ({ page }) => {
		// Gegenprobe zum Test darüber. Ohne sie belegte er nur, dass IRGENDETWAS anders
		// wird, sobald man den Schalter umlegt — nicht, dass der Kartenweg heil ist.
		await uiLogin(page);
		await page.getByTitle('Schülerdatei').click();
		await page.getByLabel('Schüler suchen').fill(`Etikett${s}`);
		await expect(page.locator('tbody tr').filter({ hasText: `Etikett${s}` })).toHaveCount(2);
		await page.getByRole('checkbox', { name: /Alle angezeigten Schüler/ }).check();

		const balken = page.getByRole('region', { name: /Aktionen für die markierten/ });
		await expect(balken.getByRole('button', { name: 'Ausweise drucken' })).toBeVisible();
		await expect(balken.getByRole('spinbutton')).toHaveCount(0);
		await expect(page.locator('.print-section-card .print-card-box')).toHaveCount(2);
	});
});
