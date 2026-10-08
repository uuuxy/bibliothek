/**
 * Die Maske einer Lieferantenzeile: der Stand beim Öffnen in den Namen der Anfrage
 * (PUT /api/lieferanten/{id}). Was sich seitdem geändert hat, liefert utils/geaendert.js.
 */

/**
 * Der Stand einer Zeile in den Namen der Anfrage.
 * @param {{ name: string, email: string, customerNumber: string, ist_hauptlieferant?: boolean, kundennummer_schultraeger?: string }} lieferant
 */
export function lieferantStand(lieferant) {
	return {
		name: lieferant.name,
		email: lieferant.email,
		customerNumber: lieferant.customerNumber,
		ist_hauptlieferant: lieferant.ist_hauptlieferant ?? false,
		kundennummer_schultraeger: lieferant.kundennummer_schultraeger ?? ''
	};
}
