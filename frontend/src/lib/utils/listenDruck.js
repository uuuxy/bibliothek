// utils/listenDruck.js
// Gerüst und Fenster für den Ausdruck einer Liste (Ausleiher eines Titels, Mahnliste).
//
// Das Gerüst nimmt nur Text entgegen und maskiert jede Einsetzstelle selbst. Ein Aufrufer
// kann die Maskierung so nicht vergessen, und eine neue Spalte ist von selbst geschützt.

import { escapeHtml } from './escapeHtml.js';

const STIL = `
  body { font-family: system-ui, -apple-system, sans-serif; padding: 2rem; color: #1e293b; }
  h1 { font-size: 1.5rem; margin-bottom: 0.5rem; }
  p.meta { margin: 0 0 1.5rem 0; color: #64748b; font-size: 0.875rem; }
  table { border-collapse: collapse; width: 100%; margin-top: 1rem; }
  th, td { padding: 0.75rem; text-align: left; border-bottom: 1px solid #e2e8f0; }
  th { background: #f8fafc; font-weight: 600; font-size: 0.875rem; color: #475569; }
  .overdue { color: #e11d48; font-weight: bold; }
  .mono { font-family: monospace; font-size: 0.875rem; }
  .schmal { width: 1%; white-space: nowrap; }
  tr { break-inside: avoid; }
  .dicht { padding: 0.5rem; font-size: 0.75rem; }
  .dicht h1 { font-size: 1.125rem; margin: 0 0 0.25rem 0; }
  .dicht p.meta { margin: 0 0 0.5rem 0; font-size: 0.75rem; }
  .dicht table { margin-top: 0; }
  .dicht th, .dicht td { padding: 0.3rem 0.5rem; font-size: 0.75rem; }
  .dicht td { overflow-wrap: anywhere; }
  @media print { @page { margin: 1cm; } }
`;

/** Die Meldung, wenn der Browser das Druckfenster nicht öffnet. */
export const FENSTER_BLOCKIERT = 'Bitte erlaube Popups, um die Liste zu drucken.';

/**
 * @typedef {string | { text: string, klasse?: string }} Zelle
 *   `klasse` im Rumpf: overdue oder mono. Im Kopf: schmal — die Spalte nimmt nur die
 *   Breite ihres Inhalts und bricht nicht um, der Rest bleibt den Textspalten. Die Klasse
 *   des Kopfs gilt für jede Zelle seiner Spalte.
 */

/** @param {Zelle} zelle */
const klasseVon = (zelle) => (typeof zelle === 'string' ? '' : (zelle.klasse ?? ''));

/** @param {Zelle} zelle @param {'td' | 'th'} marke @param {string} [spaltenKlasse] */
function zelleHtml(zelle, marke, spaltenKlasse = '') {
	const text = typeof zelle === 'string' ? zelle : zelle.text;
	const klasse = [klasseVon(zelle), spaltenKlasse].filter(Boolean).join(' ');
	const attribut = klasse ? ` class="${escapeHtml(klasse)}"` : '';
	return `<${marke}${attribut}>${escapeHtml(text)}</${marke}>`;
}

/**
 * @param {{ ueberschrift: string, meta: string, spalten: Zelle[], zeilen: Zelle[][], dicht?: boolean }} liste
 *   Überschrift und Fenstertitel sind ein Text; `meta` ist die Zeile darunter. `dicht`
 *   setzt eine lange Liste enger (kleinere Schrift, weniger Abstand), damit sie wenige
 *   Seiten braucht.
 * @returns {string} Vollständiges HTML-Dokument
 */
export function baueListenDruckHtml({ ueberschrift, meta, spalten, zeilen, dicht = false }) {
	const titel = escapeHtml(ueberschrift);
	const kopf = spalten.map((s) => zelleHtml(s, 'th')).join('');
	const rumpf = zeilen
		.map(
			(zeile) => `
      <tr>${zeile.map((z, i) => zelleHtml(z, 'td', klasseVon(spalten[i] ?? ''))).join('')}</tr>`
		)
		.join('');

	return `<!DOCTYPE html>
<html>
<head>
  <title>${titel}</title>
  <style>${STIL}</style>
</head>
<body${dicht ? ' class="dicht"' : ''}>
  <h1>${titel}</h1>
  <p class="meta">${escapeHtml(meta)}</p>
  <table>
    <thead>
      <tr>${kopf}</tr>
    </thead>
    <tbody>${rumpf}
    </tbody>
  </table>
</body>
</html>`;
}

/**
 * Schreibt das Dokument in ein eigenes Fenster und druckt es.
 *
 * Gedruckt wird von hier aus und nicht von einem Skript im geschriebenen Dokument: Ein
 * per window.open('') erzeugtes Fenster erbt die CSP des Openers, und die erlaubt nur
 * script-src 'self'. document.write ist synchron und das Dokument lädt nichts nach,
 * deshalb steht der Inhalt beim Druckaufruf schon.
 * @param {string} html
 * @returns {boolean} false, wenn der Browser das Fenster nicht öffnet
 */
export function druckeDokument(html) {
	const fenster = window.open('', '_blank', 'width=800,height=600');
	if (!fenster) return false;
	fenster.document.open();
	fenster.document.write(html);
	fenster.document.close();
	fenster.focus();
	fenster.print();
	return true;
}
