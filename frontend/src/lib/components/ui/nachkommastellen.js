/**
 * Ein Feld mit Nachkommastellen im `step` (ein Betrag: step="0.01") ist ein Textfeld mit
 * Zifferntastatur, kein Zahlenfeld: Firefox schreibt den Wert eines Zahlenfelds selbst und kürzt
 * „7,90“ zu „7,9“. Google, web.dev (Payment and address form best practices): „Avoid using
 * type="number" for numbers that aren't meant to be incremented … Use type="text" and
 * inputmode="numeric" instead.“ Ein Betrag braucht das Komma, daher inputmode="decimal".
 *
 * Das Feld zeigt den Betrag wie formatEuro („7,90“); gebunden bleibt die Zahl, und ein leeres
 * Feld oder eines ohne Zahl ist null wie bei bind:value an einem Zahlenfeld.
 */

/**
 * @param {unknown} step
 * @returns {number} die Nachkommastellen, 0 bei ganzen Schritten
 */
export function stellenAus(step) {
	return String(step ?? '').split('.')[1]?.length ?? 0;
}

/**
 * „7,9“, „7.9“ und „ 7,90 “ ergeben 7.9; leer oder keine Zahl ergibt null.
 * @param {unknown} text
 * @returns {number | null}
 */
export function alsZahl(text) {
	const roh = String(text).trim().replace(',', '.');
	return /^[+-]?(\d+\.?\d*|\.\d+)$/.test(roh) ? Number(roh) : null;
}

/**
 * Mit zwei Stellen ergibt 7.9 „7,90“; ohne Zahl bleibt das Feld leer.
 * @param {unknown} wert
 * @param {number} stellen
 */
export function alsText(wert, stellen) {
	if (wert === '' || wert == null || !Number.isFinite(Number(wert))) return '';
	return Number(wert).toFixed(stellen).replace('.', ',');
}

/**
 * Was das Feld zeigt: die Eingabe, solange sie den Wert trifft, damit „7,9“ beim Tippen stehen
 * bleibt, sonst den Wert mit allen Stellen. Ohne Stellen geht der Wert unverändert durch.
 * @param {HTMLInputElement | undefined} feld
 * @param {unknown} wert
 * @param {number} stellen
 */
export function anzeige(feld, wert, stellen) {
	if (!stellen) return wert;
	const soll = wert === '' || wert == null ? null : Number(wert);
	return feld && alsZahl(feld.value) === soll ? feld.value : alsText(wert, stellen);
}

/**
 * Was gebunden wird: bei einem Betrag die Zahl aus dem Getippten, sonst die Eingabe selbst.
 * @param {unknown} eingabe
 * @param {number} stellen
 */
export function wertAus(eingabe, stellen) {
	return stellen ? alsZahl(eingabe) : eingabe;
}

/**
 * `{@attach betragsEingabe(stellen)}`: Getippt werden nur Ziffern, Komma, Punkt und Vorzeichen,
 * wie es Chrome am Zahlenfeld hielt; beim Verlassen steht eine Zahl mit allen Stellen da.
 * @param {number} stellen
 * @returns {import('svelte/attachments').Attachment<HTMLInputElement>}
 */
export function betragsEingabe(stellen) {
	return (feld) => {
		if (!stellen) return;
		/** @param {InputEvent} e */
		const sieben = (e) => {
			if (e.data && /[^\d.,+-]/.test(e.data)) e.preventDefault();
		};
		const ordnen = () => {
			const zahl = alsZahl(feld.value);
			if (zahl !== null) feld.value = alsText(zahl, stellen);
		};
		feld.addEventListener('beforeinput', sieben);
		feld.addEventListener('blur', ordnen);
		return () => {
			feld.removeEventListener('beforeinput', sieben);
			feld.removeEventListener('blur', ordnen);
		};
	};
}
