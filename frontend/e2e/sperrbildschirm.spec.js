import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, csrfToken, uniqueSuffix, ADMIN_PASSWORD } from './helpers.js';

/**
 * Inaktivitäts-Wächter am Live-Pfad.
 *
 * Zwei Stufen: Nach der kurzen Frist verschwindet der geladene Schüler aus der Theke
 * (der nächste an der Theke darf nicht den vorigen sehen), nach der langen kommt der
 * Sperrbildschirm, der nur mit dem eigenen Passwort wieder aufgeht. Der Store-Test
 * (idleLock.test.js) kennt die Uhr; dieser Test beweist, dass App, Einstellungs-Endpunkt
 * und Sperrbildschirm zusammen funktionieren — mit Playwrights gestellter Uhr, denn die
 * Fristen sind Minuten und kein Test wartet fünf davon.
 *
 * Die Sperre gilt am Server: Hinter ihr beantwortet er keine Anfrage, und weder Neuladen
 * noch ein neuer Tab öffnen die Anwendung. Dass sie bei einem Ausfall des Mailservers mit
 * dem Passwort der Anmeldung aufgeht, steht in auth/sperre_pg_test.go — der Stack hier
 * nimmt jedes Passwort an.
 *
 * Die Fristen werden NICHT verändert (Vorgaben 5/15 Minuten) — ein Teardown, der die
 * Konfiguration der Anlage anfasst, ist eine eigene Fehlerquelle.
 */
test('Theke leert sich nach 5 Minuten, Sperrbildschirm nach 15, Passwort entsperrt', async ({
	page
}) => {
	// Eine Schleife in der Oberfläche endet als Seitenfehler und in Hunderten Anfragen,
	// während das Endbild stimmt. Beides wird deshalb mitgezählt.
	/** @type {string[]} */
	const seitenfehler = [];
	/** @type {string[]} */
	const anfragen = [];
	page.on('pageerror', (e) => seitenfehler.push(e.message));
	page.on('response', (r) => {
		const pfad = new URL(r.url()).pathname;
		if (pfad.startsWith('/api/') || pfad === '/events') anfragen.push(`${r.status()} ${pfad}`);
	});

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
	// Erst tippen, wenn das Scanfeld WIRKLICH den Fokus hat — der Kiosk fokussiert es beim
	// Laden selbst, aber nach dem Rendern; ein sofortiges Tippen lief im Suite-Lauf ins Leere.
	await expect
		.poll(() => page.evaluate(() => document.activeElement?.id ?? ''), { timeout: 5000 })
		.toBe('omnibox-input');
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

	// Lange Frist vorbei: Sperrbildschirm. Die Anwendung bleibt dahinter stehen, damit
	// Ungespeichertes die Sperre überlebt, ist aber ausgeblendet und für Screenreader nicht
	// da; Druckvorschau, Tab und die Maske selbst prüft e2e/sperre-ungespeichertes.spec.js.
	await page.clock.fastForward('10:00');
	await expect(page.getByTestId('sperrbildschirm')).toBeVisible();
	await expect(page.getByRole('button', { name: 'Abmelden', exact: true })).toHaveCount(0);
	await expect(page.getByPlaceholder(/scannen/i)).toBeHidden();

	// Bedienung im gesperrten Zustand entsperrt NICHT.
	await page.mouse.move(200, 200);
	await page.keyboard.press('Escape');
	await expect(page.getByTestId('sperrbildschirm')).toBeVisible();

	// Hinter der Sperre liefert der Server nichts: 423 statt der Leserliste, und über die
	// Anmeldung selbst nur, wer angemeldet ist. Die Sperre geht nach dem Verdecken an den
	// Server, deshalb gepollt.
	await expect
		.poll(async () => (await page.request.get('/api/schueler')).status(), { timeout: 5000 })
		.toBe(423);
	const zustand = await page.request.get('/api/auth/me');
	expect(zustand.status()).toBe(423);
	expect(Object.keys(await zustand.json()).sort()).toEqual(['email', 'error']);

	// Neuladen öffnet die Anwendung nicht — und fragt den Server nur nach dem Zustand der
	// Anmeldung und den Fristen, nicht nach Daten.
	anfragen.length = 0;
	const fristenGelesen = page.waitForResponse(
		(r) => new URL(r.url()).pathname === '/api/einstellungen/sitzung'
	);
	await page.reload();
	await expect(page.getByTestId('sperrbildschirm')).toBeVisible();
	await expect(page.getByPlaceholder(/scannen/i)).toHaveCount(0);
	await expect(page.locator('#login-email, input[type="email"]')).toHaveCount(0);
	// Die Fristen sind die letzte Anfrage des Starts. Eine Schleife schickte sie Hunderte Male;
	// ihre Antworten träfen in der Sekunde danach ein.
	await fristenGelesen;
	await page.waitForTimeout(1000);
	expect(anfragen.length, `Anfragen nach dem Neuladen: ${anfragen.join(', ')}`).toBeLessThan(8);
	expect(
		anfragen.filter(
			(a) => a.startsWith('200 ') && !a.includes('/api/auth/') && !a.includes('csrf')
		),
		'hinter der Sperre darf keine Anfrage Daten liefern'
	).toEqual([]);

	// Ein neuer Tab desselben Browsers auch nicht.
	const zweiterTab = await page.context().newPage();
	await zweiterTab.goto('/');
	await expect(zweiterTab.getByTestId('sperrbildschirm')).toBeVisible();
	await expect(zweiterTab.getByPlaceholder(/scannen/i)).toHaveCount(0);

	// Passwort der angemeldeten Person (Mock-IMAP nimmt jedes) → wieder frei.
	await page.locator('#sperre-passwort').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Entsperren' }).click();
	await expect(page.getByTestId('sperrbildschirm')).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'Abmelden' })).toBeVisible();
	expect((await page.request.get('/api/schueler')).status()).toBe(200);

	// Aufgeschlossen ist die Anmeldung, nicht das Fenster: Der zweite Tab geht mit auf.
	await expect(zweiterTab.getByTestId('sperrbildschirm')).toHaveCount(0);
	await expect(zweiterTab.getByPlaceholder(/scannen/i).first()).toBeVisible();
	await zweiterTab.close();

	expect(seitenfehler, 'Fehler in der Oberfläche während des Laufs').toEqual([]);

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
