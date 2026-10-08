// Was die Maske an einem vorhandenen Titel geändert hat. Das Speichern schickt nur diese
// Felder, und der Server schreibt nur, was der Rumpf nennt: Was ein anderer Platz inzwischen
// an den übrigen Feldern gespeichert hat, bleibt stehen.

/**
 * Der Titel, wie er hinausgeht: Lernmittel als Ja oder Nein, ein leeres Zähldatum als null.
 * @param {any} formular
 * @returns {Record<string, any>}
 */
export function alsRumpf(formular) {
	return {
		...formular,
		istLernmittel: !!formular.istLernmittel,
		lastCounted: formular.lastCounted || null
	};
}

// Kein Feld des Titels: Die Kennung steht im Pfad, der Bestand geht über bestandsAngabe
// hinaus, geladen ist der Stand vom Öffnen.
const KEIN_FELD = new Set(['id', 'stock', 'stockGesehen', 'geladen']);

/** Ein geleertes Feld und ein fehlender Wert sind für den Vergleich dasselbe wie null.
 * @param {any} wert */
const vergleichbar = (wert) => JSON.stringify(wert === '' || wert === undefined ? null : wert);

/**
 * Hält den Stand fest, mit dem die Maske öffnet: eine eigene Kopie, an der das Speichern die
 * geänderten Felder erkennt.
 * @param {any} formular
 */
export function merkeStand(formular) {
	const stand = structuredClone({ ...formular, geladen: undefined });
	// Die Kopie führt keinen eigenen Stand vom Öffnen.
	delete stand.geladen;
	formular.geladen = stand;
}

/**
 * Die Felder, die sich seit dem Öffnen geändert haben, mit ihrem neuen Wert.
 * @param {any} formular Maske eines vorhandenen Titels, geöffnet über titelFuerMaske
 * @returns {Record<string, any>}
 * @throws {Error} wenn der Stand vom Öffnen fehlt: Ohne ihn gälte jedes Feld als geändert
 */
export function geaenderteFelder(formular) {
	if (!formular.geladen) {
		throw new Error(
			'Der Stand vom Öffnen des Titels fehlt. Nichts gespeichert: bitte den Titel neu öffnen.'
		);
	}
	const jetzt = alsRumpf(formular);
	const vorher = alsRumpf(formular.geladen);
	/** @type {Record<string, any>} */
	const felder = {};
	for (const name of Object.keys(jetzt)) {
		if (KEIN_FELD.has(name) || vergleichbar(jetzt[name]) === vergleichbar(vorher[name])) continue;
		// Ein Feld ohne Wert geht als null hinaus: undefined ließe JSON weg, und der Server
		// hielte das Feld für nicht genannt.
		felder[name] = jetzt[name] ?? null;
	}
	return felder;
}

/**
 * Ein Feld, das an der Maske vorbei schon gespeichert ist (das Cover nach dem Hochladen):
 * Maske und Stand vom Öffnen bekommen den Wert, das Speichern schickt ihn nicht noch einmal.
 * @param {any} formular
 * @param {string} feld
 * @param {any} wert
 */
export function uebernimmGespeichert(formular, feld, wert) {
	formular[feld] = wert;
	if (formular.geladen) formular.geladen[feld] = wert;
}
