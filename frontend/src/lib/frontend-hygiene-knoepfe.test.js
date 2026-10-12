import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad, ohneKommentare } from './hygiene-quellen.js';

// Ratsche: Ein Knopf ohne Fläche ist ein Textknopf oder ein Symbolknopf, nie etwas dazwischen.
//
// ui/Button führt dafür zwei Varianten. ghost steht in der Hauptfarbe (Material 3, Buttons,
// Specs: „Text icon & label: Primary"), denn ohne Fläche gilt: „the label text color must
// always be recognizable from non-button text and elements". symbol trägt nur ein Symbol und
// bleibt in on-surface-variant (Material 3, Icon buttons, Specs: „Standard icon: On surface
// variant"). Vertauscht stünde ein Menü-Symbol in Blau da, oder ein Wort in der Farbe der
// Beschriftungen daneben, das niemand als Knopf erkennt.
//
// Ein Symbolknopf nennt außerdem, was er tut (Material 3, Icon buttons, Accessibility): „The
// accessibility label for icon buttons describes the action the button is executing" und „On
// web, icon buttons should display a tooltip with an accessibility label." Ein Zeichen wie „✕"
// sagt dem Screenreader nichts, und der Maus fehlt ohne Hinweis die Erklärung.
//
// Sieht nicht: eine Variante aus einem Ausdruck (variant={…}), einen von Hand gebauten
// <button> mit denselben Klassen und Inhalt, der über {@render …} in den Knopf kommt. Ob die
// Farbe am Bildschirm ankommt und ob der Hinweis erscheint, misst sie nicht; den Hinweis misst
// e2e/icon-tooltips.spec.js an Knöpfen mit Bild.

/**
 * Alle <Button> einer Quelle mit Kopf und Inhalt. Der Kopf endet am ersten „>"
 * außerhalb von Klammern und Anführungszeichen: In onclick={() => …} steht selbst eines.
 * @param {string} quelle
 */
function knoepfe(quelle) {
	/** @type {{ kopf: string, inhalt: string }[]} */
	const gefunden = [];
	for (let start = quelle.indexOf('<Button'); start !== -1;) {
		let i = start + '<Button'.length;
		const istKnopf = !/[A-Za-z0-9_]/.test(quelle[i] ?? '');
		let tiefe = 0;
		let zitat = '';
		for (; istKnopf && i < quelle.length; i++) {
			const z = quelle[i];
			if (zitat) zitat = z === zitat ? '' : zitat;
			else if (z === '"' || z === "'" || z === '`') zitat = z;
			else if (z === '{') tiefe++;
			else if (z === '}') tiefe--;
			else if (z === '>' && tiefe === 0) break;
		}
		const ende = quelle.indexOf('</Button>', i);
		if (istKnopf && ende !== -1 && quelle[i - 1] !== '/') {
			gefunden.push({ kopf: quelle.slice(start, i), inhalt: quelle.slice(i + 1, ende) });
		}
		start = quelle.indexOf('<Button', i);
	}
	return gefunden;
}

/**
 * Trägt der Inhalt ein Wort? Bauteile und Blöcke ({#if}, {:else}) zählen nicht, ein Ausdruck
 * ({offen ? 'Zu' : 'Auf'}) und jeder Buchstabe schon. Ein einzelnes Zeichen wie „✕" ist ein
 * Symbol.
 * @param {string} inhalt
 */
