// utils/ausleiherDruck.js
// Baut das Druckdokument der Ausleiher-Liste.
//
// Steht bewusst als eigene Funktion neben der Komponente und nicht in ihr: Nur so
// lässt sich das FERTIGE Dokument prüfen. Ein Test des Maskier-Helfers allein belegt
// nichts über diesen Pfad — er belegt nur, dass der Helfer maskiert, nicht, dass er
// an jeder Einsetzstelle auch aufgerufen wird. Genau diese Lücke war der Befund
// (ausleiherDruck.test.js hält sie zu).

import { escapeHtml } from './escapeHtml.js';
import { fmtDateDE } from './dates.js';

const STIL = `
  body { font-family: system-ui, -apple-system, sans-serif; padding: 2rem; color: #1e293b; }
  h1 { font-size: 1.5rem; margin-bottom: 0.5rem; }
  p.meta { margin: 0 0 1.5rem 0; color: #64748b; font-size: 0.875rem; }
  table { border-collapse: collapse; width: 100%; margin-top: 1rem; }
  th, td { padding: 0.75rem; text-align: left; border-bottom: 1px solid #e2e8f0; }
  th { background: #f8fafc; font-weight: 600; font-size: 0.875rem; color: #475569; }
  .overdue { color: #e11d48; font-weight: bold; }
  @media print { @page { margin: 1cm; } }
`;

/**
 * @param {any[]} ausleiher Bereits gefilterte Zeilen
 * @param {any} buch
 * @param {string} filterKlasse
 * @param {Date} [jetzt] Vergleichszeitpunkt für die Überfälligkeit (Tests setzen ihn)
 * @returns {string} Vollständiges HTML-Dokument
 */
export function baueAusleiherDruckHtml(ausleiher, buch, filterKlasse, jetzt = new Date()) {
	const buchTitel = escapeHtml(buch?.title || 'Buch');
	const druckDatum = jetzt.toLocaleDateString('de-DE');

	const zeilen = ausleiher
		.map((b) => {
			const ueberfaellig = new Date(b.rueckgabe_frist) < jetzt;
			return `
        <tr>
          <td>${escapeHtml(b.schueler_name)} ${escapeHtml(b.schueler_nachname)}</td>
          <td>${escapeHtml(b.klasse || '-')}</td>
          <td style="font-family: monospace; font-size: 0.875rem;">${escapeHtml(b.exemplar_barcode)}</td>
          <td>${escapeHtml(fmtDateDE(b.ausgeliehen_am))}</td>
          <td class="${ueberfaellig ? 'overdue' : ''}">${escapeHtml(fmtDateDE(b.rueckgabe_frist))}</td>
        </tr>`;
		})
		.join('');

	return `<!DOCTYPE html>
<html>
<head>
  <title>Mahnliste: ${buchTitel}</title>
  <style>${STIL}</style>
</head>
<body>
  <h1>Ausleiher-Liste: ${buchTitel}</h1>
  <p class="meta">Erstellt am: ${druckDatum} | Filter: Klasse ${escapeHtml(filterKlasse)}</p>
  <table>
    <thead>
      <tr>
        <th>Schüler/in</th>
        <th>Klasse</th>
        <th>Exemplar</th>
        <th>Ausgeliehen am</th>
        <th>Rückgabe bis</th>
      </tr>
    </thead>
    <tbody>${zeilen}
    </tbody>
  </table>
</body>
</html>`;
}
