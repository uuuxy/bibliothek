/**
 * @file format.js
 * Zahlen, Prozent und Zeitpunkte in EINER deutschen Schreibweise.
 *
 * Anlass (Rundgang 01.09.2026): Die Statistik zeigte „23.94 %" (roh vom Backend)
 * neben „0,00 €" (de-DE), das Logbuch „1.9.2026, 21:05:03", die Inventur
 * „01.09.2026". Drei Formate für dieselbe Sache auf drei Bildschirmen — hier ist
 * die eine Stelle, an der sie entstehen.
 */

/** @param {number | null | undefined} wert */
export function formatZahl(wert) {
	return (wert ?? 0).toLocaleString('de-DE');
}

/**
 * „23,9 %" — mit geschütztem Leerzeichen vor dem Zeichen (DIN 5008).
 * @param {number | string | null | undefined} wert
 */
export function formatProzent(wert) {
	const zahl = typeof wert === 'string' ? parseFloat(wert) : (wert ?? 0);
	return (
		(Number.isFinite(zahl) ? zahl : 0).toLocaleString('de-DE', { maximumFractionDigits: 1 }) + ' %'
	);
}

/**
 * „01.09.2026" — immer zweistellig, damit Spalten bündig bleiben.
 * @param {string | number | Date | null | undefined} wert
 */
export function formatDatum(wert) {
	if (!wert) return '';
	const d = new Date(wert);
	if (Number.isNaN(d.getTime())) return '';
	return d.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit', year: 'numeric' });
}

/**
 * „01.09.2026, 21:05" — Datum wie formatDatum, Uhrzeit ohne Sekunden.
 * @param {string | number | Date | null | undefined} wert
 */
export function formatZeitpunkt(wert) {
	if (!wert) return '';
	const d = new Date(wert);
	if (Number.isNaN(d.getTime())) return '';
	return d.toLocaleString('de-DE', {
		day: '2-digit',
		month: '2-digit',
		year: 'numeric',
		hour: '2-digit',
		minute: '2-digit'
	});
}
