import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

// Gate für die Navigationsleiste.
//
// Geprüft wird am BERECHNETEN Stil, nicht an Klassennamen: Die @theme-Skala des Projekts
// hat Tailwind-Namen auf M3-Tonleitern umgebogen, `bg-blue-50` sagt also nichts darüber
// aus, welche Farbe herauskommt. Und bei gleicher Spezifität entscheidet die Reihenfolge
// im Stylesheet, nicht die im class-Attribut — ein Klassen-Grep hätte hier schon zweimal
// das Falsche bestätigt.
//
// Verglichen wird gegen die CSS-Variablen selbst, nicht gegen feste RGB-Werte: Ein
// Farbwechsel in styles/rollen.css soll diesen Test NICHT rot machen. Rot werden soll er,
// wenn die Navigation aufhört, die Rolle zu benutzen.
test.describe('Navigation: Material-3-Rollen', () => {
	test('das ausgewählte Ziel trägt secondary-container, nicht die Primärfarbe', async ({
		page
	}) => {
		await uiLogin(page);
		const ziel = page.getByTitle('Schülerdatei');
		await ziel.click();

		// Auf DIESES Ziel warten, nicht auf "irgendeines ist ausgewählt": Direkt nach dem
		// Login trägt bereits "Ausleihe" die Auswahl, eine blosse Zählung wäre also sofort
		// erfüllt und misst den falschen Knopf.
		await expect(ziel).toHaveAttribute('aria-current', 'page');

		// Die Sollwerte stehen in :root und wechseln nicht — sie dürfen einmalig gelesen
		// werden. Die Ist-Farbe blendet dagegen per transition-colors ein und wird deshalb
		// GEPOLLT: Ein einmaliges Lesen erwischte "rgba(215, 227, 248, 0.992)", also die
		// Mitte des Übergangs, und meldete einen Fehler, den es nicht gab.
		const soll = await page.evaluate(() => {
			const wurzel = getComputedStyle(document.documentElement);
			const alsRGB = (/** @type {string} */ wert) => {
				const d = document.createElement('div');
				d.style.color = wert.trim();
				document.body.appendChild(d);
				const rgb = getComputedStyle(d).color;
				d.remove();
				return rgb;
			};
			return {
				bg: alsRGB(wurzel.getPropertyValue('--color-secondary-container')),
				fg: alsRGB(wurzel.getPropertyValue('--color-on-secondary-container')),
				primary: alsRGB(wurzel.getPropertyValue('--color-primary'))
			};
		});

		await expect
			.poll(() => ziel.evaluate((el) => getComputedStyle(el).backgroundColor), {
				message: 'Fläche des ausgewählten Ziels muss secondary-container sein'
			})
			.toBe(soll.bg);

		const ist = await ziel.evaluate((el) => ({
			fg: getComputedStyle(el).color,
			bg: getComputedStyle(el).backgroundColor,
			radius: getComputedStyle(el).borderRadius
		}));

		expect(ist.fg, 'Schrift auf dem ausgewählten Ziel').toBe(soll.fg);
		expect(ist.bg, 'die Auswahl darf nicht die Primärfarbe sein').not.toBe(soll.primary);

		// Pille, nicht abgerundetes Rechteck. Bewusst nur auf "groß" geprüft:
		// rounded-full rechnet der Browser auf die halbe Höhe herunter.
		expect(Number.parseFloat(ist.radius)).toBeGreaterThan(100);
	});

	test('Abmelden ist nicht als Fehler markiert', async ({ page }) => {
		await uiLogin(page);

		const farbe = await page
			.locator('aside button', { hasText: 'Abmelden' })
			.first()
			.evaluate((el) => getComputedStyle(el).color);

		// Geprüft wird der FARBTON, nicht die Gleichheit mit --color-error.
		//
		// Die erste Fassung verglich genau damit und war blind: Die danger-Variante färbt
		// mit rose-700 (#93000a), --color-error ist #ba1a1a. Zwei Töne derselben
		// Fehlerfamilie — der Test blieb grün, während der Knopf wieder rot war. Aufgefallen
		// ist das erst an der Rückbau-Probe, nicht am grünen Lauf.
		//
		// Rot ist in M3 die Fehlerrolle. Abmelden ist weder Fehler noch Datenverlust und
		// darf deshalb in keinem Rotton stehen.
		const [r, g, b] = farbe.match(/\d+/g).map(Number);
		expect(
			r > g * 1.5 && r > b * 1.5,
			`Abmelden steht in einem Rotton (${farbe}) — Rot ist die Fehlerrolle`
		).toBe(false);
	});

	test('jedes sichtbare Ziel hat ein Symbol', async ({ page }) => {
		await uiLogin(page);

		// Fängt eine Lücke in der Zuordnung von NavIcon: Steht dort ein Name, den menu.js
		// nicht kennt (oder umgekehrt), fehlt still ein Symbol. In der eingeklappten
		// Leiste ist das Symbol die EINZIGE Beschriftung — dann ist der Punkt unbedienbar.
		const ohneSymbol = await page.evaluate(() => {
			const nav = document.querySelector('aside nav');
			return [...nav.querySelectorAll('button[title]')]
				.filter((b) => !b.querySelector('svg'))
				.map((b) => b.getAttribute('title'));
		});
		expect(ohneSymbol, 'Navigationsziele ohne Symbol').toEqual([]);
	});
});
