import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

// Gate gegen unlesbaren Text (WCAG 2.1 AA).
//
// Anlass (07.08.2026): Eine Messung über zehn Bildschirme fand 1.283 Textstellen unter
// dem Mindestkontrast — verursacht von genau ZWEI Palettenwerten, die fast den gesamten
// Sekundärtext trugen:
//
//   slate-400 #909094   2,58–3,18:1   609 Stellen
//   slate-500 #72777f   3,66–4,51:1   674 Stellen
//
// Bemerkenswert daran ist der zweite: #72777f besteht auf REINEM Weiß (4,51:1). Eine
// Prüfung gegen Weiß hätte "knapp bestanden" gemeldet — in dieser Anwendung steht Text
// aber auf surface (#faf9fd) und surface-container (#f1f0f4), und dort sind es 4,30 und
// 3,97. Deshalb misst dieser Test gegen die TATSÄCHLICHE Fläche im Baum, nicht gegen eine
// angenommene.
//
// Was er NICHT bewertet: Text über Bildern, Verläufen und halbtransparenten Flächen. Dort
// müsste man die Deckung zusammenrechnen, und ein geratener Wert wäre schlimmer als kein
// Wert — er würde entweder Fehlalarm schlagen oder echte Fälle verstecken. Die Zahl der
// übersprungenen Knoten steht in der Ausgabe, damit die Lücke sichtbar bleibt.
const SEITEN = ['Mahnwesen', 'Medienkatalog', 'Schülerdatei', 'Bestellungen', 'Signaturen'];

test('Text erfüllt den WCAG-AA-Mindestkontrast', async ({ page }) => {
	await uiLogin(page);

	/** @type {string[]} */
	const verstoesse = [];
	let geprueft = 0;
	let uebersprungen = 0;

	for (const seite of SEITEN) {
		const ziel = page.getByTitle(seite);
		if ((await ziel.count()) === 0) continue;
		await ziel.click();
		// Auf den Navigationszustand warten, nicht auf eine Zeitspanne: Die Seite muss
		// wirklich stehen, sonst misst der Test den vorigen Bildschirm.
		await expect(ziel).toHaveAttribute('aria-current', 'page');
		await page.locator('main').first().waitFor();
		// Kurze Setzzeit, damit die nachgeladenen Listen im Baum stehen. Bewusst KEIN
		// networkidle: Die Anwendung hält eine dauerhafte SSE-Verbindung offen, der
		// Zustand tritt also nie ein (siehe sse-livesync.spec.js).
		await page.waitForTimeout(700);

		const ergebnis = await page.evaluate(() => {
			const leuchtdichte = (/** @type {string} */ rgb) => {
				const teile = rgb.match(/[\d.]+/g);
				if (!teile) return null;
				const [r, g, b] = teile.map(Number).slice(0, 3).map((v) => {
					const c = v / 255;
					return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
				});
				return 0.2126 * r + 0.7152 * g + 0.0722 * b;
			};

			// Nur eine DECKENDE Fläche zählt. Bild, Verlauf oder Teiltransparenz → null,
			// der Knoten wird nicht bewertet.
			const flaeche = (/** @type {Element} */ el) => {
				let e = /** @type {Element|null} */ (el);
				while (e) {
					const st = getComputedStyle(e);
					if (st.backgroundImage && st.backgroundImage !== 'none') return null;
					const bg = st.backgroundColor;
					if (bg && bg !== 'rgba(0, 0, 0, 0)') {
						if (!bg.startsWith('rgb(')) return null; // rgba/oklab mit Deckung < 1
						return bg;
					}
					e = e.parentElement;
				}
				return 'rgb(255, 255, 255)';
			};

			/** @type {string[]} */
			const treffer = [];
			let n = 0;
			let weg = 0;

			for (const el of document.querySelectorAll('main *')) {
				if (!(/** @type {HTMLElement} */ (el).offsetParent)) continue;
				const text = [...el.childNodes]
					.filter((k) => k.nodeType === 3 && k.textContent?.trim())
					.map((k) => k.textContent?.trim())
					.join(' ');
				if (!text) continue;

				const st = getComputedStyle(el);
				const bg = flaeche(el);
				if (!bg || !st.color.startsWith('rgb(')) {
					weg++;
					continue;
				}
				n++;

				const px = Number.parseFloat(st.fontSize);
				const fett = Number.parseInt(st.fontWeight, 10) >= 700;
				// WCAG: grosser Text (>= 24px, oder >= 18.66px und fett) darf auf 3:1.
				const soll = px >= 24 || (px >= 18.66 && fett) ? 3 : 4.5;

				const l1 = leuchtdichte(st.color);
				const l2 = leuchtdichte(bg);
				if (l1 === null || l2 === null) {
					weg++;
					continue;
				}
				const wert = (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05);
				if (wert < soll) {
					treffer.push(
						`${st.color} auf ${bg} → ${wert.toFixed(2)}:1 (nötig ${soll}, ${Math.round(px)}px) „${text.slice(0, 40)}"`
					);
				}
			}
			return { treffer, n, weg };
		});

		geprueft += ergebnis.n;
		uebersprungen += ergebnis.weg;
		for (const t of ergebnis.treffer) verstoesse.push(`[${seite}] ${t}`);
	}

	// Aussagekraft-Untergrenze: Findet der Test kaum Text, misst er nichts und wäre als
	// grüner Lauf wertlos — genau die Sorte Gate, die alles durchwinkt. Der Wert liegt
	// deutlich unter dem gemessenen Bestand, damit normales Datenwachstum ihn nicht
	// auslöst, aber weit über dem, was ein leerer Bildschirm liefert.
	expect(geprueft, 'zu wenige Textknoten erfasst — greift der Test noch?').toBeGreaterThan(300);

	expect(
		verstoesse,
		`Text unter WCAG-AA-Mindestkontrast (${geprueft} Knoten geprüft, ${uebersprungen} nicht bewertbar):\n  ` +
			verstoesse.slice(0, 15).join('\n  ')
	).toEqual([]);
});
