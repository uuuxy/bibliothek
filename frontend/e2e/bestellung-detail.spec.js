// Gate für die Detailansicht einer Bestellung.
//
// Peter über die alte aufklappende Zeile: „geht das Feld nach unten aber nicht mit
// merklichem Mehrwert". Sie zeigte dieselben Angaben wie die Tabellenzeile darüber, nur
// untereinander. Was fehlte, waren Cover und Exemplarnummern — und die Exemplarnummern
// waren bis Migration 063 gar nicht zuzuordnen (buecher_exemplare.bestellung_id).
//
// Warum E2E: Der Zugewinn hängt an einer Kette aus vier Gliedern — bestellen → Exemplare
// mit bestellung_id → Endpunkt → Ansicht. bestelldetail_pg_test.go beweist Glied 3, aber
// nicht, dass die Nummer beim Benutzer ankommt, und schon gar nicht, dass der ECHTE
// Bestellweg die Verknüpfung überhaupt schreibt. Genau diese Lücke hat dieses Projekt
// zweimal getroffen: eine Funktion, isoliert grün, über den Live-Pfad nie erreicht.
//
// Der Mailversand ist hier unbedenklich: Der Handler speichert die Bestellung ZUERST und
// antwortet auch dann mit 200, wenn der Versand scheitert — und der lokale Stack zeigt
// mit 127.0.0.1:1 ins Leere. Es verlässt nichts diesen Rechner.
import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, csrfToken } from './helpers.js';

const LIEFERANT = 'E2E-Detail-Haendler';
const MENGE = 3;

test('Bestellung öffnen zeigt Positionen und die gelieferten Exemplarnummern', async ({ page }) => {
	await uiLogin(page);

	// --- Eine echte Bestellung über den echten Weg aufgeben ---------------------
	// Nur so entstehen Exemplare MIT bestellung_id; ein INSERT von Hand prüfte die
	// Ansicht gegen Daten, die der Produktivpfad so nie erzeugt.
	const lieferantRes = await apiPost(page, '/api/lieferanten', {
		name: LIEFERANT,
		email: 'detail@example.invalid',
		customerNumber: 'K-DETAIL'
	});
	expect(lieferantRes.ok(), `Lieferant anlegen: ${await lieferantRes.text()}`).toBeTruthy();
	const lieferantId = (await lieferantRes.json()).id;

	const buecher = await (await page.request.get('/api/books')).json();
	const titel = (buecher.data ?? [])[0];
	expect(titel, 'Testdaten: mindestens ein Titel nötig').toBeTruthy();

	const bestellRes = await apiPost(page, '/api/bestellungen', {
		supplier_id: lieferantId,
		items: [{ titel_id: titel.id, menge: MENGE, preis: 12.5, generate_barcodes: true }]
	});
	expect(bestellRes.ok(), `Bestellung aufgeben: ${await bestellRes.text()}`).toBeTruthy();

	try {
		// Die Bestell-Antwort trägt keine ID — über die Historie suchen, so wie ein
		// Benutzer es auch täte.
		const historie = await (await page.request.get('/api/bestellhistorie')).json();
		const meine = historie.find((/** @type {any} */ b) => b.lieferant_name === LIEFERANT);
		expect(meine, 'Bestellung steht in der Historie').toBeTruthy();

		const detail = await (await page.request.get(`/api/bestellhistorie/${meine.id}`)).json();
		const barcodes = detail.exemplare.map((/** @type {any} */ e) => e.barcode_id);
		expect(barcodes.length, 'Der Bestellweg verknüpft die Exemplare mit der Bestellung').toBe(
			MENGE
		);

		// --- Über die Oberfläche hineinklicken -----------------------------------
		await page.goto('/bestellungen');
		await page.getByRole('tab', { name: 'Bestellhistorie' }).click();

		await page
			.getByRole('button', { name: new RegExp(`bei ${LIEFERANT} öffnen`) })
			.first()
			.click();

		// Der Kopf identifiziert die Bestellung …
		await expect(page.getByRole('heading', { name: LIEFERANT })).toBeVisible();
		// … die Position nennt den bestellten Titel …
		await expect(page.getByText(titel.title, { exact: false }).first()).toBeVisible();

		// … und darunter stehen DIE NUMMERN. Das ist der Punkt der ganzen Ansicht.
		const abschnitt = page.getByRole('heading', { name: /Exemplare aus dieser Bestellung/ });
		await expect(abschnitt).toBeVisible();
		await expect(abschnitt).toContainText(`(${MENGE})`);
		for (const barcode of barcodes) {
			await expect(page.getByText(barcode, { exact: true })).toBeVisible();
		}

		// --- Zurück führt zur Liste ----------------------------------------------
		await page.getByRole('button', { name: 'Zurück zur Bestellhistorie' }).click();
		await expect(page.getByRole('heading', { name: 'Bestellhistorie' })).toBeVisible();
	} finally {
		const token = await csrfToken(page);
		await page.request.delete(`/api/lieferanten/${lieferantId}`, {
			headers: { 'X-CSRF-Token': token }
		});
	}
});
