// Gate: Alle Suchpillen sehen gleich aus, und die Hero-Felder haben beim Betreten Fokus.
//
// Peter am 10.08.2026: „die omnibox bei mein portal und katalog ist eine komplett andere,
// es fehlt auch der fokus". Er hatte recht, und zwar an sieben Messwerten gleichzeitig —
// Höhe (48 gegen 42/58), Radius (Pille gegen 12 px), Fläche, Rahmen, Fokusfarbe,
// Schriftgröße und Platzhaltertext („… suchen …" gegen „… eingeben …"). Ursache waren
// drei Kopien derselben Bauart, die auseinandergelaufen sind.
//
// Warum gemessen und nicht gegrept: Genau diese Werte stehen als Utility-Klassen in vier
// Dateien. Ein Grep zählt Namen, aber niemand liest Namen — man sieht 48 px gegen 58 px.
//
// Warum der Fokus NICHT mit fill() geprüft wird: locator.fill() fokussiert das Feld
// implizit und macht damit jeden Fokusfehler unsichtbar. Genau daran ist im Kiosk schon
// einmal ein Scanner-Problem vorbeigelaufen. Hier wird blind getippt — wie ein Scanner es
// tut — und danach nachgesehen, wo die Zeichen gelandet sind.
import { test, expect } from '@playwright/test';
import { uiLogin, gehZu } from './helpers.js';

/** Die Bildschirme, auf denen eine Suchpille steht. */
const PILLEN = [
	{ name: 'Kiosk (Omnibox)', pfad: '/kiosk', id: 'omnibox-input' },
	{ name: 'Medienkatalog', pfad: '/medienkatalog', id: 'katalog-suchfeld' },
	{ name: 'Mein Portal', pfad: '/kollegium-portal', id: 'portal-suchfeld' }
];

/**
 * Der öffentliche OPAC steht bewusst getrennt: Er liegt vor jeder Anmeldung, und die
 * angemeldete Sitzung würde /katalog gar nicht erst so ausliefern (siehe
 * katalog-pfadkollision.spec.js). Gemessen wird er mit denselben Werten.
 */
const OPAC = { name: 'Öffentlicher Katalog', pfad: '/katalog', id: 'opac-suchfeld' };

/** Die Felder, die beim Betreten der Seite den Fokus tragen MÜSSEN. */
const MIT_FOKUS = [
	{ name: 'Mein Portal', pfad: '/kollegium-portal', id: 'portal-suchfeld', anmelden: true },
	{ name: 'Kiosk (Omnibox)', pfad: '/kiosk', id: 'omnibox-input', anmelden: true },
	{ name: 'Öffentlicher Katalog', pfad: OPAC.pfad, id: OPAC.id, anmelden: false }
];

/**
 * Liest die Werte, die den sichtbaren Unterschied ausmachen — vom Feld UND von der
 * Pille, in der es sitzt. Die Pille trägt Fläche, Rahmen und Radius, das Feld die
 * Schrift; ein Vergleich nur des einen oder anderen ginge an der Sache vorbei.
 */
const MESSEN = (/** @type {string} */ id) => {
	const feld = document.getElementById(id);
	if (!feld) return null;
	const pille = feld.parentElement;
	if (!pille) return null;
	const p = getComputedStyle(pille);
	const f = getComputedStyle(feld);
	return {
		hoehe: Math.round(pille.getBoundingClientRect().height),
		radius: p.borderRadius,
		flaeche: p.backgroundColor,
		rahmenbreite: p.borderTopWidth,
		// Die FARBE des Randes, nicht nur seine Breite.
		//
		// Bis zum 11.08.2026 verglich dieser Test nur borderTopWidth. Eine Pille mit rotem
		// statt blauem Fokusrand wäre also durchgegangen — und genau daran hat Peter beim
		// Nebeneinanderlegen zweier Bildschirme gezweifelt („da fehlt die blaue Linie").
		// Ein Gate, das die auffälligste Eigenschaft nicht misst, beantwortet die Frage
		// nicht, für die es gebaut wurde.
		randfarbe: p.borderTopColor,
		// Der Fokusring liegt als box-shadow an (Tailwinds ring-*), nicht als border.
		ring: p.boxShadow,
		schriftgroesse: f.fontSize,
		textfarbe: f.color
	};
};

