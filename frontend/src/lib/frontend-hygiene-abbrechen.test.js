import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import {
	srcRoot,
	sammleQuelldateien,
	relPfad,
	ohneKommentare,
	elemente
} from './hygiene-quellen.js';

// Ratsche: Ein Knopf, der einen Dialog oder ein Formular verlässt, ist nie gesperrt.
//
// Material 3, Dialogs: „Disable confirming actions until a choice is made. Dismissive actions
// are never disabled." Was beim Schließen schon abgeschickt ist, läuft zu Ende und meldet sich
// in der Snackbar (components/ui/dialogAnfrage.svelte.js).
//
// Ein schließender Knopf ist ein <button> oder <Button>, dessen Text oder aria-label mit
// „Abbrechen", „Schließen" oder „Bleiben" beginnt.
//
// Sieht nicht: eine Beschriftung aus einer Variablen und disabled über {...rest}.

const SCHLIESSEND = /^(?:Abbrechen|Schließen|Bleiben)(?!\p{L})/u;

/**
 * Der sichtbare Text eines Knopfs: ohne Blöcke ({#if}, {:else}) und ohne Tags.
 * @param {string} inhalt
 */
function sichtbarerText(inhalt) {
	return inhalt
		.replace(/\{[#:/@][^}]*\}/g, ' ')
		.replace(/<(?:=>|[^>])*>/g, ' ')
		.replace(/\s+/g, ' ')
		.trim();
}

/** @param {{ kopf: string, inhalt: string }} k */
function schliesst(k) {
	const etikett = k.kopf.match(/\saria-label="([^"]*)"/)?.[1] ?? '';
	return SCHLIESSEND.test(sichtbarerText(k.inhalt)) || SCHLIESSEND.test(etikett);
}

/** @param {string} kopf */
const gesperrt = (kopf) => /\sdisabled(?=[\s=>/]|$)/.test(kopf);

function schliessendeKnoepfe() {
	/** @type {{ ort: string, gesperrt: boolean }[]} */
	const treffer = [];
	for (const datei of sammleQuelldateien(srcRoot)) {
		if (!datei.endsWith('.svelte')) continue;
		const quelle = readFileSync(datei, 'utf8');
		for (const tag of /** @type {const} */ (['button', 'Button'])) {
			let ab = 0;
			for (const k of elemente(ohneKommentare(quelle), tag)) {
				// Die Zeile zählt in der Quelle mit Kommentaren, sonst zeigte die Meldung zu weit oben.
				const stelle = quelle.indexOf(k.kopf, ab);
				if (stelle !== -1) ab = stelle + 1;
				if (!schliesst(k)) continue;
				const zeile = stelle === -1 ? '?' : quelle.slice(0, stelle).split('\n').length;
				treffer.push({ ort: `${relPfad(datei)}:${zeile}`, gesperrt: gesperrt(k.kopf) });
			}
		}
	}
	return treffer;
}

describe('Abbrechen ist nie gesperrt', () => {
	it('der Detektor erkennt schließende Knöpfe und ihre Sperre', () => {
		expect(schliesst({ kopf: '<Button variant="ghost"', inhalt: 'Abbrechen' })).toBe(true);
		expect(schliesst({ kopf: '<Button', inhalt: '\n\t\tAbbrechen\n\t' })).toBe(true);
		expect(schliesst({ kopf: '<button aria-label="Schließen"', inhalt: '<X />' })).toBe(true);
		expect(schliesst({ kopf: '<Button', inhalt: 'Bleiben' })).toBe(true);
		expect(schliesst({ kopf: '<Button', inhalt: 'Speichern' })).toBe(false);
		expect(schliesst({ kopf: '<Button', inhalt: 'Schließfach' })).toBe(false);

		expect(gesperrt('<Button onclick={x} disabled={laeuft}')).toBe(true);
		expect(gesperrt('<button disabled')).toBe(true);
		expect(gesperrt('<Button onclick={x}')).toBe(false);
		expect(gesperrt('<Button data-disabled-grund="x"')).toBe(false);
	});

	it('findet die schließenden Knöpfe im Quellbaum', () => {
		// Hieße ein Tag eines Tages anders, bliebe die Ratsche sonst still grün.
		expect(schliessendeKnoepfe().length).toBeGreaterThan(30);
	});

	it('keiner ist gesperrt', () => {
		const funde = schliessendeKnoepfe()
			.filter((k) => k.gesperrt)
			.map((k) => k.ort);
		expect(
			funde,
			'„Abbrechen" bleibt frei, auch während eine Anfrage läuft (M3 Dialogs: „Dismissive ' +
				'actions are never disabled."). Das Ergebnis meldet components/ui/dialogAnfrage.svelte.js.'
		).toEqual([]);
	});
});
