/**
 * Die Maske einer Lieferantenzeile: der Stand beim Öffnen und was sich seitdem geändert hat,
 * in den Namen der Anfrage (PUT /api/lieferanten/{id}).
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

/**
 * Was an den Server geht: nur die Felder, die sich seit dem Öffnen geändert haben. Der Server
 * schreibt nur, was der Rumpf nennt; was ein anderer Platz inzwischen gespeichert hat, bleibt
 * stehen, auch das Merkmal Hauptlieferant.
 * @param {Record<string, any>} geladen der Stand beim Öffnen (lieferantStand)
 * @param {Record<string, any>} jetzt der Stand der Maske, mit denselben Namen
 * @returns {Record<string, any>}
 */
export function lieferantAenderung(geladen, jetzt) {
	return Object.fromEntries(
		Object.keys(geladen)
			.filter((name) => jetzt[name] !== geladen[name])
			.map((name) => [name, jetzt[name]])
	);
}
