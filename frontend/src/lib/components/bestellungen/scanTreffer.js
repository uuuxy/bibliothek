/**
 * Welcher Suchtreffer gehört zu einem gescannten Code — und ist er eindeutig genug, um ihn
 * ohne Rückfrage zu übernehmen?
 *
 * Getrennt von OrderSearch.svelte, weil hier nichts angezeigt wird: Es ist eine Entscheidung
 * über Zahlen, und sie ist der Unterschied zwischen „der Scan spart einen Handgriff" und
 * „der Scan legt das falsche Buch in die Bestellung".
 *
 * Die Regel, in dieser Reihenfolge:
 *
 *  1. Genau EIN Treffer, dessen ISBN dem Scan entspricht → der ist gemeint. Verglichen
 *     werden nur die Ziffern: Auf dem Buch steht `978-3-06-013076-4`, der Scanner liefert
 *     `9783060130764`, und die DNB antwortet mal mit, mal ohne Bindestriche.
 *  2. Kein ISBN-Treffer, aber insgesamt genau ein Suchtreffer → auch der ist gemeint.
 *  3. Alles andere (nichts gefunden, mehrere Kandidaten) → null, und der Mensch entscheidet
 *     an der Trefferliste. Bei zwei Ausgaben desselben Titels wäre eine automatische Wahl
 *     geraten, und geraten wird beim Bestellen nicht.
 *
 * Die Prüfziffer wird NICHT nachgerechnet: Ein Scanner, der eine ISBN falsch liest, liefert
 * eine Nummer, die es nicht gibt — dann fällt der Fall ohnehin auf Regel 3.
 *
 * @param {string} code Der gescannte Code
 * @param {Array<{isbn?: string}> | undefined} treffer Ergebnis der Titelsuche
 * @returns {any} der eindeutige Treffer oder null
 */
export function scanTreffer(code, treffer) {
	const liste = Array.isArray(treffer) ? treffer : [];
	const ziffern = (/** @type {unknown} */ v) => String(v ?? '').replace(/\D/g, '');
	const gescannt = ziffern(code);

	if (gescannt !== '') {
		const perIsbn = liste.filter((t) => ziffern(t.isbn) !== '' && ziffern(t.isbn) === gescannt);
		if (perIsbn.length === 1) return perIsbn[0];
		if (perIsbn.length > 1) return null;
	}

	return liste.length === 1 ? liste[0] : null;
}

/**
 * Was ein Scan in der Titelsuche auslöst: sofort suchen, und bei eindeutigem Treffer direkt
 * übernehmen. Ohne eindeutigen Treffer bleibt die Trefferliste stehen — dann entscheidet
 * der Mensch, wie beim Tippen.
 *
 * Steht hier und nicht in OrderSearch.svelte, weil es eine Abfolge von Entscheidungen ist
 * und weil die Datei dort an der 200-Zeilen-Grenze liegt.
 *
 * @param {string} code
 * @param {{ sucheSofort: (q: string) => Promise<any>, showDropdown: boolean }} store
 * @param {(treffer: any) => void} uebernehmen
 * @returns {Promise<boolean>} true, wenn direkt übernommen wurde
 */
export async function scanUebernehmen(code, store, uebernehmen) {
	const eindeutig = scanTreffer(code, await store.sucheSofort(code));
	if (!eindeutig) return false;
	store.showDropdown = false;
	uebernehmen(eindeutig);
	return true;
}