/**
 * Misst eine Pille im FOKUSSIERTEN Zustand.
 *
 * Zwei Anläufe, zwei Lehren:
 *
 * 1. Ohne jede Vorbereitung misst man Animationen. Portal und OPAC ziehen den Fokus beim
 *    Betreten, `focus-within` blendet die Fläche über 200 ms nach Weiß, und gemessen
 *    wurden zwei Zwischenbilder derselben Überblendung (rgb(247,246,249) gegen
 *    rgb(242,241,245)) — ein Unterschied, den es nicht gab.
 *
 * 2. Der naheliegende Ausweg „blur() und im Ruhezustand messen" funktioniert hier NICHT.
 *    Die Kiosk-Omnibox holt sich den Fokus per Effekt zurück, sobald sie ihn verliert —
 *    absichtlich, sie ist ein Scanner-Terminal. Die Messung fiel dort mal auf den
 *    ruhenden, mal auf den fokussierten Zustand und war damit von der Tagesform abhängig.
 *
 * Deshalb wird der Zustand gemessen, den JEDE Pille zuverlässig einnehmen kann: Feld
 * fokussieren, warten bis zwei Messungen gleich sind. Höhe, Radius, Rahmen, Schriftgröße
 * und Textfarbe hängen ohnehin nicht am Fokus — genau die Werte, an denen Portal, OPAC
 * und Katalog auseinanderliefen. Die ruhende Füllfarbe deckt die Ratsche in
 * src/lib/frontend-hygiene-suchfelder.test.js ab: Wer sie ändern wollte, müsste das
 * Bauteil anfassen.
 */
async function fokussiertMessen(/** @type {import('@playwright/test').Page} */ page, id) {
	await page.focus(`#${id}`);

	let vorher = null;
	for (let i = 0; i < 20; i++) {
		const jetzt = await page.evaluate(MESSEN, id);
		if (vorher && JSON.stringify(vorher) === JSON.stringify(jetzt)) return jetzt;
		vorher = jetzt;
		await page.waitForTimeout(50);
	}
	return vorher;
}

test('Jede Suchpille hat dieselben Maße, Farben und Schriftgröße', async ({ page }) => {
	await uiLogin(page);

	/** @type {{name: string, werte: any}[]} */
	const gemessen = [];
	for (const { name, pfad, id } of PILLEN) {
		await page.goto(pfad);
		await page.locator(`#${id}`).waitFor();
		const werte = await fokussiertMessen(page, id);
		expect(werte, `${name}: Pille nicht messbar`).not.toBeNull();
		gemessen.push({ name, werte });
	}

	// Der öffentliche OPAC kommt ohne Anmeldung — deshalb ein eigener Kontext, aber
	// dieselbe Messung. Er war eine der beiden Stellen, die Peter nebeneinandergelegt hat.
	await page.context().clearCookies();
	await page.goto(OPAC.pfad);
	await page.locator(`#${OPAC.id}`).waitFor();
	const opacWerte = await fokussiertMessen(page, OPAC.id);
	expect(opacWerte, 'OPAC: Pille nicht messbar').not.toBeNull();
	gemessen.push({ name: OPAC.name, werte: opacWerte });

	// Gegenprobe gegen einen stillen Nulllauf: Ohne sie wäre der Test auch grün, wenn
	// die Schleife nie etwas gefunden hätte.
	expect(gemessen.length).toBe(PILLEN.length + 1);

	const [erste, ...weitere] = gemessen;
	for (const p of weitere) {
		expect(
			p.werte,
			`„${p.name}" weicht von „${erste.name}" ab.\n` +
				`  ${erste.name}: ${JSON.stringify(erste.werte)}\n` +
				`  ${p.name}: ${JSON.stringify(p.werte)}\n` +
				`Suchfelder kommen aus components/ui/Suchpille.svelte — eine eigene Fassung ` +
				`läuft davon weg, wie es zwischen Portal, OPAC und Medienkatalog passiert ist.`
		).toEqual(erste.werte);
	}

	// Und die Pille ist wirklich eine Pille (M3: volle Rundung), nicht das 12-px-Rechteck
	// eines Formularfeldes. Ohne diese Zusage wären auch drei gleich FALSCHE Felder grün.
	expect(Math.round(parseFloat(erste.werte.radius))).toBeGreaterThanOrEqual(24);
	expect(erste.werte.hoehe).toBe(48);
});

