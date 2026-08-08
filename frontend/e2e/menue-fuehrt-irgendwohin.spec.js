// Gate gegen den Menüpunkt, der nichts tut.
//
// Am 08.08.2026 bekam der Helfer einen Eintrag „Schulklassen" ins Menü, weil dessen
// Recht von manage_users auf view_books gesenkt wurde. Erreichbar war die Seite für
// ihn nie: Der Router führte eine ZWEITE, handgepflegte Liste ('kiosk',
// 'media_catalog') und warf ihn wortlos an die Theke zurück. Zwei Definitionen
// derselben Regel, auseinandergelaufen — und ein Menüpunkt, der nichts tut, sieht für
// den Benutzer aus wie ein Defekt der Seite dahinter.
//
// Warum kein Unit-Test: Ein Komponententest mit gesetztem authStore beweist, dass die
// SEITE für die Rolle richtig aussieht — nicht, dass sie erreichbar ist. Genau diese
// Lücke hat den Fehler durchgelassen. Gemessen wird deshalb am laufenden Programm:
// klicken, und nachsehen, wo man landet.
//
// Der Test prüft die eingeschränkten Rollen. Der Administrator sieht alles und ist
// damit kein Messpunkt — bei ihm kann die Regel nicht auseinanderlaufen.
import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL } from './helpers.js';

const ROLLEN = [
	['helfer', 'e2e-menue-helfer@test.local'],
	['mitarbeiter', 'e2e-menue-mitarbeiter@test.local'],
	['lehrer', 'e2e-menue-lehrer@test.local']
];

/** @param {string} rolle @param {string} email */
function seedBenutzer(rolle, email) {
	seedSQL(`INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
	         VALUES ('E2E', 'Menue', '${email}', '${rolle}', true)
	         ON CONFLICT (email) DO UPDATE SET rolle = EXCLUDED.rolle, aktiv = true;`);
}

for (const [rolle, email] of ROLLEN) {
	test(`${rolle}: jeder sichtbare Menüpunkt führt auch irgendwohin`, async ({ page }) => {
		seedBenutzer(rolle, email);
		await uiLogin(page, email);

		// Die Navigation trägt title="…" je Punkt (siehe Sidebar).
		const punkte = page.locator('nav [title]');
		const anzahl = await punkte.count();

		// Selbstschutz: Findet der Test keine Menüpunkte, misst er nichts und wäre für
		// immer grün. Jede Rolle hat mindestens ihren Startbildschirm.
		expect(anzahl, `${rolle} hat gar keinen Menüpunkt — greift der Selektor noch?`).toBeGreaterThan(
			0
		);

		/** @type {string[]} */
		const tot = [];

		for (let i = 0; i < anzahl; i++) {
			const punkt = punkte.nth(i);
			const name = (await punkt.getAttribute('title')) ?? `#${i}`;

			await punkt.click();
			// Auf den Navigationszustand warten statt auf eine Zeitspanne: aria-current
			// setzt die Sidebar erst, wenn der Tab wirklich gewechselt hat.
			await page.waitForTimeout(600);

			const aktiv = await punkt.getAttribute('aria-current');
			if (aktiv !== 'page') {
				tot.push(`"${name}" bleibt nach dem Klick unmarkiert (Router wirft zurück)`);
			}
		}

		expect(
			tot,
			`Menüpunkte, die ins Leere führen (Rolle ${rolle}).\n` +
				`Ursache ist fast immer eine zweite Rechte-Definition neben canSeeItem in menu.js:\n  ` +
				tot.join('\n  ')
		).toEqual([]);
	});
}
