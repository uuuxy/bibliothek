// Gate: Zustände nach Material 3 (States), gemessen im Browser.
//
// - Tastaturfokus: M3, States: „When an element is tabbed to, it appears in its focused state with
//   a ring-like keyboard focus indicator." Die Form ist Googles md-focus-ring (material-web): 3 px
//   in secondary, 2 px außerhalb; innen („inward") bei Reiter, Listen-, Menü- und Auswahlzeile.
//   Ein Textfeld trägt keinen Ring, sein Rahmen wird primary (outlined text field).
// - Zeigen: eine Schicht in der Farbe des Inhalts mit 8 % („Hover +8% opacity"), beim Textfeld
//   der Rahmen in on-surface (hover-outline-color); gesperrt keine Schicht.
// - Drücken: „A ripple overlay signifies a pressed state" (md-ripple), auch über die Tastatur.
// Sieht nicht: Stopps nach den ersten 30 je Seite, Seiten und Zustände, die der Test nicht öffnet;
// die Schicht misst es an den ersten acht Knöpfen im Inhalt je Seite.
import { test, expect } from '@playwright/test';
import { uiLogin, gehZu } from './helpers.js';
import { beobachteDatenanfragen, warteAufInhalt } from './messhilfe.js';
import { SEITEN } from './seiten.js';

const STOPPS = 30;

/** Liest den Fokus am aktiven Element. Läuft im Browser. */
function messeFokus() {
	const el = /** @type {HTMLElement | null} */ (document.activeElement);
	if (!el || el === document.body) return null;
	const probe = document.createElement('span');
	document.body.append(probe);
	const farbe = (/** @type {string} */ rolle) => {
		probe.style.color = `var(--color-${rolle})`;
		return getComputedStyle(probe).color;
	};
	const secondary = farbe('secondary');
	const primary = farbe('primary');
	probe.remove();
	const s = getComputedStyle(el);
	const rolle = el.getAttribute('role') ?? '';
	const typ = el.getAttribute('type') ?? '';
	const textfeld =
		el.tagName === 'TEXTAREA' ||
		el.tagName === 'SELECT' ||
		rolle === 'combobox' ||
		(el.tagName === 'INPUT' && !/^(checkbox|radio|range|button|submit|file)$/.test(typ));
	const innen =
		/^(tab|menuitem|menuitemradio|menuitemcheckbox|option|row)$/.test(rolle) ||
		el.tagName === 'TR' ||
		el.tagName === 'LI';
	let rahmenPrimary = false;
	for (let x = /** @type {Element | null} */ (el), i = 0; x && i < 4; x = x.parentElement, i++)
		if (getComputedStyle(x).borderTopColor === primary) rahmenPrimary = true;
	return {
		wer: `${el.tagName.toLowerCase()}${rolle ? `[${rolle}]` : ''} „${(el.getAttribute('aria-label') || el.textContent || '').trim().slice(0, 30)}“`,
		textfeld,
		innen,
		stil: s.outlineStyle,
		breite: s.outlineWidth,
		abstand: s.outlineOffset,
		farbeOk: s.outlineColor === secondary,
		rahmenPrimary
	};
}

