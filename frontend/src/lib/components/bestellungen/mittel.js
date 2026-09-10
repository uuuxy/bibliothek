// Der Topf einer Bestellung: Lernmittelfreiheit (Land) oder Schülerbücherei (Schulträger).
//
// Dasselbe Vokabular wie repository/mittel.go und bestellungen_verlauf.mittel (Migration
// 109). Eine Bestellung = ein Topf; der Titel schlägt ihn über ist_lernmittel nur vor,
// und eine Position lässt sich im Warenkorb in den anderen Topf schieben (falsch
// gekennzeichneter Titel).

/** @typedef {'land' | 'schultraeger'} Mittel */

/** @type {Record<Mittel, { label: string, traeger: string }>} */
export const MITTEL = {
	land: { label: 'Lernmittelfreiheit', traeger: 'Land' },
	schultraeger: { label: 'Schülerbücherei', traeger: 'Schulträger' }
};

/** Reihenfolge im Warenkorb und in Listen: Lernmittel zuerst — sie sind der Regelfall. */
export const MITTEL_REIHENFOLGE = /** @type {Mittel[]} */ (['land', 'schultraeger']);

/**
 * Vorschlag aus dem Titel: Lernmittel → Land, sonst Schulträger.
 * @param {boolean | undefined} istLernmittel
 * @returns {Mittel}
 */
export function mittelVorschlag(istLernmittel) {
	return istLernmittel ? 'land' : 'schultraeger';
}

/** @param {Mittel} mittel @returns {Mittel} */
export function anderesMittel(mittel) {
	return mittel === 'land' ? 'schultraeger' : 'land';
}

/**
 * Anzeigetext. Ein leerer Wert ist eine Alt-Bestellung ohne eindeutige Zuordnung — sie
 * wird als solche benannt, nie einem Topf zugeschlagen.
 * @param {string | null | undefined} mittel
 */
export function mittelLabel(mittel) {
	return mittel && mittel in MITTEL
		? `${MITTEL[/** @type {Mittel} */ (mittel)].label} (${MITTEL[/** @type {Mittel} */ (mittel)].traeger})`
		: 'ohne Zuordnung';
}
