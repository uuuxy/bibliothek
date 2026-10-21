import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad, ohneKommentare } from './hygiene-quellen.js';

// Ratsche: Jeder Knopf, der nur ein Symbol trägt, hat Trefferfläche, Namen und Sprechblase.
//
// Material 3, Icon buttons, Specs: „Extra small and small icon buttons must have a target size
// of 48x48dp or larger to be accessible." Die Fläche gibt .icon-btn (styles/komponenten.css),
// beim <Button variant="symbol"> die Variante. Accessibility: „The accessibility label for icon
// buttons describes the action the button is executing" und „On web, icon buttons should
// display a tooltip with an accessibility label": aria-label und data-tip. title zählt nicht:
// Er zeigt den Hinweis des Betriebssystems spät und bei Bedienung mit der Tastatur gar nicht.
//
// Ein Symbolknopf ist ein <button> oder <Button>, in dem nur Symbole aus @lucide/svelte, der
// Ladekreis oder ein einzelnes Zeichen wie „✕" stehen. Ausgenommen sind die rechte Hälfte eines
// Split-Buttons (data-split-pfeil) und der Schalter (role="switch"). Wer bewusst keine
// Sprechblase zeigt, nennt den Grund in data-tip-eigen.
//
// Sieht nicht: ein Symbol aus einem anderen Bauteil oder über {@render …}, Attribute, die
// über {...rest} kommen, und ob Name und Sprechblase passen. Ob Größe und Abstand im Browser
// stimmen, misst e2e/icon-trefferflaechen.spec.js.

/**
 * Alle Elemente eines Tags mit Kopf und Inhalt. Der Kopf endet am ersten „>" außerhalb von
 * Klammern und Anführungszeichen: In onclick={() => …} steht selbst eines.
 * @param {string} quelle
 * @param {'button' | 'Button'} tag
 */
function elemente(quelle, tag) {
	/** @type {{ start: number, kopf: string, inhalt: string }[]} */
	const gefunden = [];
	for (let start = quelle.indexOf(`<${tag}`); start !== -1;) {
		let i = start + tag.length + 1;
		const istTag = !/[A-Za-z0-9_]/.test(quelle[i] ?? '');
		let tiefe = 0;
		let zitat = '';
		for (; istTag && i < quelle.length; i++) {
			const z = quelle[i];
			if (zitat) zitat = z === zitat ? '' : zitat;
			else if (z === '"' || z === "'" || z === '`') zitat = z;
			else if (z === '{') tiefe++;
			else if (z === '}') tiefe--;
			else if (z === '>' && tiefe === 0) break;
		}
		const ende = quelle.indexOf(`</${tag}>`, i);
		if (istTag && ende !== -1 && quelle[i - 1] !== '/') {
			gefunden.push({ start, kopf: quelle.slice(start, i), inhalt: quelle.slice(i + 1, ende) });
		}
		start = quelle.indexOf(`<${tag}`, i);
	}
	return gefunden;
}

/**
 * Die Namen, unter denen eine Datei Symbole einbindet, und der Ladekreis, der ein Symbol
 * vertritt, solange eine Anfrage läuft.
 * @param {string} quelle
 */
function symbolNamen(quelle) {
	const namen = new Set(['Ladekreis']);
	for (const m of quelle.matchAll(/import\s*\{([^}]*)\}\s*from\s*['"]@lucide\/svelte['"]/g)) {
		for (const teil of m[1].split(',')) {
			const name = teil
				.trim()
				.split(/\s+as\s+/)
				.pop();
			if (name) namen.add(name);
		}
	}
	return namen;
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

/** @param {'button' | 'Button'} tag */
function symbolknoepfe(tag) {
	/** @type {{ ort: string, fehlt: string[] }[]} */
	const treffer = [];
	for (const datei of sammleQuelldateien(srcRoot)) {
		if (!datei.endsWith('.svelte')) continue;
		const quelle = readFileSync(datei, 'utf8');
		const namen = symbolNamen(quelle);
		let ab = 0;
		for (const k of elemente(ohneKommentare(quelle), tag)) {
			// Die Zeile zählt in der Quelle mit Kommentaren, sonst zeigte die Meldung zu weit oben.
			const stelle = quelle.indexOf(k.kopf, ab);
			if (stelle !== -1) ab = stelle + 1;
			if (/\sdata-split-pfeil\b|\srole="switch"/.test(k.kopf)) continue;
			if (!nurSymbol(k.inhalt, namen)) continue;
			const zeile = stelle === -1 ? '?' : quelle.slice(0, stelle).split('\n').length;
			treffer.push({ ort: `${relPfad(datei)}:${zeile}`, fehlt: fehlt(k.kopf, tag) });
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
