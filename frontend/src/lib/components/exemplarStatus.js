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
 * Bestand, davon verfügbar, und bestellt. Ein ausgesondertes Exemplar steht in keiner der
 * Zahlen.
 *
 * @param {{ ist_ausleihbar?: boolean, ist_ausgesondert?: boolean, im_bestand?: boolean, ist_verfuegbar?: boolean }[]} exemplare
 * @returns {{ bestand: number, verfuegbar: number, bestellt: number }}
 */
export function exemplarZahlen(exemplare) {
	let bestand = 0;
	let verfuegbar = 0;
	let bestellt = 0;
	for (const ex of exemplare) {
		if (ex.ist_ausgesondert) continue;
		if (ex.im_bestand === false) {
			bestellt += 1;
			continue;
		}
		bestand += 1;
		if (ex.ist_ausleihbar && ex.ist_verfuegbar) verfuegbar += 1;
	}
	return { bestand, verfuegbar, bestellt };
}