test('Im Ruhezustand sind die Pillen gefüllt und randlos — nicht dauerhaft im Fokus-Aussehen', async ({
	page
}) => {
	await uiLogin(page);

	// Hier stehen nur Seiten, deren Pille den Fokus NICHT von selbst nimmt — und diese Liste
	// ist am 11.08.2026 von zwei auf einen Eintrag geschrumpft, weil der zweite eine
	// Falschaussage war:
	//
	// Der Kiosk fehlte schon immer bewusst. Seine Omnibox holt sich den Fokus per Effekt
	// zurück, sobald sie ihn verliert — sie ist ein Scanner-Terminal, ein verlorener Fokus
	// wäre ein verlorener Scan. Sie steht deshalb DAUERHAFT im Fokus-Aussehen.
	//
	// „Mein Portal" stand hier zu Unrecht. Die Pille dort trägt `autofokus`
	// (KollegiumPortal.svelte) und nimmt sich den Fokus beim Mounten. Der Test kam trotzdem
	// durch, weil er sich mit `document.activeElement.blur()` erst den Zustand herstellte,
	// den er messen wollte — einen, den kein Nutzer je zu Gesicht bekommt. Peter ist der
	// blaue Rand auf der Produktion aufgefallen, während dieses Gate grün war. Gemessen am
	// laufenden Stack:
	//
	//   Medienkatalog  Fläche rgb(241,240,244)  Rand transparent      Fokus: nein
	//   Mein Portal    Fläche rgb(255,255,255)  Rand rgb(0,97,164)    Fokus: JA
	//
	// Das `blur()` ist deshalb ersatzlos weg: Was gemessen wird, muss der Zustand beim
	// Laden sein. Für das Portal steht die Zusage jetzt im Test darunter — als das, was sie
	// ist, nämlich Absicht.
	const RUHIG = [{ name: 'Medienkatalog', pfad: '/medienkatalog', id: 'katalog-suchfeld' }];

	for (const { name, pfad, id } of RUHIG) {
		await gehZu(page, pfad);
		await page.locator(`#${id}`).waitFor();

		// Auf einen stabilen Wert warten: focus-within blendet über 200 ms um.
		let vorher = null;
		let werte = null;
		for (let i = 0; i < 20; i++) {
			werte = await page.evaluate(MESSEN, id);
			if (vorher && JSON.stringify(vorher) === JSON.stringify(werte)) break;
			vorher = werte;
			await page.waitForTimeout(50);
		}

		expect(werte.flaeche, `${name}: im Ruhezustand gefüllt, nicht weiß`).not.toBe(
			'rgb(255, 255, 255)'
		);
		expect(werte.randfarbe, `${name}: im Ruhezustand randlos`).toBe('rgba(0, 0, 0, 0)');
	}
});

