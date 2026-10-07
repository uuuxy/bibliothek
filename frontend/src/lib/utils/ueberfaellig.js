// utils/ueberfaellig.js
// Ob eine Ausleihe überfällig ist.

/**
 * Überfällig ist eine Ausleihe, deren Frist vorbei ist. Eine Dauerleihe (Kollegium) hat keine
 * Frist und wird es nie, wie in der Sperr-Automatik.
 *
 * @param {{ rueckgabe_frist: string, ist_dauerleihe?: boolean | null }} ausleihe
 * @param {Date} [jetzt] Vergleichszeitpunkt (Tests setzen ihn)
 * @returns {boolean}
 */
export function istUeberfaellig(ausleihe, jetzt = new Date()) {
	return !ausleihe.ist_dauerleihe && new Date(ausleihe.rueckgabe_frist) < jetzt;
}