test('Tastaturfokus wie Googles md-focus-ring auf jeder Seite', async ({ page }) => {
	test.setTimeout(600_000);
	// Ohne Bewegung steht der Ring sofort auf 3 px (basis.css kürzt jede Animation auf 0 s).
	await page.emulateMedia({ reducedMotion: 'reduce' });
	const anfragen = beobachteDatenanfragen(page);
	const zaehle = () => page.locator('body *').count();
	/** @type {string[]} */
	const abweichungen = [];
	let gemessen = 0;
	let textfelder = 0;
	let innen = 0;
	/** @param {string} seite */
	const gehDurch = async (seite) => {
		await page.locator('body').click({ position: { x: 1, y: 1 } });
		const gesehen = new Set();
		for (let i = 0; i < STOPPS; i++) {
			await page.keyboard.press('Tab');
			const f = await page.evaluate(messeFokus);
			if (!f || gesehen.has(f.wer)) continue;
			gesehen.add(f.wer);
			gemessen++;
			if (f.textfeld) {
				textfelder++;
				if (f.stil !== 'none' || !f.rahmenPrimary)
					abweichungen.push(
						`${seite}: ${f.wer} Textfeld mit Ring ${f.stil} oder ohne Rahmen in primary`
					);
				continue;
			}
			if (f.innen) innen++;
			const soll = f.innen ? '-3px' : '2px';
			if (f.stil !== 'solid' || f.breite !== '3px' || !f.farbeOk || f.abstand !== soll)
				abweichungen.push(
					`${seite}: ${f.wer} ${f.stil} ${f.breite} Abstand ${f.abstand}, secondary ${f.farbeOk}, soll 3px ${soll}`
				);
		}
	};
	for (const [seite, pfad] of [
		['Anmeldung', '/'],
		['Katalog', '/katalog']
	]) {
		await page.goto(pfad);
		await warteAufInhalt(anfragen, zaehle);
		await gehDurch(seite);
	}
	await uiLogin(page);
	for (const [seite, pfad] of SEITEN) {
		await gehZu(page, pfad);
		await warteAufInhalt(anfragen, zaehle);
		await gehDurch(seite);
	}
	// Selbstschutz: Bricht die Navigation, misst der Test still nichts mehr. Die Schwellen hält
	// auch die frische Datenbank der CI, die weniger Zeilen zum Fokussieren hat.
	expect(gemessen, 'kaum Fokusstopps gemessen').toBeGreaterThan(150);
	expect(textfelder, 'kaum Textfelder gemessen').toBeGreaterThan(10);
	expect(innen, 'keine Zeile mit innerem Ring gemessen').toBeGreaterThan(0);
	expect(abweichungen, 'Fokus anders als md-focus-ring (material-web)').toEqual([]);
});

/** Liest die Schicht (::after) eines Elements. Läuft im Browser. @param {Element} el */
function schicht(el) {
	const s = getComputedStyle(el, '::after');
	return {
		inhalt: s.content,
		deckkraft: Number(s.opacity),
		farbeWieInhalt: s.backgroundColor === getComputedStyle(el).color
	};
}

test('Jeder Knopf zeigt beim Zeigen eine Schicht mit 8 % in der Farbe des Inhalts', async ({
	page
}) => {
	test.setTimeout(300_000);
	await page.emulateMedia({ reducedMotion: 'reduce' });
	const anfragen = beobachteDatenanfragen(page);
	const zaehle = () => page.locator('body *').count();
	await uiLogin(page);
	/** @type {string[]} */
	const ohne = [];
	let gemessen = 0;
	for (const [seite, pfad] of SEITEN) {
		await gehZu(page, pfad);
		await warteAufInhalt(anfragen, zaehle);
		// Im Inhalt, nicht in der Navigation: Deren Knöpfe stünden sonst auf jeder Seite vorn.
		const knoepfe = page.locator('main button:visible:not(:disabled):not(.no-state-layer)');
		const n = Math.min(await knoepfe.count(), 8);
		for (let i = 0; i < n; i++) {
			const knopf = knoepfe.nth(i);
			const ruhe = await knopf.evaluate(schicht);
			try {
				await knopf.hover({ timeout: 2000 });
			} catch {
				continue; // verdeckt oder außerhalb des Fensters
			}
			const zeigen = await knopf.evaluate(schicht);
			gemessen++;
			if (
				ruhe.inhalt === 'none' ||
				ruhe.deckkraft !== 0 ||
				zeigen.deckkraft !== 0.08 ||
				!zeigen.farbeWieInhalt
			) {
				const name = ((await knopf.getAttribute('aria-label')) || (await knopf.innerText()))
					.trim()
					.slice(0, 30);
				ohne.push(
					`${seite}: „${name}“ ruhend ${ruhe.deckkraft}, zeigend ${zeigen.deckkraft}, Farbe des Inhalts ${zeigen.farbeWieInhalt}`
				);
			}
		}
	}
	expect(gemessen, 'kaum Knöpfe gemessen').toBeGreaterThan(30);
	expect(ohne, 'Knopf ohne State-Layer nach M3 (Hover +8% opacity)').toEqual([]);
});

