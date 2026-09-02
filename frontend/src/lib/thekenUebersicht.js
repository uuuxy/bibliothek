/**
 * @file thekenUebersicht.js
 * Was die Theke im Ruhezustand über den Tag weiß — als ZAHLEN, nie als Namen.
 *
 * Der Theken-Bildschirm steht am Tresen und ist für Schüler einsehbar. Deshalb
 * trägt die Übersicht nur Anzahlen (überfällig, Abholfach, Klassensätze, Anliegen)
 * und nie, WER säumig ist — das steht im Mahnwesen, hinter dem Klick.
 *
 * Jede Kachel hängt am Recht IHRER Route (wie hintergrundAbrufe.svelte.js): Ohne das
 * Recht wird weder abgefragt (kein 403-Toast) noch angezeigt.
 */

/** @typedef {{ id: string, label: string, recht: string, ziel: string | null }} Kachel */

/** @type {Kachel[]} */
export const KACHELN = [
	{ id: 'ueberfaellig', label: 'überfällig', recht: 'view_students', ziel: 'mahnwesen' },
	// Kein Ziel: Abholbereite Vormerkungen haben keine eigene Liste — sie erscheinen an
	// der Theke, sobald der Ausweis gescannt wird (OmniboxThekeHinweise).
	{ id: 'abholbereit', label: 'im Abholfach', recht: 'manage_vormerkungen', ziel: null },
	{ id: 'klassensaetze', label: 'Klassensätze wartend', recht: 'view_orders', ziel: 'orders' },
	{ id: 'anliegen', label: 'Anliegen offen', recht: 'view_orders', ziel: 'orders' }
];

/**
 * Anzahl der Vormerkungen, die im Abholfach liegen.
 * @param {unknown} liste Antwort von GET /api/vormerkungen
 */
export function zaehleAbholbereit(liste) {
	if (!Array.isArray(liste)) return 0;
	return liste.filter((v) => v && v.status === 'abholbereit').length;
}

/**
 * Anzahl überfälliger Ausleihen aus GET /api/dashboard/summary.
 * @param {unknown} summary
 */
export function zaehleUeberfaellig(summary) {
	const n = /** @type {any} */ (summary)?.total_overdue;
	return typeof n === 'number' && n > 0 ? n : 0;
}

/**
 * Welche Kacheln ein Benutzer sieht — nur die, deren Route er lesen darf.
 * @param {(recht: string) => boolean} darf
 */
export function sichtbareKacheln(darf) {
	return KACHELN.filter((k) => darf(k.recht));
}
