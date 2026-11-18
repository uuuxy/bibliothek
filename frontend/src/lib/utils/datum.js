// utils/datum.js
// Lokale Datumsformatierung für Berichts-Zeiträume und den Kalender des Datumsfelds.
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
 * Letzter Tag eines Monats als YYYY-MM-DD.
 * @param {string} yyyyMM z. B. "2026-02"
 */
export function lastOfMonth(yyyyMM) {
	const [y, m] = yyyyMM.split('-').map(Number);
	return localISO(new Date(y, m, 0));
}

/**
 * Ein Datum als YYYY-MM-DD als lokaler Tag, ohne den Umweg über UTC.
 * @param {string} iso
 */
function ausISO(iso) {
	const [j, m, t] = iso.split('-').map(Number);
	return new Date(j, m - 1, t);
}

// Die Schreibweisen, die ein Datumsfeld annimmt. M3 (Date pickers, Accessibility): „accept a range of
// formats including dashes, spaces, slashes, dots, and 0 to the left of a single digit
// month/day“. Das Jahr steht vierstellig: Zwei Ziffern ließen beim Geburtsdatum offen, welches
// Jahrhundert gemeint ist.
const SCHREIBWEISEN = [
	{ muster: /^(\d{4})-(\d{1,2})-(\d{1,2})$/, jahr: 1, monat: 2, tag: 3 },
	{ muster: /^(\d{1,2})[.\-/\s]+(\d{1,2})[.\-/\s]+(\d{4})$/, jahr: 3, monat: 2, tag: 1 },
	{ muster: /^(\d{2})(\d{2})(\d{4})$/, jahr: 3, monat: 2, tag: 1 }
];

/**
 * Das Datum einer Eingabe als YYYY-MM-DD: '' für eine leere Eingabe, null für eine, die kein
 * Datum nennt, etwa den 31. Februar oder ein Jahr mit zwei Ziffern.
 * @param {string} text
 * @returns {string | null}
 */
export function leseDatum(text) {
	const s = text.trim();
	if (!s) return '';
	for (const weise of SCHREIBWEISEN) {
		const m = weise.muster.exec(s);
		if (!m) continue;
		const [jahr, monat, tag] = [m[weise.jahr], m[weise.monat], m[weise.tag]].map(Number);
		const d = new Date(jahr, monat - 1, tag);
		const stimmt = d.getFullYear() === jahr && d.getMonth() === monat - 1 && d.getDate() === tag;
		return stimmt ? localISO(d) : null;
	}
	return null;
}

/**
 * Die 42 Tage des Kalenderblatts eines Monats: sechs Wochen, jede ab Montag.
 * @param {number} jahr
 * @param {number} monat 1 bis 12
 */
export function kalenderBlatt(jahr, monat) {
	const versatz = (new Date(jahr, monat - 1, 1).getDay() + 6) % 7;
	return Array.from({ length: 42 }, (_, i) => localISO(new Date(jahr, monat - 1, 1 - versatz + i)));
}

/**
 * Verschiebt ein Datum um Tage oder um Monate und Jahre. Bei Monaten bleibt ein Monatsende im
 * Zielmonat: 31. Januar und ein Monat ist der 28. Februar, nicht der 3. März.
 * @param {string} iso
 * @param {{ tage?: number, monate?: number, jahre?: number }} schritt
 */
export function verschiebe(iso, { tage = 0, monate = 0, jahre = 0 }) {
	const d = ausISO(iso);
	if (tage) return localISO(new Date(d.getFullYear(), d.getMonth(), d.getDate() + tage));
	const ziel = new Date(d.getFullYear() + jahre, d.getMonth() + monate, 1);
	const letzter = new Date(ziel.getFullYear(), ziel.getMonth() + 1, 0).getDate();
	return localISO(new Date(ziel.getFullYear(), ziel.getMonth(), Math.min(d.getDate(), letzter)));
}

/**
 * „Montag, 17. August 2026“. M3 (Date pickers, Accessibility): „labels are used to enumerate
 * the complete date“.
 * @param {string} iso
 */
export function langesDatum(iso) {
	return ausISO(iso).toLocaleDateString('de-DE', {
		weekday: 'long',
		day: 'numeric',
		month: 'long',
		year: 'numeric'
	});
}

/** Die Monate mit vollem Namen, Januar zuerst. */
export const MONATE = Array.from({ length: 12 }, (_, i) =>
	new Date(2026, i, 1).toLocaleDateString('de-DE', { month: 'long' })
);

/** Die Wochentage ab Montag, kurz für die Kopfzeile und lang für Sprechblase und Screenreader. */
export const WOCHENTAGE = [
	['Mo', 'Montag'],
	['Di', 'Dienstag'],
	['Mi', 'Mittwoch'],
	['Do', 'Donnerstag'],
	['Fr', 'Freitag'],
	['Sa', 'Samstag'],
	['So', 'Sonntag']
].map(([kurz, lang]) => ({ kurz, lang }));
