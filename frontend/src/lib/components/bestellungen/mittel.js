// Der Topf einer Bestellung: Lernmittelfreiheit des Landes, Schülerbücherei des Schulträgers oder
// eine Spende ohne Topf, etwa vom Förderverein.
//
// Dasselbe Vokabular wie internal/pkg/mitteltopf und bestellungen_verlauf.mittel (Migrationen
// 109 und 182). Eine Bestellung = ein Topf; der Titel schlägt ihn über ist_lernmittel nur vor,
// und eine Position lässt sich im Warenkorb in einen anderen Topf schieben (falsch
// gekennzeichneter Titel, Spende).

/** @typedef {'land' | 'schultraeger' | 'spende'} Mittel */

/** @type {Record<Mittel, { label: string, traeger: string }>} */
export const MITTEL = {
	land: { label: 'Lernmittelfreiheit', traeger: 'Land' },
	schultraeger: { label: 'Schülerbücherei', traeger: 'Schulträger' },
	spende: { label: 'Spende', traeger: 'ohne Topf' }
};

/** Reihenfolge im Warenkorb und in Listen: Lernmittel zuerst — sie sind der Regelfall. */
export const MITTEL_REIHENFOLGE = /** @type {Mittel[]} */ (['land', 'schultraeger', 'spende']);

/**
 * Vorschlag aus dem Titel: Lernmittel → Land, sonst Schulträger.
 * @param {boolean | undefined} istLernmittel
 * @returns {Mittel}
 */
export function mittelVorschlag(istLernmittel) {
	return istLernmittel ? 'land' : 'schultraeger';
}

/**
 * Die Töpfe, in die eine Position wechseln kann: alle außer ihrem eigenen, in der Reihenfolge.
 * @param {Mittel} mittel @returns {Mittel[]}
 */
export function andereMittel(mittel) {
	return MITTEL_REIHENFOLGE.filter((m) => m !== mittel);
}

/**
 * Anzeigetext. Ein leerer Wert ist eine Alt-Bestellung ohne eindeutige Zuordnung — sie
 * wird als solche benannt, nie einem Topf zugeschlagen.
 * @param {string | null | undefined} mittel
 */
export function mittelBeschriftung(mittel) {
	return mittel && mittel in MITTEL
		? `${MITTEL[/** @type {Mittel} */ (mittel)].label} · ${MITTEL[/** @type {Mittel} */ (mittel)].traeger}`
		: 'ohne Zuordnung';
}
