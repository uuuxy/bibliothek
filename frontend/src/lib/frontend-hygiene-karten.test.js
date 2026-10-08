import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import {
	srcRoot,
	sammleQuelldateien,
	relPfad,
	vergleicheMitBestand,
	ohneKommentare
} from './hygiene-quellen.js';

// Ratsche: Ein umrandeter Kasten trägt keinen Flächenton.
//
// Material 3, Cards, Specs, kennt drei Karten: umrandet („Surface" mit „Outline variant"),
// gefüllt („Surface container highest", ohne Rahmen) und erhöht („Surface container low" mit
// Schatten). Ein Grauton mit Rahmen ist keine der drei. Im Haus ist die Karte umrandet und
// zeigt die Fläche der Seite; bg-surface und bg-surface-container-lowest zählen deshalb
// nicht als Ton.
//
// Erkannt wird eine Klassenliste mit ganzem Rahmen in einer Umriss-Rolle, einer Ecke ab
// rounded-lg und einem Innenabstand, also ein Kasten mit Inhalt. Ein Bedienelement (m3-state,
// hover:bg-…) ist kein Kasten, und ein gestrichelter Rahmen heißt „hier gehört etwas hin".
//
// Sieht nicht: Klassen aus Variablen oder Ausdrücken, einen Rahmen aus einzelnen Seiten
// (border-t …), einen Ton, den ein Elternteil oder eine CSS-Datei gibt, und ob die Fläche
// dahinter selbst getönt ist — das misst nur der Browser.
const TON = /^bg-surface-(?:container(?:-low|-high|-highest)?|variant)$/;
const ECKE = /^rounded-(?:lg|xl|2xl|3xl|4xl)$/;
const INNENABSTAND = /^p[xy]?-\d/;

/** @param {string} klassenliste */
function istGetoenterKasten(klassenliste) {
	const k = klassenliste.split(/\s+/).filter(Boolean);
	const hat = (/** @type {RegExp} */ muster) => k.some((x) => muster.test(x));
	return (
		k.includes('border') &&
		hat(/^border-outline/) &&
		!k.includes('border-dashed') &&
		hat(ECKE) &&
		hat(INNENABSTAND) &&
		!k.includes('m3-state') &&
		!hat(/^hover:bg-/) &&
		hat(TON)
	);
}

// Was bleibt, ist kein Kasten mit Inhalt: Die zwei Schienen im Ausweis-Designer halten je
// zwei Umschaltknöpfe zusammen; die Fläche gehört zum Bedienelement.
const BESTAND = ['src/lib/designer/Toolbar.svelte', 'src/lib/designer/ToolbarDruck.svelte'];

describe('Ein umrandeter Kasten trägt keinen Flächenton', () => {
	it('der Detektor trennt Kasten, Bedienelement und Platzhalter', () => {
		const rahmen = 'rounded-xl border border-outline-variant';
		expect(istGetoenterKasten(`mt-3 p-4 ${rahmen} bg-surface-container-low`)).toBe(true);
		expect(istGetoenterKasten(`px-4 py-3 ${rahmen} bg-surface-container`)).toBe(true);
		// umrandete Karte: Fläche der Seite, Surface oder Weiß
		expect(istGetoenterKasten(`mt-3 p-4 ${rahmen}`)).toBe(false);
		expect(istGetoenterKasten(`p-5 ${rahmen} bg-surface`)).toBe(false);
		expect(istGetoenterKasten(`p-4 ${rahmen} bg-surface-container-lowest`)).toBe(false);
		// Ton ohne Rahmen, Rahmen nur an einer Seite
		expect(istGetoenterKasten('p-4 rounded-xl bg-surface-container-low')).toBe(false);
		expect(
			istGetoenterKasten('px-6 py-4 border-t border-outline-variant bg-surface-container-low')
		).toBe(false);
		// Bedienelement, Ablagefläche, Platzhalter ohne Inhalt
		expect(istGetoenterKasten(`m3-state px-3 py-2 ${rahmen} bg-surface-container-low`)).toBe(false);
		expect(
			istGetoenterKasten(
				`px-5 py-2.5 ${rahmen} bg-surface-container hover:bg-surface-container-high`
			)
		).toBe(false);
		expect(istGetoenterKasten(`p-4 ${rahmen} border-dashed bg-surface-container`)).toBe(false);
		expect(istGetoenterKasten(`h-20 w-16 ${rahmen} bg-surface-container-low`)).toBe(false);
	});

	it('kein neuer Kasten mit Rahmen und Flächenton', () => {
		/** @type {string[]} */
		const betroffen = [];
		let gelesen = 0;
		for (const datei of sammleQuelldateien(srcRoot)) {
			if (!datei.endsWith('.svelte')) continue;
			gelesen++;
			const quelle = ohneKommentare(readFileSync(datei, 'utf8'));
			const listen = [...quelle.matchAll(/class="([^"]*)"/g)].map((m) => m[1]);
			if (listen.some(istGetoenterKasten)) betroffen.push(relPfad(datei));
		}
		// Läse der Sammler keine Bauteile mehr, bliebe die Ratsche still grün.
		expect(gelesen).toBeGreaterThan(100);

		const { neu, inzwischenSauber } = vergleicheMitBestand(betroffen, BESTAND);
		expect(
			neu,
			'Kasten mit Rahmen und Flächenton. Material 3 kennt die Karte umrandet („Surface" mit ' +
				'„Outline variant") oder gefüllt (ohne Rahmen): den Ton weglassen, der Rahmen bleibt.'
		).toEqual([]);
		expect(inzwischenSauber, 'Trägt keinen Ton mehr — aus BESTAND austragen.').toEqual([]);
	});
});
