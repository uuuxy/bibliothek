// Messhilfe für e2e/icon-trefferflaechen.spec.js: misst jeden sichtbaren Symbolknopf einer Seite.
//
// Ob die 48 × 48 px um einen Knopf ihm gehören, zeigt nur der Browser: Ein Nachbar, ein
// abschneidender Container oder eine Leiste darüber nimmt ihm einen Streifen, ohne dass sich
// an seiner Klasse etwas ändert. Deshalb fragt die Messung an neun Punkten der Fläche
// document.elementFromPoint und prüft außerdem, dass die Fläche kein anderes Bedienelement
// überdeckt. Punkte außerhalb des Fensters zählen nicht: Dort trifft auch niemand.

/**
 * Läuft im Browser (page.evaluate) und kommt deshalb ohne Import aus. Ohne `messen` tastet sie
 * nichts ab und rollt nichts: Das Warten auf eine ruhige Seite zählt nur und soll keine Liste
 * nachladen lassen.
 * @param {{ ziel: number, messen: boolean }} auftrag ziel ist die Kantenlänge der Fläche in px
 */
export const SYMBOLKNOEPFE = ({ ziel, messen }) => {
	const BEDIENBAR = [
		'a[href]',
		'button',
		'input:not([type="hidden"])',
		'select',
		'textarea',
		'summary',
		'label:has(input)',
		'[role="button"]',
		'[role="checkbox"]',
		'[role="radio"]',
		'[role="switch"]',
		'[role="tab"]',
		'[role="link"]',
		'[role="menuitem"]',
		'[role="option"]'
	].join(', ');
	/** @param {Element} el */
	const sichtbar = (el) => {
		const r = el.getBoundingClientRect();
		return (
			r.width >= 4 &&
			r.height >= 4 &&
			el.checkVisibility({ visibilityProperty: true, opacityProperty: true })
		);
	};
	/** @param {Element} el */
	const name = (el) =>
		el.getAttribute('aria-label') ||
		el.getAttribute('data-tip') ||
		el.getAttribute('title') ||
		(el.textContent || '').trim().slice(0, 40) ||
		el.tagName.toLowerCase();
	const knoepfe = [...document.querySelectorAll('button, [role="button"]')].filter(
		(knopf) =>
			knopf.querySelector('svg') !== null &&
			// Beschriftete Knöpfe misst control-hoehen.spec.js; eine Zahl neben dem Symbol zählt
			// nicht als Beschriftung, ein Wort wie „LMF" schon.
			!/\p{L}/u.test(knopf.textContent || '') &&
			// Die rechte Hälfte eines Split-Buttons gehört zu einem Knopf mit Beschriftung.
			!knopf.hasAttribute('data-split-pfeil') &&
			sichtbar(knopf)
	);
	const bedienelemente = [...document.querySelectorAll(BEDIENBAR)];
	const PUNKTE = /** @type {const} */ ([
		[0, 0, 'Mitte'],
		[-1, -1, 'links oben'],
		[0, -1, 'oben'],
		[1, -1, 'rechts oben'],
		[-1, 0, 'links'],
		[1, 0, 'rechts'],
		[-1, 1, 'links unten'],
		[0, 1, 'unten'],
		[1, 1, 'rechts unten']
	]);
	const ergebnis = [];
	for (const knopf of knoepfe) {
		/** @type {string[]} */
		const fremd = [];
		/** @type {string[]} */
		const ueberdeckt = [];
		// Ziele der Navigation sind keine Symbolknöpfe, sie folgen der Navigation rail.
		const navigation = knopf.closest('nav') !== null;
		const abtasten = messen && !navigation;
		if (abtasten) {
			// Ein Knopf am Rand einer Liste mit eigener Bildlaufleiste ist abgeschnitten, ohne dass
			// checkVisibility es merkt. Gemessen wird er dort, wo man ihn bedient: mitten im Bild.
			knopf.scrollIntoView({ block: 'center', inline: 'nearest', behavior: 'instant' });
		}
		const r = knopf.getBoundingClientRect();
		if (abtasten) {
			const mx = r.left + r.width / 2;
			const my = r.top + r.height / 2;
			const halbB = Math.max(ziel, r.width) / 2;
			const halbH = Math.max(ziel, r.height) / 2;
			for (const [dx, dy, wo] of PUNKTE) {
				const x = mx + dx * (halbB - 1);
				const y = my + dy * (halbH - 1);
				if (x < 0 || y < 0 || x >= innerWidth || y >= innerHeight) continue;
				const getroffen = document.elementFromPoint(x, y);
				if (getroffen && (getroffen === knopf || knopf.contains(getroffen))) continue;
				const wer = getroffen ? name(getroffen.closest(BEDIENBAR) ?? getroffen) : 'nichts';
				fremd.push(`${wo} trifft ${wer}`);
			}
			for (const anderes of bedienelemente) {
				if (anderes === knopf || anderes.contains(knopf) || knopf.contains(anderes)) continue;
				const a = anderes.getBoundingClientRect();
				// Ein Symbol im Feld (Kamera, Löschen) gehört zum Feld: Material 3 legt seine Fläche
				// in den Rand des Textfelds, und der Text endet davor.
				const imFeld =
					anderes.matches('input, textarea, select') &&
					mx > a.left &&
					mx < a.right &&
					my > a.top &&
					my < a.bottom;
				if (imFeld) continue;
				const trennt =
					a.right <= mx - halbB ||
					a.left >= mx + halbB ||
					a.bottom <= my - halbH ||
					a.top >= my + halbH;
				if (!trennt && sichtbar(anderes)) ueberdeckt.push(name(anderes));
			}
		}
		ergebnis.push({
			breite: Math.round(r.width),
			hoehe: Math.round(r.height),
			label: name(knopf),
			klassen: knopf.getAttribute('class') || '',
			radius: Number.parseFloat(getComputedStyle(knopf).borderTopLeftRadius) || 0,
			navigation,
			fremd,
			ueberdeckt
		});
	}
	return ergebnis;
};
