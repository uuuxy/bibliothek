import { test, expect } from '@playwright/test';
import { gehZu, menuepunkt, seedBenutzer, seedSQL, uiLogin, uniqueSuffix } from './helpers.js';

// Eine Seite lädt auf ein Signal oder beim Aufbau der Leitung neu, während jemand sie liest. Scheitert
// das, bleibt ihr Inhalt stehen; überholen sich zwei Antworten, gilt die jüngere. Die Antworten sind
// hier ersetzt: Die Fehlerfälle lassen sich am Server nicht auf Bestellung herbeiführen.

/**
 * Die Leitung steht zunächst nicht (401). Auf `sende` kommen die Ereignisse einmal an, beim nächsten
 * Aufbau, den liveEvents.js selbst versucht.
 * @param {import('@playwright/test').Page} page
 */
async function leitung(page) {
	/** @type {string[] | null} */
	let ereignisse = null;
	await page.route('**/events', (route) => {
		if (!ereignisse) {
			return route.fulfill({ status: 401, contentType: 'application/json', body: '{}' });
		}
		const body = ereignisse.map((name) => `event: ${name}\ndata: {}\n\n`).join('');
		ereignisse = null;
		return route.fulfill({ status: 200, contentType: 'text/event-stream', body });
	});
	return {
		/** @param {string[]} namen */
		sende(namen) {
			ereignisse = namen;
		}
	};
}

const SERVERFEHLER = { status: 500, contentType: 'application/json', body: '{"error":"Probe"}' };
const FRIST = { timeout: 40_000 };

test.afterEach(async ({ page }) => {
	await page.unrouteAll({ behavior: 'ignoreErrors' });
});

test.describe('LMF-Plan im Portal', () => {
	const email = `e2e-nachladen-${uniqueSuffix()}@test.local`;
	test.beforeAll(() => seedBenutzer(email, 'kollegium'));
	test.afterAll(() => seedSQL(`DELETE FROM benutzer WHERE email = '${email}';`));

	/** @param {string} klasse */
	const plan = (klasse) => ({
		ab: '2026-08-01',
		termine: [
			{ id: 'a', datum: '2027-06-28', stunde: 3, art: 'rueckgabe', klassen: [klasse], vermerk: '' }
		],
		eingangsjahrgaenge: []
	});

	/** @param {import('@playwright/test').Page} page */
	async function oeffnePlan(page) {
		await uiLogin(page, email);
		await menuepunkt(page, 'Mein Portal').click();
		await page.getByRole('tab', { name: 'LMF-Plan' }).click();
	}

	test('Scheitert das Nachladen, bleibt der Plan stehen, und eine Meldung nennt den Grund', async ({
		page
	}) => {
		const { sende } = await leitung(page);
		let abrufe = 0;
		await page.route('**/api/lmf-termine**', (route) =>
			++abrufe === 1 ? route.fulfill({ json: plan('09H1') }) : route.fulfill(SERVERFEHLER)
		);
		await oeffnePlan(page);
		await expect(page.getByText('09H1').first()).toBeVisible();

		sende(['connected']);

		await expect(page.getByText('Der Plan wurde nicht aktualisiert: Probe')).toBeVisible(FRIST);
		await expect(page.getByText('09H1').first()).toBeVisible();
		await expect(page.getByText('LMF-Plan nicht geladen')).toHaveCount(0);
	});

	test('Scheitert der erste Abruf, gibt es den Weg zurück', async ({ page }) => {
		let scheitert = true;
		await page.route('**/api/lmf-termine**', (route) =>
			scheitert ? route.fulfill(SERVERFEHLER) : route.fulfill({ json: plan('09H1') })
		);
		await oeffnePlan(page);
		await expect(page.getByText('LMF-Plan nicht geladen')).toBeVisible();

		scheitert = false;
		await page.getByRole('button', { name: 'Erneut versuchen' }).click();

		await expect(page.getByText('09H1').first()).toBeVisible();
		await expect(page.getByText('LMF-Plan nicht geladen')).toHaveCount(0);
	});

	test('Kommt die ältere Antwort zuletzt, bleibt der neue Plan stehen', async ({ page }) => {
		const { sende } = await leitung(page);
		/** @type {() => void} */
		let gibAlteFrei = () => {};
		const alteFrei = new Promise((fertig) => {
			gibAlteFrei = () => fertig(undefined);
		});
		let abrufe = 0;
		await page.route('**/api/lmf-termine**', async (route) => {
			const nr = ++abrufe;
			if (nr === 1) return route.fulfill({ json: plan('KLASSE-ANFANG') });
			if (nr === 2) {
				await alteFrei;
				return route.fulfill({ json: plan('KLASSE-ALT') });
			}
			return route.fulfill({ json: plan('KLASSE-NEU') });
		});
		await oeffnePlan(page);
		await expect(page.getByText('KLASSE-ANFANG').first()).toBeVisible();

		sende(['connected', 'lmf-plan']);
		await expect(page.getByText('KLASSE-NEU').first()).toBeVisible(FRIST);

		const alteAntwort = page.waitForResponse('**/api/lmf-termine**');
		gibAlteFrei();
		await alteAntwort;
		await page.evaluate(() => new Promise((fertig) => requestAnimationFrame(fertig)));

		await expect(page.getByText('KLASSE-NEU').first()).toBeVisible();
		await expect(page.getByText('KLASSE-ALT')).toHaveCount(0);
	});
});

// Die Liste steht nur von Mai bis Juli da, deshalb ist die Antwort hier ersetzt.
test('Abgängerliste: Scheitert das Nachladen, bleibt die Liste stehen', async ({ page }) => {
	const { sende } = await leitung(page);
	let abrufe = 0;
	await page.route('**/api/abgaenger', (route) =>
		++abrufe === 1
			? route.fulfill({
					json: {
						fenster: { offen: true, von: '01.05.', bis: '31.07.' },
						abgaenger: [
							{
								id: 'a1',
								vorname: 'Mia',
								nachname: 'Lange',
								klasse: '10a',
								barcode_id: 'A-1',
								offene_buecher: 2,
								ueberfaellig: 0
							}
						]
					}
				})
			: route.fulfill(SERVERFEHLER)
	);
	await uiLogin(page);
	await gehZu(page, '/abgaenger');
	await expect(page.getByText('Mia Lange')).toBeVisible();

	sende(['connected']);

	await expect(page.getByText('Die Liste wurde nicht aktualisiert: Probe')).toBeVisible(FRIST);
	await expect(page.getByText('Mia Lange')).toBeVisible();
	await expect(page.getByText('Abgängerliste nicht geladen')).toHaveCount(0);
});

// Für den Bildschirmleser stand „Keine Bestellung wartet", obwohl der Abruf gar nicht geantwortet hatte.
test('Bestellhistorie: Scheitert der erste Abruf, behauptet der Kopf nichts über wartende Bestellungen', async ({
	page
}) => {
	await page.route('**/api/bestellhistorie**', (route) => route.fulfill(SERVERFEHLER));
	await uiLogin(page);
	await menuepunkt(page, 'Bestellungen').click();
	await page.getByRole('tab', { name: 'Bestellhistorie', exact: true }).click();

	await expect(page.getByText('Bestellhistorie nicht geladen')).toBeVisible();
	await expect(page.getByText('Keine Bestellung wartet')).toHaveCount(0);
});
