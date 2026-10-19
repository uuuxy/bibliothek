import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad, ohneKommentare } from './hygiene-quellen.js';

// Ratsche: Trägt ein Knopf oder Verweis sichtbaren Text und ein aria-label, steht der sichtbare
// Text im aria-label, Wort für Wort in derselben Reihenfolge.
//
// Material 3, Buttons, Accessibility: „The accessibility label for a button should match the
// visible label text on the button such as Done, Send, or Reply." WCAG 2.1, Understanding 2.5.3
// Label in Name: „the name contains the text that is presented visually". Wer per Sprache
// steuert, nennt, was er sieht; ein aria-label in anderen Worten ist für ihn nicht erreichbar.
//
// e2e/barrierefreiheit-axe.spec.js prüft dasselbe im Browser (axe-Regel
// label-content-name-mismatch), aber nur an den Hauptansichten im Grundzustand. Diese Ratsche
// sieht auch Dialoge, Zustände und Bauteile, die dort nicht offen sind.
//
// Sieht nicht: ein aria-label mit einem Ausdruck (aria-label={…} oder {…} darin), Text aus einem
// Ausdruck im Inhalt (er gilt als unbekannt), Text mit sr-only in einer verschachtelten
// Klassenliste, Knöpfe aus role="button" und Namen über aria-labelledby.

const OEFFNER = /<(button|Button|a)(?=[\s>])/g;

/**
 * Alle Knöpfe und Verweise einer Quelle, die ein festes aria-label tragen, mit Kopf und Inhalt.
 * Der Kopf endet am ersten „>" außerhalb von Klammern und Anführungszeichen: In
 * onclick={() => …} steht selbst eines.
 * @param {string} quelle
 */
function benannte(quelle) {
	/** @type {{ kopf: string, name: string, inhalt: string }[]} */
	const gefunden = [];
	for (const treffer of quelle.matchAll(OEFFNER)) {
		const anfang = treffer.index;
		let i = anfang + treffer[0].length;
		let tiefe = 0;
		let zitat = '';
		for (; i < quelle.length; i++) {
			const z = quelle[i];
			if (zitat) zitat = z === zitat ? '' : zitat;
			else if (z === '"' || z === "'" || z === '`') zitat = z;
			else if (z === '{') tiefe++;
			else if (z === '}') tiefe--;
			else if (z === '>' && tiefe === 0) break;
		}
		const kopf = quelle.slice(anfang, i);
		const name = /\saria-label="([^"{}]*)"/.exec(kopf);
		if (!name || kopf.endsWith('/')) continue;
		const schluss = new RegExp(`</${treffer[1]}\\s*>`, 'g');
		schluss.lastIndex = i;
		const ende = schluss.exec(quelle);
		if (ende) gefunden.push({ kopf, name: name[1], inhalt: quelle.slice(i + 1, ende.index) });
	}
	return gefunden;
}

/** @param {string} text */
const woerter = (text) => text.toLowerCase().match(/[\p{L}\p{N}]+/gu) ?? [];

/**
 * Die Wörter, die im Inhalt am Bildschirm stehen. Tags, Bauteile und Blöcke ({#if}, {:else},
 * {@render}) tragen keins; ein Ausdruck ({anzahl}) ist unbekannt und fällt heraus, ebenso Text
 * mit sr-only, der nur für den Screenreader da ist.
 * @param {string} inhalt
 */
function sichtbareWoerter(inhalt) {
	return woerter(
		inhalt
			.replace(/<(\w+)[^>]*\bclass="[^"]*\bsr-only\b[^"]*"[^>]*>[\s\S]*?<\/\1>/g, ' ')
			.replace(/<(?:=>|[^>])*>/g, ' ')
			.replace(/\{[^}]*\}/g, ' ')
	);
}

/**
 * Stehen die sichtbaren Wörter in derselben Reihenfolge im Namen? Weitere Wörter dürfen
 * dazwischen und dahinter stehen; WCAG verlangt „contains", nicht Gleichheit.
 * @param {string[]} sichtbar
 * @param {string} name
 */
