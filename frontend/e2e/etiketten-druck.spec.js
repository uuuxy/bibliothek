import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, csrfToken, uniqueSuffix } from './helpers.js';

// Etikettendruck — die Brücke zur physischen Welt: ohne funktionierende
// Barcode-Bögen kann das Sekretariat keine neuen Bücher auszeichnen.
// Beide Server-Pfade als PDF-Smoke: der Titel-Bogen aus dem Buchformular
// ("Barcodes drucken") und der freie Druck über /api/print/labels.

test('Etiketten: Titel-Bogen und freier Labeldruck liefern echte PDFs', async ({ page }) => {
	const s = uniqueSuffix();

	seedSQL(`
        WITH t AS (
            INSERT INTO buecher_titel (isbn, titel, autor)
            VALUES ('978e${s}', 'E2E Etikettenbuch ${s}', 'Druck Autor')
            RETURNING id
        )
        INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
        SELECT id, b, true FROM t, unnest(ARRAY['B-eti1-${s}', 'B-eti2-${s}']) AS b;
    `);
	const titelId = querySQL(`SELECT id FROM buecher_titel WHERE isbn = '978e${s}'`);

	await uiLogin(page);

	// 1. Titel-Etikettenbogen (Button "Barcodes drucken" im Buchformular)
	const bogen = await page.request.get(`/api/buecher/titel/${titelId}/etiketten`);
	expect(bogen.status()).toBe(200);
	expect(bogen.headers()['content-type']).toContain('application/pdf');
	expect((await bogen.body()).length).toBeGreaterThan(1000);

	// 2. Freier Labeldruck (A4 Zweckform, wie ihn die Etiketten-Verwaltung nutzt)
	const token = await csrfToken(page);
	const labels = await page.request.post('/api/print/labels', {
		headers: { 'X-CSRF-Token': token },
		data: {
			formatId: 'zweckform_l4760',
			startPosition: 0,
			isQR: false,
			items: [
				{
					BarcodeID: `B-eti1-${s}`,
					Titel: `E2E Etikettenbuch ${s}`,
					Autor: 'Druck Autor',
					ISBN: `978e${s}`
				},
				{
					BarcodeID: `B-eti2-${s}`,
					Titel: `E2E Etikettenbuch ${s}`,
					Autor: 'Druck Autor',
					ISBN: `978e${s}`
				}
			]
		}
	});
	expect(labels.status()).toBe(200);
	expect(labels.headers()['content-type']).toContain('application/pdf');
	expect((await labels.body()).length).toBeGreaterThan(1000);
});

// Regression: Der Druck-Vorschlag nach dem Wareneingang führte auf eine WEISSE SEITE.
// Ursache: App.svelte setzte beim gefüllten printQueue activeTab='labels' — diesen Tab
// kennt der Router nicht (der App-Route-Name ist 'druck-center'; 'labels' ist nur der
// interne Unter-Tab in DruckCenter), also rendert <main> nichts. Der bisherige Etiketten-
// Test deckte nur die Server-PDF-Endpunkte ab, nie diesen UI-Handoff.
test('Wareneingang → Druck-Vorschlag öffnet den Etikettendruck (keine weiße Seite)', async ({
	page
}) => {
	const s = uniqueSuffix();

	// Ein Exemplar „Im Zulauf" ohne gedrucktes Etikett (Amazon-Fall) — genau das löst
	// nach dem Einbuchen den Druck-Vorschlag aus (etikett_gedruckt = false).
	seedSQL(`
        WITH t AS (
            INSERT INTO buecher_titel (isbn, titel, autor)
            VALUES ('978z${s}', 'E2E Zulaufbuch ${s}', 'Zulauf Autor')
            RETURNING id
        )
        INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, etikett_gedruckt, zustand_notiz)
        SELECT id, 'B-zul-${s}', false, false, 'Im Zulauf (E2E-Lieferant ${s})' FROM t;
    `);

	await uiLogin(page);
	await page.getByTitle('Bestellungen').click();

	// Zulauf-Streifen → Wareneingang öffnen
	await page.getByRole('button', { name: 'Einbuchen' }).click();

	// Alles auswählen und einbuchen
	await page.getByRole('button', { name: 'Alle auswählen' }).click();
	await page.getByRole('button', { name: 'Ausgewählte Positionen einbuchen' }).click();

	// Der Druck-Vorschlag erscheint (Etikett war nicht gedruckt) → drucken
	const printBtn = page.getByRole('button', { name: /Etiketten für diese Lieferung drucken/ });
	await expect(printBtn).toBeVisible();
	await printBtn.click();

	// FIX-BEWEIS: Diese Meldung stammt aus dem Etikettendruck (LabelSettings) und erscheint
	// NUR, wenn printQueue.copies dort ankommt. Vor dem Fix war die Seite leer, der Text
	// existierte nie. Er beweist beide Hälften: Navigation UND Queue-Übergabe.
	await expect(page.getByText(/Etiketten aus der freigegebenen Lieferung geladen/)).toBeVisible();
});
