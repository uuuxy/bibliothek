/**
 * @file exemplarStatus.js
 * Der Zustand eines Exemplars als Wort und Ton für den StatusChip, eine Zuordnung für die
 * Buchakte und die Buchmaske.
 *
 * Was nicht zum Bestand zählt, heißt „Ausgesondert" oder „Bestellt" und nicht „Gesperrt":
 * Gesperrt ist ein Buch, das im Regal steht und nicht ausgeliehen werden darf. Ob ein
 * Exemplar zum Bestand zählt, sagt der Server (`im_bestand`); fehlt das Feld, gilt es als
 * Bestand. Beide Wörter tragen den neutralen Ton, die Fehlerfarbe bleibt der Sperre
 * (M3, Color roles: „Use error roles to communicate error states").
 *
 * @param {{ ist_ausleihbar?: boolean, ist_ausgesondert?: boolean, im_bestand?: boolean, ist_verfuegbar?: boolean }} ex
 * @returns {{ ton: 'erfolg' | 'warten' | 'neutral' | 'fehler', text: string }}
 */
export function exemplarStatus(ex) {
	if (ex.ist_ausgesondert) return { ton: 'neutral', text: 'Ausgesondert' };
	if (ex.im_bestand === false) return { ton: 'neutral', text: 'Bestellt' };
	if (!ex.ist_ausleihbar) return { ton: 'fehler', text: 'Gesperrt' };
	if (!ex.ist_verfuegbar) return { ton: 'warten', text: 'Ausgeliehen' };
	return { ton: 'erfolg', text: 'Verfügbar' };
}

/**
 * Zählt die Exemplare eines Titels nach derselben Grenze wie die Wörter der Karten: im
 * Bestand und bestellt. Ein ausgesondertes Exemplar steht in keiner der beiden Zahlen.
 *
 * @param {{ ist_ausgesondert?: boolean, im_bestand?: boolean }[]} exemplare
 * @returns {{ bestand: number, bestellt: number }}
 */
export function exemplarZahlen(exemplare) {
	let bestand = 0;
	let bestellt = 0;
	for (const ex of exemplare) {
		if (ex.ist_ausgesondert) continue;
		if (ex.im_bestand === false) bestellt += 1;
		else bestand += 1;
	}
	return { bestand, bestellt };
}
