/** Die Tastaturbedienung des M3-Auswahlfelds — als reine Funktionen.
 *
 *  Warum eigene Datei: Ein nachgebautes Auswahlfeld muss ohne Maus dasselbe können
 *  wie ein natives (Pfeile, Pos1/Ende, Enter/Leertaste, Escape, Tippen springt).
 *  Diese Regeln sind der Teil, der leicht still kaputtgeht — in Select.svelte lagen
 *  sie als verschachtelte if-Kette zwischen Geometrie, Fokus und Markup und waren
 *  nur über das gerenderte Bauteil prüfbar.
 *
 *  Ausgelagert am 12.09.2026 (Paket 5, #593): Die Datei stand mit 207 Zeilen im
 *  Bestand der Größen-Ratsche und sollte vor dem nächsten Eingriff kleiner werden. */

/**
 * Der nächste wandernde Index — überspringt gesperrte Einträge und läuft um.
 * Gibt es nur gesperrte Einträge, bleibt der Index, wo er war.
 * @param {Array<{ disabled?: boolean }>} options
 * @param {number} aktiv
 * @param {number} richtung 1 = abwärts, -1 = aufwärts
 */
export function naechsterIndex(options, aktiv, richtung) {
	if (!options.length) return aktiv;
	let i = aktiv;
	for (let n = 0; n < options.length; n++) {
		i = (i + richtung + options.length) % options.length;
		if (!options[i].disabled) break;
	}
	return i;
}

/**
 * Tippen springt zum ersten Eintrag, dessen Beschriftung mit dem Puffer beginnt —
 * wie beim nativen select. −1 heißt: kein Treffer, der Index bleibt stehen.
 * @param {Array<{ label: string }>} options
 * @param {string} puffer bereits kleingeschrieben
 */
export function tippsprungIndex(options, puffer) {
	return options.findIndex((o) => o.label.toLowerCase().startsWith(puffer));
}

/**
 * @typedef {{
 *   tat: 'nichts' | 'oeffnen' | 'schliessen' | 'verlassen' | 'wandern' | 'springen' | 'waehlen' | 'tippen',
 *   verhindern: boolean,
 *   richtung?: number,
 *   ziel?: 'anfang' | 'ende',
 *   zeichen?: string
 * }} Tastenbefehl
 */

/**
 * Übersetzt einen Tastendruck in das, was das Auswahlfeld tun soll.
 *
 * `verhindern` sagt, ob der Browser seine eigene Deutung lassen soll: Pfeiltasten
 * scrollen sonst die Seite, die Leertaste ebenso. NICHT verhindert werden Tab (der
 * Fokus soll weiterwandern) und das Tippen (ein Zeichen darf nichts blockieren).
 *
 * @param {string} taste `KeyboardEvent.key`
 * @param {boolean} offen
 * @returns {Tastenbefehl}
 */
export function tastenBefehl(taste, offen) {
	if (!offen) {
		return ['Enter', ' ', 'ArrowDown', 'ArrowUp'].includes(taste)
			? { tat: 'oeffnen', verhindern: true }
			: { tat: 'nichts', verhindern: false };
	}
	switch (taste) {
		case 'ArrowDown':
			return { tat: 'wandern', richtung: 1, verhindern: true };
		case 'ArrowUp':
			return { tat: 'wandern', richtung: -1, verhindern: true };
		case 'Escape':
			return { tat: 'schliessen', verhindern: true };
		case 'Home':
			return { tat: 'springen', ziel: 'anfang', verhindern: true };
		case 'End':
			return { tat: 'springen', ziel: 'ende', verhindern: true };
		case 'Enter':
		case ' ':
			return { tat: 'waehlen', verhindern: true };
		case 'Tab':
			return { tat: 'verlassen', verhindern: false };
		default:
			// Ein einzelnes Zeichen ist ein Tippsprung; alles andere (F5, Shift, …)
			// gehört dem Browser.
			return taste.length === 1
				? { tat: 'tippen', zeichen: taste, verhindern: false }
				: { tat: 'nichts', verhindern: false };
	}
}
