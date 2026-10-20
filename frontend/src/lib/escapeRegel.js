import { escapeIstBelegt } from './components/ui/escapeSchliesst.js';

/**
 * Escape bringt von überall zurück an die Theke, aber nur, wenn die Taste nicht schon
 * jemandem gehört. In den Berichten etwa schließt Escape ein aufgeklapptes Auswahlfenster
 * für Monat, Jahr oder Datum; ginge der Tastendruck weiter, schlösse sich die ganze
 * Ansicht, und das Programm spränge grundlos in die Ausleihe.
 *
 * Zwei Ausnahmen, beide am Ereignis selbst ablesbar und damit auch für Bauteile gültig,
 * die es noch gar nicht gibt:
 * @param {KeyboardEvent} e
 * @returns {boolean}
 */
export function escapeGehoertJemandAnderem(e) {
	// 1. Jemand hat die Taste bereits verarbeitet (Dialog, Menü, Overlay).
	if (e.defaultPrevented) return true;
	// 1b. Ein Overlay ist offen, das Escape schließt (escapeSchliesst). Sein Lauscher
	//     kommt nach dem Router dran; das Ereignis kann es also noch nicht verraten, der
	//     Stapel schon. Ohne diese Frage spränge der Bildschirm zur Theke, und der Dialog
	//     ginge zu, mit einem Tastendruck.
	if (escapeIstBelegt()) return true;
	// 2. Der Fokus steht in einem Eingabefeld. Dort heisst Escape "Auswahl schließen"
	//    oder "Eingabe verwerfen" — nie "Ansicht verlassen".
	const ziel = /** @type {HTMLElement | null} */ (e.target);
	if (!ziel) return false;
	return ziel.isContentEditable || ['INPUT', 'SELECT', 'TEXTAREA'].includes(ziel.tagName);
}
