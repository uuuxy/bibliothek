/**
 * Die Etikettenbögen, die das Programm bedrucken kann: eine Liste für das ganze Frontend.
 *
 * Maßgeblich bleibt `internal/pdf/etikett_formate.go`; `etikettformate-konsistenz.test.js` hält
 * diese Datei dagegen. Mehrere Oberflächen bieten dieselben Bögen an (Buch- und
 * Schüler-Etiketten, Ausweis-Werkzeugleiste), und jede eigene Kopie liefe auseinander.
 *
 * Kein Abruf vom Server: Dafür bräuchte es einen neuen Endpunkt und einen Ladezustand in
 * drei Verbrauchern des täglich benutzten Druck-Bildschirms. Diese Datei ist die eine
 * Kopie, die der Test gegen internal/pdf/etikett_formate.go hält.
 */

/**
 * `label` steht in der Auswahlliste, `kurz` dort, wo nur der Bogenname hingehört
 * (Vorschau-Überschrift) — sonst schreibt jemand die Namen ein zweites Mal hin. Die Maße in
 * Millimetern kennt nur der Server: Die Vorschau zeigt sein Druck-PDF.
 *
 * @typedef {{ value: string, label: string, kurz: string, spalten: number, zeilen: number, ohneStartfeld?: boolean }} EtikettFormat
 * @type {EtikettFormat[]}
 */
export const ETIKETT_FORMATE = [
	{
		value: 'zweckform_l4760',
		label: 'Zweckform L4760 (3x7, 21 Etiketten)',
		kurz: 'Zweckform L4760',
		spalten: 3,
		zeilen: 7
	},
	{
		value: 'avery_3475',
		label: 'Avery 3475 (3x8, 24 Etiketten)',
		kurz: 'Avery 3475',
		spalten: 3,
		zeilen: 8
	},
	{
		value: 'standard_52',
		label: 'Kleine Barcodes (4x13, 52 Etiketten)',
		kurz: 'Standard 52',
		spalten: 4,
		zeilen: 13
	}
];

/**
 * Das große Lernmittel-Etikett, nur bei den Buch-Etiketten des Druck-Centers. Es ist kein
 * vorgestanzter Bogen: Vier Felder liegen auf einem A4-Blatt und werden ausgeschnitten, ein
 * Startfeld gibt es deshalb nicht. Erzeuger und Topf-Regel liegen beim Server
 * (pdf.LernmittelEtikettFormat).
 *
 * @type {EtikettFormat}
 */
export const LERNMITTEL_ETIKETT = {
	value: 'lernmittel_gross',
	label: 'Große Lernmittel-Etiketten (2x2, 4 Etiketten)',
	kurz: 'Große Lernmittel-Etiketten',
	spalten: 2,
	zeilen: 2,
	ohneStartfeld: true
};

/** Die Auswahl der Buch-Etiketten im Druck-Center: die Bögen und das große Etikett. */
export const BUCH_ETIKETT_FORMATE = [...ETIKETT_FORMATE, LERNMITTEL_ETIKETT];

/**
 * Das Format zu einer Kennung. Unbekanntes liefert den ersten Eintrag, damit Überschrift und
 * Startposition bedienbar bleiben.
 *
 * @param {string} formatId
 * @returns {EtikettFormat}
 */
export function etikettFormat(formatId) {
	return BUCH_ETIKETT_FORMATE.find((f) => f.value === formatId) ?? ETIKETT_FORMATE[0];
}

/**
 * Felder auf einem Bogen dieses Formats — die Obergrenze der Startposition.
 *
 * Unbekanntes Format liefert die Felderzahl des ersten Eintrags statt 0: Eine 0 machte
 * das Eingabefeld unbedienbar (min=1, max=0), und zwar genau dann, wenn ohnehin schon
 * etwas nicht stimmt.
 *
 * @param {string} formatId
 * @returns {number}
 */
export function felderProBogen(formatId) {
	const format = etikettFormat(formatId);
	return format.spalten * format.zeilen;
}

/**
 * Die Startposition, mit der gedruckt wird: zwischen dem ersten und dem letzten Feld des Bogens.
 * Das Eingabefeld nimmt auch eine größere Zahl an; ohne Grenze zählten Vorschau und Druck die
 * Felder verschieden.
 *
 * @param {number} startFeld
 * @param {string} formatId
 * @returns {number}
 */
export function startfeldAufBogen(startFeld, formatId) {
	if (etikettFormat(formatId).ohneStartfeld) return 1;
	return Math.min(Math.max(1, Math.trunc(Number(startFeld)) || 1), felderProBogen(formatId));
}
