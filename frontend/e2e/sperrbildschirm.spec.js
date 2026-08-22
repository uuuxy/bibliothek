import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, csrfToken, uniqueSuffix, ADMIN_PASSWORD } from './helpers.js';

/**
 * Inaktivitäts-Wächter am Live-Pfad (docs/datenschutz_offene_punkte.md A4).
 *
 * Zwei Stufen: Nach der kurzen Frist verschwindet der geladene Schüler aus der Theke
 * (der nächste an der Theke darf nicht den vorigen sehen), nach der langen kommt der
 * Sperrbildschirm, der nur mit dem eigenen Passwort wieder aufgeht. Der Store-Test
 * (idleLock.test.js) kennt die Uhr; dieser Test beweist, dass App, Einstellungs-Endpunkt
 * und Sperrbildschirm zusammen funktionieren — mit Playwrights gestellter Uhr, denn die
 * Fristen sind Minuten und kein Test wartet fünf davon.
 *
 * Die Fristen werden NICHT verändert (Vorgaben 5/15 Minuten) — ein Teardown, der die
 * Konfiguration der Anlage anfasst, ist eine eigene Fehlerquelle.
 */
test('Theke leert sich nach 5 Minuten, Sperrbildschirm nach 15, Passwort entsperrt', async ({
	page
}) => {
	await page.clock.install();
	await uiLogin(page);

	const suffix = uniqueSuffix();
	const created = await apiPost(page, '/api/schueler', {
		geburtsdatum: '2012-03-03',
		vorname: 'E2E',
		nachname: `Sperre-${suffix}`,
		klasse: '7A',
		barcode_id: `S-${suffix}`
	});
	expect(created.ok(), `Schüler-Seeding: ${created.status()}`).toBeTruthy();

	// Der Endpunkt, den JEDER angemeldete Client liest: Vorgaben 5 / 15.
	const fristen = await page.request.get('/api/einstellungen/sitzung');
	expect(fristen.ok()).toBeTruthy();
	expect(await fristen.json()).toEqual({ theke_leeren_minuten: 5, sperre_minuten: 15 });

	await expect(page.getByPlaceholder(/scannen/i).first()).toBeVisible();
	await page.keyboard.type(`S-${suffix}`, { delay: 5 });
	await page.keyboard.press('Enter');
	await expect(page.getByText(`Sperre-${suffix}`).first()).toBeVisible();

	// Knapp unter der kurzen Frist: Schüler steht noch.
	await page.clock.fastForward('04:50');
	await expect(page.getByText(`Sperre-${suffix}`).first()).toBeVisible();

	// Kurze Frist vorbei: Theke leer, aber keine Sperre.
	await page.clock.fastForward('00:20');
	await expect(page.getByText(`Sperre-${suffix}`)).toHaveCount(0);
	await expect(page.getByTestId('sperrbildschirm')).toHaveCount(0);

	// Lange Frist vorbei: Sperrbildschirm — und die App ist NICHT mehr im DOM (sonst zeigte
	// Strg+P die Seite dahinter, Tab verließe die Sperre; Prüfung 22.08.2026, A6).
	await page.clock.fastForward('10:00');
	await expect(page.getByTestId('sperrbildschirm')).toBeVisible();
	await expect(page.getByRole('button', { name: 'Abmelden', exact: true })).toHaveCount(0);
	await expect(page.getByPlaceholder(/scannen/i)).toHaveCount(0);

	// Bedienung im gesperrten Zustand entsperrt NICHT.
	await page.mouse.move(200, 200);
	await page.keyboard.press('Escape');
	await expect(page.getByTestId('sperrbildschirm')).toBeVisible();

	// Passwort der angemeldeten Person (Mock-IMAP nimmt jedes) → wieder frei.
	await page.locator('#sperre-passwort').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Entsperren' }).click();
	await expect(page.getByTestId('sperrbildschirm')).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'Abmelden' })).toBeVisible();

	// Aufräumen: Testschüler weg (Soft-Delete reicht).
	const token = await csrfToken(page);
	const liste = await (await page.request.get(`/api/schueler?q=Sperre-${suffix}`)).json();
	const treffer = (Array.isArray(liste) ? liste : (liste.items ?? liste.data ?? [])).find(
		(/** @type {any} */ s) => s.barcode_id === `S-${suffix}`
	);
	if (treffer?.id) {
		await page.request.delete(`/api/schueler/${treffer.id}`, {
			headers: { 'X-CSRF-Token': token }
		});
	}
});