function stehtImNamen(sichtbar, name) {
	const imNamen = woerter(name);
	let i = 0;
	for (const wort of sichtbar) {
		while (i < imNamen.length && imNamen[i] !== wort) i++;
		if (i === imNamen.length) return false;
		i++;
	}
	return true;
}

/** Alle Knöpfe und Verweise mit festem aria-label im Quellbaum, auch die ohne sichtbaren Text. */
function fundstellen() {
	/** @type {{ ort: string, name: string, sichtbar: string[] }[]} */
	const ergebnis = [];
	for (const datei of sammleQuelldateien(srcRoot)) {
		if (!datei.endsWith('.svelte')) continue;
		const quelle = readFileSync(datei, 'utf8');
		let ab = 0;
		for (const b of benannte(ohneKommentare(quelle))) {
			// Die Zeile zählt in der Quelle mit Kommentaren, sonst zeigte die Meldung zu weit oben.
			const stelle = quelle.indexOf(b.kopf, ab);
			if (stelle !== -1) ab = stelle + 1;
			const zeile = stelle === -1 ? '?' : quelle.slice(0, stelle).split('\n').length;
			const sichtbar = sichtbareWoerter(b.inhalt);
			ergebnis.push({ ort: `${relPfad(datei)}:${zeile}`, name: b.name, sichtbar });
		}
	}
	return ergebnis;
}

describe('Der sichtbare Text eines Knopfs steht in seinem Namen', () => {
	it('der Detektor liest Namen und sichtbaren Text', () => {
		const quelle = `
			<Button onclick={() => (offen = !offen)} aria-label="Neuen Leser anlegen"><Plus class="h-4 w-4" />Neuer Leser</Button>
			<button aria-label="Menge erhöhen">+</button>
			<a href="/mahnwesen" aria-label="Zum Mahnwesen">Im Mahnwesen {#if n}({n}){/if}</a
			>
			<Button aria-label={etikett}>Speichern</Button>
			<button aria-label="Drucken: Klassenliste"><span class="sr-only">Liste</span>Drucken</button>
			<ButtonLeiste aria-label="Leiste">kein Knopf</ButtonLeiste>`;
		expect(benannte(quelle).map((b) => [b.name, sichtbareWoerter(b.inhalt)])).toEqual([
			['Neuen Leser anlegen', ['neuer', 'leser']],
			['Menge erhöhen', []],
			['Zum Mahnwesen', ['im', 'mahnwesen']],
			['Drucken: Klassenliste', ['drucken']]
		]);
	});

	it('der Vergleich verlangt die Wörter in ihrer Reihenfolge, weitere dürfen dazukommen', () => {
		expect(stehtImNamen(['alle', 'anzeigen'], 'Alle anzeigen: Ladenhüter')).toBe(true);
		expect(stehtImNamen(['drucken'], 'Drucken')).toBe(true);
		expect(stehtImNamen(['neuer', 'leser'], 'Neuen Leser anlegen')).toBe(false);
		expect(stehtImNamen(['deine', 'bestellung'], 'Bestellung, deine')).toBe(false);
	});

	it('findet Knöpfe mit festem aria-label im Quellbaum', () => {
		// Liefe der Detektor leer, bliebe die Ratsche still grün. Die meisten Treffer sind
		// Symbolknöpfe ohne sichtbaren Text; sie zählen hier, weil der Detektor sie lesen muss.
		expect(fundstellen().length).toBeGreaterThan(30);
	});

	it('jeder Knopf mit sichtbarem Text trägt ihn in seinem Namen', () => {
		expect(
			fundstellen()
				.filter((f) => f.sichtbar.length > 0 && !stehtImNamen(f.sichtbar, f.name))
				.map((f) => `${f.ort}: Name „${f.name}", sichtbar „${f.sichtbar.join(' ')}"`),
			'Der Name eines Knopfs enthält seinen sichtbaren Text (M3 Buttons, Accessibility: ' +
				'„should match the visible label text"). Meist reicht es, das aria-label wegzulassen.'
		).toEqual([]);
	});
});
