/**
 * Die vier Gründe, mit denen ein Exemplar aus dem Bestand geht, in der Reihenfolge der Auswahl
 * im Löschdialog. Der Server schreibt denselben Text ins Abgangsbuch (repository/aussonderungsgrund.go);
 * beide Seiten prüfen gegen aussonderungsgruende.faelle.json.
 *
 * „Alter Bestand" ist vorgewählt: In Littera stand er bei knapp drei Vierteln aller Abgänge mit
 * Grund, und M3 verlangt eine Vorauswahl („Radio buttons should always have one option
 * pre-selected", Radio button, Guidelines).
 */

/** @type {{ wert: string, text: string }[]} */
export const AUSSONDERUNGSGRUENDE = [
	{ wert: 'AUSSORTIERT', text: 'Alter Bestand' },
	{ wert: 'BESCHAEDIGUNG', text: 'Beschädigt' },
	{ wert: 'VERLUST', text: 'Verlust' },
	{ wert: 'BESTANDSKORREKTUR', text: 'Falsch angelegt' }
];

/** Der vorgewählte Grund. */
export const GRUND_VORGABE = 'AUSSORTIERT';
