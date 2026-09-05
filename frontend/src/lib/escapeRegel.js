/**
 * Escape bringt von überall zurück an die Theke — aber nur, wenn die Taste nicht
 * schon jemandem gehört.
 *
 * Vorher galt sie bedingungslos, und das machte ausgerechnet die Berichte unbenutzbar:
 * Die Ansicht besteht aus Monats-, Jahres- und Datumsfeldern, und Escape ist dort die
 * normale Art, ein aufgeklapptes Auswahlfenster wieder zuzumachen. Der Tastendruck kam
 * beim Fenster an, blubberte weiter — und statt des Auswahlfensters schloss sich die
 * ganze Ansicht. Für den Benutzer sprang das Programm grundlos in die Ausleihe.
 *
 * Zwei Ausnahmen, beide am Ereignis selbst ablesbar und damit auch für Bauteile
 * gültig, die es noch gar nicht gibt (bis 05.09.2026 im Router selbst; ausgelagert,
 * weil die Datei nicht weiter wachsen darf und die Regel rein ist):
 * @param {KeyboardEvent} e
 * @returns {boolean}
 */
export function escapeGehoertJemandAnderem(e) {
	// 1. Jemand hat die Taste bereits verarbeitet (Dialog, Menü, Overlay).
	if (e.defaultPrevented) return true;
	// 2. Der Fokus steht in einem Eingabefeld. Dort heisst Escape "Auswahl schließen"
	//    oder "Eingabe verwerfen" — nie "Ansicht verlassen".
	const ziel = /** @type {HTMLElement | null} */ (e.target);
	if (!ziel) return false;
	return ziel.isContentEditable || ['INPUT', 'SELECT', 'TEXTAREA'].includes(ziel.tagName);
}
