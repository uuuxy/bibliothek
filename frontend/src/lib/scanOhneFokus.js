// An der Theke tippt ein Handscanner blind in das Element, das den Fokus hat. Steht der Fokus
// nach einem Klick auf einem Reiter, einem Knopf oder nirgends, bekommt das Scanfeld das Zeichen
// — eine Regel für die ganze Seite statt eines Rückrufs je Knopf.

// Elemente, die Getipptes selbst entgegennehmen. Kästchen und Knöpfe sind <input>, tippen aber
// nichts.
const NIMMT_TASTEN_AN = [
	'textarea',
	'select',
	'input:not([type="checkbox"]):not([type="radio"]):not([type="button"]):not([type="submit"]):not([type="reset"]):not([type="file"]):not([type="range"]):not([type="color"])',
	'[contenteditable=""]',
	'[contenteditable="true"]',
	'[role="combobox"]',
	'[role="listbox"]',
	'[role="menu"]',
	'[role="textbox"]',
	'[role="searchbox"]',
	'[role="slider"]',
	'[role="spinbutton"]'
].join(', ');

// Ein offenes Menü oder eine offene Auswahlliste springt beim Tippen zum passenden Eintrag.
const AUSWAHL_OFFEN = '[role="menu"], [role="listbox"]';

/**
 * Ein Zeichen, wie es Scanner und Tastatur tippen. Leertaste, Steuertasten und Kürzel mit
 * Strg, Alt oder Meta gehören dem Element, das den Fokus hat.
 * @param {KeyboardEvent} e
 */
export function istTippzeichen(e) {
	return (
		e.key.length === 1 && e.key !== ' ' && !e.ctrlKey && !e.metaKey && !e.altKey && !e.isComposing
	);
}

/**
 * Nimmt das Element Getipptes selbst entgegen — ein Feld, eine Auswahlliste, ein Menü?
 * @param {Element | null} el
 */
export function nimmtTastenAn(el) {
	return !!el?.closest(NIMMT_TASTEN_AN);
}

/**
 * Liegt das Scanfeld frei? Ein Dialog, der Sperrbildschirm und die Webcam-Fläche liegen
 * darüber und behalten ihre Eingabe.
 * @param {HTMLElement} feld
 */
function liegtFrei(feld) {
	const r = feld.getBoundingClientRect();
	return document.elementFromPoint(r.left + r.width / 2, r.top + r.height / 2) === feld;
}

/**
 * Tastendruck an der Theke: Fiele das Zeichen ins Leere, bekommt das Scanfeld den Fokus, bevor
 * der Browser das Zeichen einsetzt.
 * @param {KeyboardEvent} e
 * @param {() => boolean} bereit false, solange eine Rückfrage der Theke offen ist oder die Kamera scannt
 */
export function tasteInsScanfeld(e, bereit) {
	if (!istTippzeichen(e) || nimmtTastenAn(document.activeElement)) return;
	const feld = document.getElementById('omnibox-input');
	if (!feld || !bereit() || document.querySelector(AUSWAHL_OFFEN) || !liegtFrei(feld)) return;
	feld.focus();
}
