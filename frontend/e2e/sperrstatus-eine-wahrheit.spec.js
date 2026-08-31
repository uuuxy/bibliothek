import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, seedSQL, uniqueSuffix } from './helpers.js';

// Fund 31.08.2026 (Peter, flasch3): Ein Demo-Schüler stand als „Gesperrt" in Liste
// und Profil, der Knopf bot trotzdem „Schüler sperren" an; nach Sperren+Entsperren
// zeigte der Bildschirm „Aktiv", nach einem Reload wieder „Gesperrt".
//
// Dahinter lagen DREI Sperr-Definitionen: Die Liste las nur die Systemsperre
// (ist_gesperrt — ein manuell Gesperrter stand als „Alles ok" da), das Profil
// rechnete nach dem Umschalten eine erfundene Formel (manuell || offene Schäden)
// und überschrieb den Serverwert. Jetzt: EIN Anzeige-Prädikat (sperrStatus.js),
// dasselbe Paar wie an der Theke.
test('Sperrstatus: eine Wahrheit für Liste, Profil und Umschalter', async ({ page }) => {
	await uiLogin(page);
	const suffix = uniqueSuffix();

	// Schüler 1: regulär MANUELL gesperrt (der Weg, den es in der App gibt).
	const s1 = await apiPost(page, '/api/schueler', {
		geburtsdatum: '2012-02-02',
		vorname: 'E2E',
		nachname: `Handschloss-${suffix}`,
		klasse: '8A',
		barcode_id: `SPERR1-${suffix}`
	});
	expect(s1.ok()).toBeTruthy();
	const { id: id1 } = await s1.json();
	seedSQL(
		`UPDATE schueler SET is_manually_blocked = true, block_reason = 'E2E Testsperre' WHERE id = '${id1}';`
	);

	// Schüler 2: SYSTEM-Sperre wie der Demo-Seed auf flasch3 sie hinterließ —
	// ein Zustand, den die App selbst nicht erzeugt, den die Anzeige aber trotzdem
	// wahrheitsgemäß zeigen muss (die Theke blockt ihn ja auch).
	const s2 = await apiPost(page, '/api/schueler', {
		geburtsdatum: '2012-03-03',
		vorname: 'E2E',
		nachname: `Systemsperre-${suffix}`,
		klasse: '8A',
		barcode_id: `SPERR2-${suffix}`
	});
	expect(s2.ok()).toBeTruthy();
	const { id: id2 } = await s2.json();
	seedSQL(
		`UPDATE schueler SET ist_gesperrt = true, block_reason = 'E2E Altlast' WHERE id = '${id2}';`
	);

	// 1) Die LISTE nennt beide „Gesperrt" — vorher stand der manuell Gesperrte
	//    als „Alles ok" da (Liste las nur ist_gesperrt).
	await page.getByTitle('Schülerdatei').click();
	const suche = page.getByPlaceholder(/Name, Klasse oder Barcode/);
	await suche.fill(`Handschloss-${suffix}`);
	// Die Zeile trägt role="button" (klickbare Zeile) — daher tr-Locator statt Rolle.
	const zeile1 = page.locator('tr').filter({ hasText: `Handschloss-${suffix}` });
	await expect(zeile1.getByText('Gesperrt')).toBeVisible();

	await suche.fill(`Systemsperre-${suffix}`);
	const zeile2 = page.locator('tr').filter({ hasText: `Systemsperre-${suffix}` });
	await expect(zeile2.getByText('Gesperrt')).toBeVisible();

	// 2) Profil des System-Gesperrten: Sperren+Entsperren (Handschloss) darf die
	//    Anzeige NICHT auf „Aktiv" kippen — die Systemsperre besteht weiter.
	//    Vorher log die erfundene Formel genau hier.
	await zeile2.click();
	await expect(page.getByText('Konto-Status')).toBeVisible();
	await expect(page.getByText('Gesperrt', { exact: true }).first()).toBeVisible();

	await page.getByRole('button', { name: 'Schüler sperren' }).click();
	await page.getByLabel(/Grund der Sperre/).fill('E2E Zusatzsperre');
	await page.getByRole('button', { name: 'Bestätigen' }).click();
	await page.getByRole('button', { name: 'Sperre aufheben' }).click();
	await page.getByRole('button', { name: 'Bestätigen' }).click();
	await expect(page.getByRole('button', { name: 'Schüler sperren' })).toBeVisible();

	// Die Anzeige bleibt bei der Wahrheit der Datenbank:
	await expect(page.getByText('Gesperrt', { exact: true }).first()).toBeVisible();
});
