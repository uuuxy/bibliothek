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

	// Geprüft wird die ANTWORT, die im Fenster ankommt, nicht dessen Adresse.
	//
	// Vorher stand hier eine Zusicherung auf popup.url(). Die war grün, während die
	// Funktion kaputt war: Der Service Worker beantwortete die Navigation aus dem Cache
	// mit der App-Shell — der Benutzer sah die Ausleihe statt des Elternbriefs, aber die
	// Adresse des Fensters lautete weiterhin auf /pdf. Der begleitende page.request.get
	// deckte es nicht auf, weil ein fetch am Service Worker vorbeiging.
	/** @type {{ url: string, typ: string }[]} */
	const antworten = [];
	page.context().on('response', (r) => {
		if (r.url().includes('/api/schadensfaelle/') && r.url().endsWith('/pdf')) {
			antworten.push({ url: r.url(), typ: r.headers()['content-type'] ?? '' });
		}
	});

	const popupPromise = page.waitForEvent('popup');
	await page.getByRole('button', { name: 'Melden & PDF generieren' }).click();
	const popup = await popupPromise;

	await expect
		.poll(() => antworten.length, { message: 'Elternbrief-Antwort trifft ein' })
		.toBeGreaterThan(0);
	expect(antworten[0].typ, 'Elternbrief ist ein PDF').toContain('application/pdf');
	const elternbriefURL = antworten[0].url;
	await popup.close();

	// Und die Route liefert auch beim direkten Abruf.
	const elternbrief = await page.request.get(elternbriefURL);
	expect(elternbrief.status(), 'Elternbrief-PDF').toBe(200);
	expect(elternbrief.headers()['content-type']).toContain('application/pdf');

	// Die Ausleihe ist beendet — das Buch verschwindet aus der AUSLEIH-Liste.
	// Bewusst auf die Karte gescopet (Kartenwurzel = div, dessen Kopfzeilen-div das h3
	// trägt): Seit der Gebühren-Sektion (16.08.2026) taucht derselbe Titel rechtmäßig
	// in der Forderungsliste wieder auf — ein seitenweites not.toBeVisible würde die
	// neue Wahrheit als Fehler melden.
	const ausleihKarte = page.locator('div:has(> div > h3:text-matches("Entliehene Bücher"))');
	// Wache gegen leer-grünes Scoping: Erst belegen, dass der Karten-Locator überhaupt
	// etwas findet — sonst wäre das not.toBeVisible darunter vakuum-grün.
	await expect(ausleihKarte).toBeVisible();
	await expect(ausleihKarte.getByText(`E2E-Schadenbuch-${suffix}`)).not.toBeVisible();

	// ... und steht als offene Forderung in der Gebühren-Sektion.
	const gebuehrenKarte = page.locator('div:has(> div > h3:text-matches("Gebühren & Schäden"))');
	await expect(gebuehrenKarte.getByText(`E2E-Schadenbuch-${suffix}`)).toBeVisible();
	await expect(gebuehrenKarte.getByText('offen', { exact: true })).toBeVisible();

	// Offene Forderung: Rechnung-PDF-Smoke über den ungezahlten Betrag
	const pdf = await page.request.get(`/api/print/rechnung/${studentId}`);
	expect(pdf.status(), 'Rechnung-PDF Status').toBe(200);
	expect(pdf.headers()['content-type']).toContain('application/pdf');
	expect((await pdf.body()).length).toBeGreaterThan(1000);

	// Storno-Weg: ohne Grund gesperrt, mit Grund wird die Forderung erlassen.
	await gebuehrenKarte.getByRole('button', { name: 'Stornieren' }).click();
	const stornoModal = page.locator('div:has(> h3:text-is("Gebühr wirklich stornieren?"))');
	await expect(stornoModal.getByRole('button', { name: 'Stornieren' })).toBeDisabled();
	await stornoModal.locator('#storno-grund').fill('E2E: Buch wiedergefunden');
	await stornoModal.getByRole('button', { name: 'Stornieren' }).click();

	// Nach dem Neuladen des Profils trägt der Fall das Storno-Kennzeichen samt Grund,
	// und die Aktionsknöpfe sind weg — der Fall ist erledigt.
	await expect(gebuehrenKarte.getByText('storniert', { exact: true })).toBeVisible();
	await expect(gebuehrenKarte.getByText('Grund: E2E: Buch wiedergefunden')).toBeVisible();
	await expect(gebuehrenKarte.getByRole('button', { name: 'Bezahlt' })).not.toBeVisible();
});