function hatWort(inhalt) {
	const rest = inhalt.replace(/<(?:=>|[^>])*>/g, ' ').replace(/\{[#:/@][^}]*\}/g, ' ');
	return /\{/.test(rest) || /[\p{L}\p{N}]/u.test(rest);
}

/**
 * Trägt der Kopf eines Knopfs einen Namen für den Screenreader und einen Hinweis für die Maus?
 * @param {string} kopf
 */
function nenntSichSelbst(kopf) {
	return /\saria-label=/.test(kopf) && /\s(?:data-tip|title)=/.test(kopf);
}

/** @param {'ghost' | 'symbol'} variante */
function fundstellen(variante) {
	/** @type {{ ort: string, wort: boolean, kopf: string }[]} */
	const treffer = [];
	for (const datei of sammleQuelldateien(srcRoot)) {
		if (!datei.endsWith('.svelte')) continue;
		const quelle = readFileSync(datei, 'utf8');
		let ab = 0;
		for (const k of knoepfe(ohneKommentare(quelle))) {
			// Die Zeile zählt in der Quelle mit Kommentaren, sonst zeigte die Meldung zu weit oben.
			const stelle = quelle.indexOf(k.kopf, ab);
			if (stelle !== -1) ab = stelle + 1;
			if (!k.kopf.includes(`variant="${variante}"`)) continue;
			const zeile = stelle === -1 ? '?' : quelle.slice(0, stelle).split('\n').length;
			treffer.push({ ort: `${relPfad(datei)}:${zeile}`, wort: hatWort(k.inhalt), kopf: k.kopf });
		}
	}
	return treffer;
}

describe('Textknopf und Symbolknopf sind zwei Varianten', () => {
	it('der Detektor trennt Wort und Symbol', () => {
		const quelle = `
			<Button variant="ghost" onclick={() => (offen = !offen)}>Abbrechen</Button>
			<Button variant="ghost" disabled={a > b}>{laedt ? 'Wird geholt …' : 'Cover neu holen'}</Button>
			<Button variant="ghost" title="Sicherung übernehmen"><Upload size={16} /> Sicherung einspielen</Button>
			<Button variant="symbol" aria-label="Zeile {nummer} nach oben"><ArrowUp class="h-4 w-4" /></Button>
			<Button variant="symbol">{#if offen}<X />{:else}<Menu />{/if}</Button>
			<Button variant="symbol" onclick={() => (bearbeiten = false)}>✕</Button>
			<ButtonLeiste variant="ghost">kein Knopf</ButtonLeiste>`;
		expect(knoepfe(quelle).map((k) => hatWort(k.inhalt))).toEqual([
			true,
			true,
			true,
			false,
			false,
			false
		]);
	});

	it('kein Textknopf (ghost) trägt nur ein Symbol', () => {
		expect(
			fundstellen('ghost')
				.filter((f) => !f.wort)
				.map((f) => f.ort),
			'Ein Knopf, der nur ein Symbol trägt, ist variant="symbol": Als ghost stünde das ' +
				'Symbol in der Hauptfarbe (M3 Icon buttons: „Standard icon: On surface variant").'
		).toEqual([]);
	});

	it('kein Symbolknopf (symbol) trägt ein Wort', () => {
		expect(
			fundstellen('symbol')
				.filter((f) => f.wort)
				.map((f) => f.ort),
			'Ein Knopf mit Wort ist variant="ghost": Als symbol stünde das Wort in der Farbe der ' +
				'Beschriftungen und wäre nicht als Knopf zu erkennen (M3 Buttons: „Text icon & label: Primary").'
		).toEqual([]);
	});

	it('findet beide Varianten im Quellbaum', () => {
		// Hieße eine Variante eines Tages anders, bliebe die Ratsche sonst still grün.
		expect(fundstellen('ghost').length).toBeGreaterThan(0);
		expect(fundstellen('symbol').length).toBeGreaterThan(0);
	});

	it('der Detektor erkennt Namen und Hinweis in jeder Form', () => {
		const mit = [
			'<Button variant="symbol" aria-label="Abbrechen" data-tip="Abbrechen"',
			'<Button\n\tvariant="symbol"\n\taria-label={etikett}\n\tdata-tip={etikett}',
			'<Button variant="symbol" title="Nach oben" aria-label="Zeile {nummer} nach oben"'
		];
		const ohne = [
			'<Button variant="symbol" size="sm" onclick={() => (bearbeiten = false)}',
			'<Button variant="symbol" title="Abbrechen"',
			'<Button variant="symbol" aria-label="Abbrechen"',
			'<Button variant="symbol" data-aria-label="x" data-tip="x"'
		];
		expect(mit.map(nenntSichSelbst)).toEqual([true, true, true]);
		expect(ohne.map(nenntSichSelbst)).toEqual([false, false, false, false]);
	});

	it('jeder Symbolknopf nennt, was er tut', () => {
		expect(
			fundstellen('symbol')
				.filter((f) => !nenntSichSelbst(f.kopf))
				.map((f) => f.ort),
			'Ein Symbolknopf braucht aria-label für den Screenreader und data-tip, die Sprechblase ' +
				'für die Maus (M3 Icon buttons: „display a tooltip with an accessibility label").'
		).toEqual([]);
	});
});
