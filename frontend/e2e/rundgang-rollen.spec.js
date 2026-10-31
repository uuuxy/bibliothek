// Rundgang je Rolle: Was das Öffnen eines Bildschirms oder Reiters auslöst, lehnt der Server nicht
// ab, und nichts stürzt ab. Die übrige Suite meldet sich fast nur als Administrator an, dem der
// Server nichts ablehnt. Einen Fehler hinter einem Knopf sieht der Test nicht; er drückt nur Reiter.
import { test, expect } from '@playwright/test';
import { uiLogin, seedBenutzer, seedSQL } from './helpers.js';
import { beobachteDatenanfragen, warteAufInhalt } from './messhilfe.js';
import { pfadJeBildschirm } from '../src/lib/routenTabelle.js';
import { ROLLEN_AUSWAHL } from '../src/lib/components/benutzer/benutzerRollen.js';

const BILDSCHIRME = Object.values(pfadJeBildschirm);

/**
 * @param {import('@playwright/test').Page} page
 * @param {() => string} ort wo der Rundgang gerade steht
 * @returns {string[]} wächst mit jeder Störung
 */
function sammleStoerungen(page, ort) {
	/** @type {string[]} */
	const stoerungen = [];
	page.on('pageerror', (fehler) => stoerungen.push(`${ort()}: Absturz: ${fehler.message}`));
	page.on('console', (meldung) => {
		// Eine abgelehnte Anfrage steht unten mit ihrer Adresse; der Browser meldet sie hier ohne.
		if (meldung.type() === 'error' && !meldung.text().startsWith('Failed to load resource')) {
			stoerungen.push(`${ort()}: Konsole: ${meldung.text()}`);
		}
	});
	page.on('response', (antwort) => {
		const { pathname } = new URL(antwort.url());
		// 401 gehört zur Anmeldung: Davor fragt die Seite, ob schon eine Sitzung besteht.
		if (pathname.startsWith('/api/') && antwort.status() >= 400 && antwort.status() !== 401) {
			stoerungen.push(
				`${ort()}: ${antwort.status()} auf ${antwort.request().method()} ${pathname}`
			);
		}
	});
	return stoerungen;
}

test.afterAll(() => {
	seedSQL(`DELETE FROM benutzer WHERE email LIKE 'e2e-rundgang-%@test.local';`);
});

for (const { value: rolle } of ROLLEN_AUSWAHL) {
	test(`${rolle}: Bildschirme und Reiter öffnen ohne Ablehnung und ohne Absturz`, async ({
		page
	}) => {
		test.setTimeout(180_000);
		const email = `e2e-rundgang-${rolle}@test.local`;
		seedBenutzer(email, rolle);

		let ort = 'Anmeldung';
		const stoerungen = sammleStoerungen(page, () => ort);
		const anfragen = beobachteDatenanfragen(page);
		const steht = () => warteAufInhalt(anfragen, () => page.locator('body *').count());
		const reiter = page.getByRole('tab', { disabled: false });
		await uiLogin(page, email);

		/** @type {Set<string>} */
		const besucht = new Set();
		let geklickt = 0;
		for (const pfad of BILDSCHIRME) {
			ort = pfad;
			await page.goto(pfad);
			await page.getByRole('button', { name: 'Abmelden' }).waitFor();
			await steht();
			// Was der Rolle verschlossen ist, führt auf ihre Startseite; die zählt nur einmal.
			const angekommen = new URL(page.url()).pathname;
			if (besucht.has(angekommen)) continue;
			besucht.add(angekommen);

			for (let i = 0; i < (await reiter.count()); i++) {
				ort = `${angekommen} › ${await reiter.nth(i).innerText()}`;
				await reiter.nth(i).click();
				await steht();
				geklickt++;
			}
		}

		expect(stoerungen, `Störungen im Rundgang als ${rolle}`).toEqual([]);
		// Grün heißt nur dann etwas, wenn der Rundgang auch unterwegs war.
		expect(besucht.size).toBeGreaterThanOrEqual(rolle === 'admin' ? BILDSCHIRME.length : 1);
		expect(geklickt).toBeGreaterThan(0);
	});
}
