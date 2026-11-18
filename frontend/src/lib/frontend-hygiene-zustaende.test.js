import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad, ohneKommentare } from './hygiene-quellen.js';

// Ratsche: Zustände nach Material 3 (States), am Quelltext.
//
// - Zeigen und Drücken tauschen keine Farben. M3, State layers: „To transition from an enabled
//   style to a stateful style requires the addition of a state layer" mit der Farbe des Inhalts.
//   Die Schicht und die Welle beim Drücken kommen aus styles/komponenten.css und
//   utils/welle.js; eine Fläche, die beim Zeigen die Farbe wechselt, ein Symbol, das beim
//   Zeigen rot wird, oder ein Knopf, der beim Drücken schrumpft, steht daneben. Erlaubt ist die
//   Schicht selbst als Fläche mit 8 % (hover:bg-…/8, Kästchen und Radio nach material-web).
// - Den Tastaturfokus zeichnet der Ring nach Googles md-focus-ring in komponenten.css; ein
//   eigener Ring oder Umriss an einer Stelle wiche in Farbe, Breite und Abstand ab.
//
// Sieht nicht: Klassen aus einer Variablen, die woanders zusammengesetzt werden, und Stile in
// <style>-Blöcken. Ob Ring und Schicht im Browser ankommen, misst e2e/zustaende.spec.js.

/** Wo Googles Token beim Zeigen eine Farbe nennt, mit der Stelle des Tokens. */
const TOKEN = {
	// md.comp.switch: selected-hover-handle-color primary-container, unselected-hover-handle-color
	// on-surface-variant; dazu die Schicht um den Griff.
	'src/lib/components/ui/Switch.svelte':
		/^group-(hover|focus-visible|active):(bg-(primary-container|on-surface-variant)|opacity-(8|10))$/,
	// md.comp.primary-navigation-tab: inactive hover label-text-color on-surface.
	'src/lib/components/ui/Reiter.svelte': /^hover:text-on-surface$/,
	// md.comp.navigation-drawer: inactive-hover-, -focus-, -pressed-label-text-color on-surface.
	'src/lib/components/layout/SeitenleisteZiele.svelte':
		/^(hover|focus-visible|active):text-on-surface$/,
	// Der Ausweis-Designer umrandet beim Zeigen wie Googles Editoren das Element unter der Maus.
	'src/lib/designer/CanvasElement.svelte': /^hover:ring(-[a-z0-9-]+)?$/,
	// md.comp.elevated-card: container-elevation level1, hover-container-elevation level2.
	'src/inventur/lib/components/admin/KlassensatzTitelRaster.svelte': /^hover:shadow-xl$/
};

const VERBOTEN =
	/(?<![\w-])((?:group-)?(?:hover|active):(?:bg|text|opacity|shadow|scale|border|ring|from|via|to)(?:-[\w./[\]-]+)?)/g;
// Die Schicht als Fläche mit 8 %, eine beim Zeigen eingeblendete Aktion wie in Gmail und der
// Rahmen des Textfelds (md.comp.outlined-text-field: hover-outline-color on-surface,
// error-hover-outline-color on-error-container).
const ERLAUBT =
	/^(hover:bg-[a-z-]+\/8|group-hover:opacity-100|hover:border-(on-surface|on-error-container))$/;
const EIGENER_RING =
	/(?<![\w-])((?:peer-|group-)?focus(?:-visible|-within)?:(?:ring|outline)-(?!none\b)[\w/[\]-]+)/g;
// Ein Ring um ein Textfeld ist sein Rahmen im Fokus (outlined text field: focus-outline 2 px).
const FELDRAHMEN = /^focus(?:-visible|-within)?:ring-(1|primary|error|0)$/;

describe('Zustände nach M3 am Quelltext', () => {
	const dateien = sammleQuelldateien(srcRoot).filter((f) => f.endsWith('.svelte'));

	it('tauscht beim Zeigen und Drücken keine Farben', () => {
		/** @type {string[]} */
		const funde = [];
		for (const f of dateien) {
			const pfad = relPfad(f);
			const quelle = ohneKommentare(readFileSync(f, 'utf8'));
			for (const [, klasse] of quelle.matchAll(VERBOTEN)) {
				if (ERLAUBT.test(klasse) || TOKEN[pfad]?.test(klasse)) continue;
				funde.push(`${pfad}: ${klasse}`);
			}
		}
		expect(
			funde,
			'Farbwechsel beim Zeigen oder Drücken: Die Schicht (button, .m3-state) und die Welle zeigen den Zustand'
		).toEqual([]);
	});

	it('zeichnet keinen eigenen Fokusring', () => {
		/** @type {string[]} */
		const funde = [];
		for (const f of dateien) {
			const quelle = ohneKommentare(readFileSync(f, 'utf8'));
			for (const [, klasse] of quelle.matchAll(EIGENER_RING))
				if (!FELDRAHMEN.test(klasse)) funde.push(`${relPfad(f)}: ${klasse}`);
		}
		expect(funde, 'eigener Fokusring: den Ring aus styles/komponenten.css stehen lassen').toEqual(
			[]
		);
	});

	it('findet die Stellen, die es prüft', () => {
		expect(dateien.length).toBeGreaterThan(200);
		const css = readFileSync(`${srcRoot}/styles/komponenten.css`, 'utf8');
		expect(css).toContain('outline: 3px solid var(--color-secondary)');
		expect(css).toContain('outline-offset: -3px');
		expect(Object.keys(TOKEN).every((p) => dateien.some((f) => relPfad(f) === p))).toBe(true);
	});
});
