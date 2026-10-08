/**
 * Die Felder einer Maske, deren Wert sich gegenüber dem Stand vom Öffnen geändert hat.
 *
 * Eine Maske füllt sich aus dem, was beim Öffnen geladen war. Schickte sie beim Speichern jedes
 * Feld zurück, schriebe sie diesen Stand über das, was ein anderer Platz inzwischen gespeichert
 * hat. Sie schickt deshalb nur das Geänderte, und der Server schreibt nur, was der Rumpf nennt.
 * @param {Record<string, any>} geladen die Werte beim Öffnen
 * @param {Record<string, any>} jetzt die Werte der Maske, unter denselben Namen
 * @returns {Record<string, any>}
 */
export function nurGeaendertes(geladen, jetzt) {
	return Object.fromEntries(Object.entries(jetzt).filter(([name, wert]) => wert !== geladen[name]));
}
