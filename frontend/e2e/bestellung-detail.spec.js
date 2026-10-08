// Gate für die Detailansicht einer Bestellung.
//
// Rückmeldung zur alten aufklappenden Zeile: „geht das Feld nach unten aber nicht mit
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
// Der Bestellweg verschickt nach dem Speichern die Bestellmail, und der lokale Stack trägt
// den Mailserver aus der .env. Der Lieferant hat hier deshalb eine Adresse, die keine ist:
// Der Versand scheitert an ihr, bevor eine Verbindung entsteht (api/mail_sender.go,
// TestSendEmail_InvalidRecipient), und der Handler antwortet trotzdem mit 200.
import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, csrfToken, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

const KEINE_ADRESSE = 'keine Adresse';
const MENGE = 3;

test('Bestellung öffnen zeigt Positionen und die gelieferten Exemplarnummern', async ({ page }) => {
	// Eigener Titel statt des ersten im Katalog: Die Bestellung legt drei Exemplare im
	// Zulauf an, und die hingen sonst an einem fremden Titel. Das Aufräumen unten löscht
	// den eigenen Titel, und seine Exemplare fallen mit ihm (ON DELETE CASCADE).
	const s = uniqueSuffix();
	const LIEFERANT = `E2E-Detail-Haendler ${s}`;
	seedSQL(`INSERT INTO buecher_titel (titel, isbn) VALUES ('E2E-Detail-Titel ${s}', '97d${s}');`);
	const titel = {
		id: querySQL(`SELECT id FROM buecher_titel WHERE isbn = '97d${s}'`),
		title: `E2E-Detail-Titel ${s}`
	};
	expect(titel.id, 'eigener Titel angelegt').toBeTruthy();

	await uiLogin(page);

	// --- Eine echte Bestellung über den echten Weg aufgeben ---------------------
	// Nur so entstehen Exemplare MIT bestellung_id; ein INSERT von Hand prüfte die
	// Ansicht gegen Daten, die der Produktivpfad so nie erzeugt.
	const lieferantRes = await apiPost(page, '/api/lieferanten', {
		name: LIEFERANT,
		email: KEINE_ADRESSE,
		customerNumber: 'K-DETAIL'
	});
	expect(lieferantRes.ok(), `Lieferant anlegen: ${await lieferantRes.text()}`).toBeTruthy();
	const lieferantId = (await lieferantRes.json()).id;

	const bestellRes = await apiPost(page, '/api/bestellungen', {
		supplier_id: lieferantId,
		mittel: 'land',
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
		// Die Bestellung zuerst: buecher_exemplare.bestellung_id steht auf ON DELETE SET
		// NULL, ihre drei Exemplare blieben sonst als Zulauf ohne Bestellung stehen. Sie
		// fallen mit dem eigenen Titel.
		seedSQL(`
			DELETE FROM bestellungen_verlauf WHERE lieferant_name = 'E2E-Detail-Haendler ${s}';
			DELETE FROM buecher_titel WHERE isbn = '97d${s}';
		`);
	}
});