// Die Gegenprobe zum Test darüber — und der Grund, warum „Mein Portal" dort nicht mehr steht.
//
// Die Pille im Portal sieht beim Laden anders aus als im Medienkatalog, und das ist kein
// Designbruch, sondern ein Zustand: Die Seite hat genau eine Funktion (suchen, um einen
// Klassensatz zu reservieren), deshalb trägt ihre Suchpille `autofokus`. Wer den Fokus
// bekommt, wird weiß und blau umrandet — dasselbe Bauteil, andere Lage.
//
// Dieser Test hält das als ABSICHT fest. Nimmt jemand `autofokus` heraus, wird er rot und
// erzwingt die Entscheidung, statt sie stillschweigend zu kippen.
test('Mein Portal nimmt den Fokus beim Laden — und trägt deshalb bewusst das Fokus-Aussehen', async ({
	page
}) => {
	await uiLogin(page);
	await gehZu(page, '/kollegium-portal');
	await page.locator('#portal-suchfeld').waitFor();

	// Ohne Zutun — kein Klick, kein focus(). Genau darin liegt der Unterschied zum
	// Medienkatalog, und genau das hat der alte Test mit seinem blur() zugedeckt.
	await expect(
		page.locator('#portal-suchfeld'),
		'die Portal-Suchpille holt sich den Fokus selbst (autofokus in KollegiumPortal.svelte)'
	).toBeFocused();

	let vorher = null;
	let werte = null;
	for (let i = 0; i < 20; i++) {
		werte = await page.evaluate(MESSEN, 'portal-suchfeld');
		if (vorher && JSON.stringify(vorher) === JSON.stringify(werte)) break;
		vorher = werte;
		await page.waitForTimeout(50);
	}

	// Bewusst als Gegensatz zum Ruhezustand formuliert und nicht als Palettenwert: WELCHES
	// Blau der Fokusrand trägt, prüft der Vergleichstest weiter oben für alle Pillen
	// gemeinsam. Hier zählt nur, dass diese Pille beim Laden im Fokus-Aussehen steht.
	expect(werte.flaeche, 'im Fokus weiß, nicht gefüllt').toBe('rgb(255, 255, 255)');
	expect(werte.randfarbe, 'im Fokus umrandet, nicht randlos').not.toBe('rgba(0, 0, 0, 0)');
});

for (const { name, pfad, id, anmelden } of MIT_FOKUS) {
	test(`${name}: die Suchpille hat beim Betreten den Fokus`, async ({ page }) => {
		if (anmelden) await uiLogin(page);
		await page.goto(pfad);
		await page.locator(`#${id}`).waitFor();

		// Erst: Bekommt das Feld den Fokus überhaupt, ohne dass jemand hinklickt?
		//
		// Die Frist ist bewusst großzügig. Gemessen am 10.08.2026 zieht der Kiosk den
		// Fokus 35–60 ms nach dem Rendern (Omnibox.svelte holt ihn über ein
		// setTimeout(…, 50), weil dort kein Element-Ref gebunden ist). Das ist für einen
		// Menschen unbemerkbar; die Eigenschaft, um die es geht, ist „das Feld nimmt den
		// ersten Anschlag" und nicht „innerhalb von N Millisekunden". Eine knappe Frist
		// machte den Test auf einem ausgelasteten CI-Rechner flatterig, ohne mehr zu
		// beweisen — bleibt der Fokus GANZ aus, läuft sie ohnehin ab.
		await expect
			.poll(() => page.evaluate(() => document.activeElement?.id ?? ''), { timeout: 2000 })
			.toBe(id);

		// Dann: blind tippen, ohne vorher irgendwo hinzuklicken — so verhält sich ein
		// Barcode-Scanner. Das ist nicht dieselbe Frage wie oben: Zwischen Fokus und
		// erstem Zeichen kann ihn etwas anderes wieder an sich ziehen, und dann ist der
		// Scan verloren, ohne dass jemand eine Fehlermeldung sieht.
		await page.keyboard.type('9783');

		const wo = await page.evaluate(() => ({
			id: document.activeElement?.id ?? '',
			wert: /** @type {HTMLInputElement} */ (document.activeElement)?.value ?? ''
		}));

		expect(
			wo.id,
			`Der erste Anschlag ging ins Leere — er landete bei „${wo.id || '(nichts)'}" statt in #${id}.`
		).toBe(id);
		expect(wo.wert).toContain('9783');
	});
}
