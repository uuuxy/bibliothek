import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import {
	srcRoot,
	sammleQuelldateien,
	relPfad,
	ohneKommentare,
	elemente
} from './hygiene-quellen.js';

// Ratsche: Jeder Knopf, der nur ein Symbol trägt, hat Trefferfläche, Namen und Sprechblase.
//
// Material 3, Icon buttons, Specs: „Extra small and small icon buttons must have a target size
// of 48x48dp or larger to be accessible." Die Fläche gibt .icon-btn (styles/komponenten.css),
// beim <Button variant="symbol"> die Variante. Accessibility: „The accessibility label for icon
// buttons describes the action the button is executing" und „On web, icon buttons should
// display a tooltip with an accessibility label": aria-label und data-tip. title zählt nicht:
// Er zeigt den Hinweis des Betriebssystems spät und bei Bedienung mit der Tastatur gar nicht.
//
// Ein Symbolknopf ist ein <button> oder <Button>, in dem nur ein ui/Icon, der Ladekreis oder ein
// einzelnes Zeichen wie „✕" stehen. Ausgenommen sind die rechte Hälfte eines
// Split-Buttons (data-split-pfeil) und der Schalter (role="switch"). Wer bewusst keine
// Sprechblase zeigt, nennt den Grund in data-tip-eigen.
//
// Sieht nicht: ein Symbol aus einem anderen Bauteil oder über {@render …}, Attribute, die
// über {...rest} kommen, und ob Name und Sprechblase passen. Ob Größe und Abstand im Browser
// stimmen, misst e2e/icon-trefferflaechen.spec.js.

/**
 * Die Bauteile, die ein Symbol zeichnen: ui/Icon, das Icon der Navigation und der Ladekreis, der
 * ein Symbol vertritt, solange eine Anfrage läuft.
 */
function symbolNamen() {
	return new Set(['Icon', 'NavIcon', 'Ladekreis']);
}

/**
 * Trägt der Inhalt nur Symbole? Blöcke ({#if}, {:else}, {@render}) zählen nicht, ein Ausdruck
 * ({anzahl}) und jeder Buchstabe machen eine Beschriftung, ein fremdes Bauteil auch.
 * @param {string} inhalt
 * @param {Set<string>} namen
 */
