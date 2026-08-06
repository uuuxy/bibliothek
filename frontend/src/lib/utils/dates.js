// utils/dates.js
// Lokale Datumsformatierung für Berichts-Zeiträume.
// Niemals über toISOString() gehen — das konvertiert nach UTC und kippt
// in UTC+x-Zeitzonen auf den Vortag (Monatsberichte verloren so den letzten Tag).

/**
 * Lokales Datum als YYYY-MM-DD.
 * @param {Date} d
 */
export function localISO(d) {
	return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

/**
 * Datum in deutscher Schreibweise; '-' für leere Werte, Rohwert für unlesbare.
 * Ausgabe ist Anzeigetext — wer sie in zusammengebautes HTML setzt, maskiert sie
 * trotzdem (siehe utils/escapeHtml.js), denn der Rohwert-Zweig reicht die Eingabe
 * unverändert durch.
 * @param {string} d
 * @returns {string}
 */
export function fmtDateDE(d) {
	if (!d) return '-';
	try {
		return new Date(d).toLocaleDateString('de-DE');
	} catch {
		return d;
	}
}

/**
 * Letzter Tag eines Monats als YYYY-MM-DD.
 * @param {string} yyyyMM z. B. "2026-02"
 */
export function lastOfMonth(yyyyMM) {
	const [y, m] = yyyyMM.split('-').map(Number);
	return localISO(new Date(y, m, 0));
}
