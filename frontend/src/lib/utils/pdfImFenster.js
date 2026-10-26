// utils/pdfImFenster.js
// Zeigt ein PDF vom Server in einem neuen Tab, statt es herunterzuladen.
//
// Ein Download bliebe im Download-Ordner des Arbeitsplatzes liegen, auch für den Nächsten am
// Rechner. Im Tab steht das PDF, solange der Tab offen ist; gedruckt und gespeichert wird über
// die Knöpfe des Browsers.

/** Die Meldung, wenn der Browser den Tab nicht öffnet. */
export const PDF_BLOCKIERT = 'Bitte erlaube Popups, um das PDF zu öffnen.';

/** So oft sieht die Seite nach, ob der Tab geschlossen ist. */
const PRUEFTAKT_MS = 10_000;

/**
 * Öffnet einen Tab und zeigt darin das PDF, das `hole` vom Server holt.
 *
 * Der Tab geht beim Klick auf, vor der Anfrage: Nach einer Antwort, die länger als einige
 * Sekunden braucht, blockiert Chromium ein neues Fenster, und ein Druck, der beim Server etwas
 * zählt (Mahnbriefe), hätte dann gezählt, ohne dass jemand das Blatt sieht. Ist die Antwort ein
 * Fehler, schließt sich der Tab wieder; den Fehler meldet der Aufrufer.
 *
 * @param {() => Promise<Response>} hole
 * @returns {Promise<Response | null>} die Antwort des Servers; null, wenn der Browser den Tab
 *   blockiert hat und keine Anfrage hinausging
 */
export async function zeigePdf(hole) {
	const fenster = window.open('', '_blank');
	if (!fenster) return null;
	try {
		const antwort = await hole();
		if (!antwort.ok) {
			fenster.close();
			return antwort;
		}
		const adresse = URL.createObjectURL(await antwort.blob());
		fenster.location.replace(adresse);
		gibFreiWennZu(fenster, adresse);
		return antwort;
	} catch (fehler) {
		fenster.close();
		throw fehler;
	}
}

/**
 * Gibt die Adresse frei, sobald der Tab geschlossen ist. Vorher nicht: „Speichern" im Tab lädt
 * die Adresse noch einmal und bricht nach der Freigabe ab.
 * @param {Window} fenster
 * @param {string} adresse
 */
function gibFreiWennZu(fenster, adresse) {
	const wache = setInterval(() => {
		if (!fenster.closed) return;
		clearInterval(wache);
		URL.revokeObjectURL(adresse);
	}, PRUEFTAKT_MS);
}
