/**
 * `{@attach (feld) => nachkommastellen(feld, wert)}` — ein Zahlenfeld zeigt so viele
 * Nachkommastellen, wie sein `step` hat. Ein Betrag (step="0.01") steht dann als „7,90“, wie
 * formatEuro ihn in Summe und Bestelldetail schreibt, und nicht als „7,9“.
 *
 * Gesetzt wird nur die Anzeige: außerhalb des Fokus bei jeder Änderung des Werts und beim
 * Verlassen des Feldes; wer tippt, behält seine Eingabe. Der gebundene Wert bleibt die Zahl,
 * und bind:value schreibt das Feld nicht neu, solange es dieselbe Zahl zeigt.
 * @param {HTMLInputElement} feld
 * @param {unknown} wert - der gebundene Wert; das Argument lässt die Anzeige jeder Änderung folgen
 */
export function nachkommastellen(feld, wert) {
	const stellen = feld.type === 'number' ? (feld.step.split('.')[1]?.length ?? 0) : 0;
	if (!stellen) return;
	/** @param {unknown} zahl */
	const zeige = (zahl) => {
		if (zahl !== '' && zahl != null && Number.isFinite(Number(zahl)))
			feld.value = Number(zahl).toFixed(stellen);
	};
	if (document.activeElement !== feld) zeige(wert);
	const verlassen = () => zeige(feld.value);
	feld.addEventListener('blur', verlassen);
	return () => feld.removeEventListener('blur', verlassen);
}
