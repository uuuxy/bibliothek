// Rundgang je Rolle: Was das Öffnen eines Bildschirms oder Reiters auslöst, lehnt der Server nicht
// ab, und nichts stürzt ab. Die übrige Suite meldet sich fast nur als Administrator an, dem der
// Server nichts ablehnt. Einen Fehler hinter einem Knopf sieht der Test nicht; er drückt nur Reiter.
import { test, expect } from '@playwright/test';
import { seedBenutzer, seedSQL } from './helpers.js';
import { geheRundgang, legeAktenAn } from './rundgang-hilfe.js';
import { pfadJeBildschirm } from '../src/lib/routenTabelle.js';
import { ROLLEN_AUSWAHL } from '../src/lib/components/benutzer/benutzerRollen.js';

const BILDSCHIRME = Object.values(pfadJeBildschirm);

/** @type {ReturnType<typeof legeAktenAn>} */
let akten;

test.beforeAll(() => {
	akten = legeAktenAn();
});

test.afterAll(() => {
	akten.raeumeAuf();
	seedSQL(`DELETE FROM benutzer WHERE email LIKE 'e2e-rundgang-%@test.local';`);
});

for (const { value: rolle } of ROLLEN_AUSWAHL) {
	test(`${rolle}: Bildschirme und Reiter öffnen ohne Ablehnung und ohne Absturz`, async ({
		page
	}) => {
		test.setTimeout(180_000);
		const email = `e2e-rundgang-${rolle}@test.local`;
		seedBenutzer(email, rolle);

		const { stoerungen, besucht, geklickt } = await geheRundgang(
			page,
			email,
			[...BILDSCHIRME, akten.buchakte],
			akten.ausweis
		);

		expect(stoerungen, `Störungen im Rundgang als ${rolle}`).toEqual([]);
		// Grün heißt nur dann etwas, wenn der Rundgang auch unterwegs war.
		expect(besucht.size).toBeGreaterThanOrEqual(rolle === 'admin' ? BILDSCHIRME.length : 1);
		expect(geklickt).toBeGreaterThan(0);
	});
}