function nurSymbol(inhalt, namen) {
	const ohneBloecke = inhalt.replace(/\{[#:/@][^}]*\}/g, ' ');
	const tags = [...ohneBloecke.matchAll(/<([A-Za-z][\w.]*)/g)].map((m) => m[1]);
	const text = ohneBloecke.replace(/<(?:=>|[^>])*>/g, ' ');
	if (text.includes('{') || /[\p{L}\p{N}]/u.test(text)) return false;
	if (tags.length === 0) return text.trim().length > 0;
	return tags.every((t) => namen.has(t));
}

/**
 * Was dem Kopf eines Symbolknopfs fehlt.
 * @param {string} kopf
 * @param {'button' | 'Button'} tag
 */
function fehlt(kopf, tag) {
	const flaeche = /icon-btn/.test(kopf) || (tag === 'Button' && /\svariant="symbol"/.test(kopf));
	return [
		...(flaeche ? [] : ['.icon-btn']),
		...(/\saria-label=/.test(kopf) ? [] : ['aria-label']),
		...(/\sdata-tip(?:-eigen)?=/.test(kopf) ? [] : ['data-tip'])
	];
}

// Ein Symbolknopf, der löscht oder entfernt, wird beim Zeigen rot: Abweichung von M3 mit Grund
// in docs/sweeps.md (Interaction states). Die Regel steht einmal in ui/Button (loeschen); erkannt
// wird ein solcher Knopf am Namen („löschen“, „entfernen“) oder am Symbol delete und link_off.
const LOESCHT = /l(?:ö|oe)schen|entfernen/i;
const LOESCH_SYMBOL = /<Icon name="(?:delete|link_off)"/;
/** Knöpfe, die löschen und trotzdem nicht rot werden, mit Grund. */
const KEIN_LOESCHKNOPF = {
	// Das Kreuz gehört zum Eingabe-Chip und behält beim Zeigen seine Farbe (material-web, input
	// chip: hover-trailing-icon-color gleich trailing-icon-color).
	'src/lib/components/ui/ChipFeld.svelte': 'Kreuz des Eingabe-Chips',
	'src/inventur/lib/components/admin/KlassensatzKlassenwahl.svelte': 'Kreuz des Eingabe-Chips',
	// „Endgültig löschen“ steht schon auf einer Fläche in der Fehlerfarbe (error-container).
	'src/lib/components/leser/Papierkorb.svelte': 'Fläche in error-container'
};

/** @param {string} kopf @param {string} inhalt */
function loeschtEtwas(kopf, inhalt) {
	const namen = [...kopf.matchAll(/\s(?:aria-label|data-tip)=("[^"]*"|\{[^}]*\})/g)]
		.map((m) => m[1])
		.join(' ');
	return LOESCHT.test(namen) || LOESCH_SYMBOL.test(inhalt);
}

/** @param {string} kopf */
function traegtLoeschen(kopf) {
	return (
		/^<Button\b/.test(kopf) && /\svariant="symbol"/.test(kopf) && /\sloeschen(?![\w=])/.test(kopf)
	);
}

/** @param {'button' | 'Button'} tag */
function symbolknoepfe(tag) {
	/** @type {{ ort: string, fehlt: string[], kopf: string, inhalt: string }[]} */
	const treffer = [];
	for (const datei of sammleQuelldateien(srcRoot)) {
		if (!datei.endsWith('.svelte')) continue;
		const quelle = readFileSync(datei, 'utf8');
		const namen = symbolNamen();
		let ab = 0;
		for (const k of elemente(ohneKommentare(quelle), tag)) {
			// Die Zeile zählt in der Quelle mit Kommentaren, sonst zeigte die Meldung zu weit oben.
			const stelle = quelle.indexOf(k.kopf, ab);
			if (stelle !== -1) ab = stelle + 1;
			if (/\sdata-split-pfeil\b|\srole="switch"/.test(k.kopf)) continue;
			if (!nurSymbol(k.inhalt, namen)) continue;
			const zeile = stelle === -1 ? '?' : quelle.slice(0, stelle).split('\n').length;
			treffer.push({
				ort: `${relPfad(datei)}:${zeile}`,
				fehlt: fehlt(k.kopf, tag),
				kopf: k.kopf,
				inhalt: k.inhalt
			});
		}
	}
	return treffer;
}

describe('Symbolknöpfe haben Trefferfläche, Namen und Sprechblase', () => {
	const namen = new Set(['X', 'Menu', 'RefreshCw', 'Ladekreis']);

	it('der Detektor erkennt Symbolknöpfe in jeder Form', () => {
		const symbol = [
			'<X class="h-4 w-4" aria-hidden="true" />',
			'{#if offen}<X />{:else}<Menu />{/if}',
			'✕',
			'{#if laeuft}<Ladekreis size="md" />{:else}<RefreshCw class="h-5 w-5" />{/if}'
		];
		const beschriftet = [
			'Abbrechen',
			'<X class="h-4 w-4" /> Entfernen',
			'<X /> {anzahl}',
			'<AuflagenZeile auflage={t.auflage} />',
			'<span class="sr-only">Schließen</span><X />',
			'{@render children?.()}'
		];
		expect(symbol.map((s) => nurSymbol(s, namen))).toEqual([true, true, true, true]);
		expect(beschriftet.map((s) => nurSymbol(s, namen))).toEqual([
			false,
			false,
			false,
			false,
			false,
			false
		]);
	});

	it('der Detektor findet, was dem Kopf fehlt', () => {
		expect(
			fehlt('<button class="icon-btn" aria-label="Schließen" data-tip="Schließen"', 'button')
		).toEqual([]);
		expect(
			fehlt('<Button variant="symbol" aria-label={etikett} data-tip={etikett}', 'Button')
		).toEqual([]);
		expect(
			fehlt('<button class={a ? "" : "icon-btn"} aria-label="x" data-tip-eigen="Grund"', 'button')
		).toEqual([]);
		expect(fehlt('<button class="h-8 w-8" aria-label="x" title="x"', 'button')).toEqual([
			'.icon-btn',
			'data-tip'
		]);
		expect(fehlt('<Button variant="secondary" data-tip="x"', 'Button')).toEqual([
			'.icon-btn',
			'aria-label'
		]);
		expect(fehlt('<button class="icon-btn" data-aria-label="x" data-tip="x"', 'button')).toEqual([
			'aria-label'
		]);
	});

	it('findet Symbolknöpfe beider Arten im Quellbaum', () => {
		// Hieße ein Tag eines Tages anders, bliebe die Ratsche sonst still grün.
		expect(symbolknoepfe('button').length).toBeGreaterThan(20);
		expect(symbolknoepfe('Button').length).toBeGreaterThan(5);
	});

	it('jedem Symbolknopf fehlt nichts', () => {
		const funde = [...symbolknoepfe('button'), ...symbolknoepfe('Button')]
			.filter((k) => k.fehlt.length > 0)
			.map((k) => `${k.ort} fehlt ${k.fehlt.join(', ')}`);
		expect(
			funde,
			'Ein Symbolknopf braucht .icon-btn (48 px Trefferfläche), aria-label für den ' +
				'Screenreader und data-tip, die Sprechblase für die Maus (M3 Icon buttons).'
		).toEqual([]);
	});
});

describe('Lösch-Symbolknöpfe werden beim Zeigen rot', () => {
	const alle = () => [...symbolknoepfe('button'), ...symbolknoepfe('Button')];
	const datei = (/** @type {string} */ ort) => ort.replace(/:[^:]*$/, '');

	it('der Detektor erkennt Löschen am Namen und am Symbol', () => {
		expect(loeschtEtwas('<button aria-label="Exemplar löschen"', '<Icon name="delete" />')).toBe(
			true
		);
		expect(
			loeschtEtwas('<Button aria-label="x" data-tip="Entfernen"', '<Icon name="close" />')
		).toBe(true);
		expect(loeschtEtwas('<button aria-label={`${a} lösen`}', '<Icon name="link_off" />')).toBe(
			true
		);
		expect(
			loeschtEtwas('<button aria-label="Schließen" data-tip="Schließen"', '<Icon name="close" />')
		).toBe(false);
		expect(traegtLoeschen('<Button variant="symbol" size="sm" loeschen onclick={x}')).toBe(true);
		expect(traegtLoeschen('<Button\n\tvariant="symbol"\n\tloeschen\n')).toBe(true);
		expect(traegtLoeschen('<Button variant="symbol" loeschenNach={x}')).toBe(false);
		expect(traegtLoeschen('<button class="icon-btn hover:text-error" loeschen')).toBe(false);
	});

	it('jeder Lösch-Symbolknopf ist ein Button variant="symbol" mit loeschen', () => {
		const knoepfe = alle().filter(
			(k) => loeschtEtwas(k.kopf, k.inhalt) && !(datei(k.ort) in KEIN_LOESCHKNOPF)
		);
		// Fände der Detektor keinen, bliebe die Ratsche still grün.
		expect(knoepfe.length).toBeGreaterThan(10);
		expect(
			knoepfe.filter((k) => !traegtLoeschen(k.kopf)).map((k) => k.ort),
			'Ein Symbolknopf, der löscht oder entfernt, ist <Button variant="symbol" loeschen>: ' +
				'Er wird beim Zeigen rot (docs/sweeps.md, Interaction states).'
		).toEqual([]);
	});

	it('loeschen steht nur an Knöpfen, die löschen, und jede Ausnahme trifft noch einen', () => {
		expect(
			alle()
				.filter((k) => traegtLoeschen(k.kopf) && !loeschtEtwas(k.kopf, k.inhalt))
				.map((k) => k.ort)
		).toEqual([]);
		const getroffen = new Set(
			alle()
				.filter((k) => loeschtEtwas(k.kopf, k.inhalt))
				.map((k) => datei(k.ort))
		);
		expect(Object.keys(KEIN_LOESCHKNOPF).filter((d) => !getroffen.has(d))).toEqual([]);
	});
});
