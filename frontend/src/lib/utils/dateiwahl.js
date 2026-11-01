/**
 * Übernimmt die gewählte Datei und leert das Feld. Bliebe sie darin stehen, meldete der Browser bei
 * derselben Datei keine Änderung: Eine korrigierte Liste oder ein zweites Bild käme nicht an. Die
 * Datei selbst bleibt danach lesbar.
 * @param {Event} e das change-Ereignis des Dateifelds
 * @returns {File | null}
 */
export function nimmDatei(e) {
	const feld = /** @type {HTMLInputElement} */ (e.currentTarget);
	const datei = feld.files?.[0] ?? null;
	feld.value = '';
	return datei;
}
