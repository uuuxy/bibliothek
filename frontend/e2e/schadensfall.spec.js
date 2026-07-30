import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, seedSQL, uniqueSuffix } from './helpers.js';

// Schadensfall: Verlust melden beendet die Ausleihe, erzeugt den Elternbrief
// (PDF-Popup) und macht die offene Forderung im Profil sichtbar —
// inkl. Rechnung-PDF-Smoke über den offenen Betrag.
test('Schadensfall: melden beendet Ausleihe und öffnet Forderung', async ({ page }) => {
	await uiLogin(page);
	const suffix = uniqueSuffix();

	const created = await apiPost(page, '/api/schueler', {
		vorname: 'E2E',
		nachname: `Schaden-${suffix}`,
		klasse: '9R',
		barcode_id: `S-${suffix}`
	});
	expect(created.ok(), `Schüler-Seeding: ${created.status()}`).toBeTruthy();
	const { id: studentId } = await created.json();

	seedSQL(`
        WITH t AS (
            INSERT INTO buecher_titel (titel) VALUES ('E2E-Schadenbuch-${suffix}') RETURNING id
        ), e AS (
            INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
            SELECT id, 'B-${suffix}', true FROM t RETURNING id
        )
        INSERT INTO ausleihen (exemplar_id, schueler_id, bearbeiter_id, ausgeliehen_am, rueckgabe_frist)
        SELECT e.id, '${studentId}', (SELECT id FROM benutzer ORDER BY erstellt_am LIMIT 1), NOW(), NOW() + INTERVAL '14 days' FROM e;
    `);

	// Konto öffnen, entliehenes Buch sichtbar
	await page.getByTitle('Ausleihe').click();
	const scanInput = page.getByPlaceholder(/scannen/i).first();
	await scanInput.fill(`S-${suffix}`);
	await scanInput.press('Enter');
	await expect(page.getByText(`E2E-Schadenbuch-${suffix}`).first()).toBeVisible();

	// Schaden melden → Modal ausfüllen → Elternbrief öffnet als PDF-Popup
	// Am zugänglichen Namen festmachen, nicht am title-Attribut: Der Locator hing an
	// „Verlust/Schaden melden" und fiel um, als title und aria-label vereinheitlicht
	// wurden. getByRole prüft, was Nutzer und Screenreader tatsächlich adressieren.
	await page.getByRole('button', { name: 'Verlust oder Schaden melden' }).first().click();
	await page.locator('#damage-reason').fill('E2E Wasserschaden');
	await page.locator('#damage-amount').fill('12.50');
	const popupPromise = page.waitForEvent('popup');
	await page.getByRole('button', { name: 'Melden & PDF generieren' }).click();
	const popup = await popupPromise;

	// popup.url() sofort auszulesen war der Grund, warum dieser Test unter Last
	// gelegentlich umfiel: Das popup-Ereignis feuert, sobald das Fenster existiert —
	// die Adresse steht erst, wenn die Navigation übernommen hat. Bis dahin liefert
	// url() "about:blank". Allein lief der Test deshalb immer durch, im vollen Lauf
	// (Backend erzeugt gleichzeitig das PDF) reichte die Zeit manchmal nicht.
	await expect
		.poll(() => popup.url(), { message: 'Elternbrief-Popup öffnet das PDF' })
		.toContain('/pdf');
	const elternbriefURL = popup.url();
	await popup.close();

	// Ein geöffnetes Fenster ist noch kein Elternbrief — die Route muss auch liefern.
	const elternbrief = await page.request.get(elternbriefURL);
	expect(elternbrief.status(), 'Elternbrief-PDF').toBe(200);
	expect(elternbrief.headers()['content-type']).toContain('application/pdf');

	// Die Ausleihe ist beendet — das Buch verschwindet aus der Liste
	await expect(page.getByText(`E2E-Schadenbuch-${suffix}`)).not.toBeVisible();

	// Offene Forderung: Rechnung-PDF-Smoke über den ungezahlten Betrag
	const pdf = await page.request.get(`/api/print/rechnung/${studentId}`);
	expect(pdf.status(), 'Rechnung-PDF Status').toBe(200);
	expect(pdf.headers()['content-type']).toContain('application/pdf');
	expect((await pdf.body()).length).toBeGreaterThan(1000);
});