test('Gesperrt keine Schicht, Textfeld zeigt mit Rahmen in on-surface', async ({ page }) => {
	await page.emulateMedia({ reducedMotion: 'reduce' });
	await page.goto('/');
	const email = page.locator('#login-email');
	// Das Feld der Adresse bekommt beim Laden den Fokus (dann ist sein Rahmen primary); gemessen
	// wird am Feld des Passworts.
	await expect(email).toBeFocused();
	const passwort = page.locator('#login-password');
	await passwort.hover();
	const [rahmen, onSurface] = await passwort.evaluate((el) => {
		const probe = document.createElement('span');
		probe.style.color = 'var(--color-on-surface)';
		document.body.append(probe);
		const soll = getComputedStyle(probe).color;
		probe.remove();
		return [getComputedStyle(el).borderTopColor, soll];
	});
	expect(rahmen, 'outlined text field: hover-outline-color on-surface').toBe(onSurface);

	// Der Knopf „Anmelden" ist gesperrt, solange die Felder leer sind: keine Schicht beim Zeigen.
	await email.fill('');
	const anmelden = page.getByRole('button', { name: 'Anmelden' });
	if (await anmelden.isDisabled()) {
		// Der gesperrte Knopf ist nie bereit; gezeigt wird absichtlich trotzdem.
		await anmelden.hover({ force: true }); // NOSONAR
		expect((await anmelden.evaluate(schicht)).deckkraft, 'gesperrt ohne Schicht').toBe(0);
	}
});

test('Drücken lässt eine Welle wachsen, mit Zeiger und mit Tastatur', async ({ page }) => {
	await page.goto('/');
	const knopf = page.getByRole('button', { name: 'Anmelden' });
	await page.locator('#login-email').fill('niemand@schule.example');
	await page.locator('#login-password').fill('x');
	await expect(knopf).toBeEnabled();
	const box = /** @type {{ x: number, y: number, width: number, height: number }} */ (
		await knopf.boundingBox()
	);
	// Zum Messen ohne Absenden: Ein Druck, der den Knopf verlässt, löst keinen Klick aus.
	await page.mouse.move(box.x + 10, box.y + box.height / 2);
	await page.mouse.down();
	await expect(knopf.locator('.m3-welle')).toHaveCount(1);
	const kreis = await knopf.locator('.m3-welle > span').evaluate((el) => ({
		farbe: getComputedStyle(el).backgroundImage,
		form: getComputedStyle(el.parentElement ?? el).borderRadius
	}));
	expect(kreis.farbe, 'Welle als Verlauf wie md-ripple').toContain('radial-gradient');
	await page.mouse.move(box.x + box.width + 40, box.y + box.height + 40);
	await page.mouse.up();
	await expect(knopf.locator('.m3-welle'), 'die Welle blendet aus').toHaveCount(0, {
		timeout: 2000
	});

	// Über die Tastatur: ein Reiter des Bildschirms, ohne dass sich etwas Unumkehrbares tut.
	await page.locator('#login-email').focus();
	await page.keyboard.press('Shift+Tab');
	const fokus = await page.evaluate(() => document.activeElement?.tagName);
	if (fokus === 'BUTTON' || fokus === 'A') {
		await page.keyboard.press('Space');
		await expect(page.locator('.m3-welle').first()).toBeAttached();
	}
});
