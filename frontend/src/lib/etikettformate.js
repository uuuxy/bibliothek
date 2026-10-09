/**
 * Die Etikettenbögen, die das Programm bedrucken kann — EINE Liste für das ganze
 * Frontend.
 *
 * Maßgeblich bleibt `pdf/etikett_formate.go`; `etikettformate-konsistenz.test.js` hält
 * diese Datei dagegen. Es gibt sie, weil mit den Schüler-Etiketten (24.08.2026) eine
 * zweite Oberfläche dieselben Bögen anbietet: Vorher stand die Liste hartkodiert in
 * `LabelLayoutOptionen.svelte`, und die Stückzahlen ein zweites Mal in
 * `stores/labels.svelte.js`. Eine dritte Kopie in der Ausweis-Werkzeugleiste wäre genau
 * die Bauform, die in diesem Projekt schon mehrfach auseinandergelaufen ist.
 *
 * KEIN Abruf vom Server: Am 31.08.2026 geprüft und ENTSCHIEDEN so gelassen (Register,
 * Kategorie C) — ein Server-Umbau bräuchte einen neuen Endpunkt plus Ladezustand in
 * drei Verbrauchern des täglich benutzten Druck-Bildschirms. Diese Datei ist die eine
 * Kopie, die der Test gegen pdf/etikett_formate.go hält.
 */

/**
 * `label` steht in der Auswahlliste, `kurz` dort, wo nur der Bogenname hingehört
 * (Vorschau-Überschrift) — sonst schreibt jemand die Namen ein zweites Mal hin.
 *
 * Die Maße in Millimetern (Etikett, Rand des Blatts, Abstand zwischen den Etiketten) braucht
 * die Vorschau des Druck-Centers, damit sie den gewählten Bogen zeichnet.
 *
 * @typedef {{ value: string, label: string, kurz: string, spalten: number, zeilen: number, breite: number, hoehe: number, randOben: number, randLinks: number, abstandX: number, abstandY: number }} EtikettFormat
 * @type {EtikettFormat[]}
 */
export const ETIKETT_FORMATE = [
	{
		value: 'zweckform_l4760',
		label: 'Zweckform L4760 (3x7, 21 Etiketten)',
		kurz: 'Zweckform L4760',
		spalten: 3,
		zeilen: 7,
		breite: 63.5,
		hoehe: 38.1,
		randOben: 15.1,
		randLinks: 7.2,
		abstandX: 2.5,
		abstandY: 0
	},
	{
		value: 'avery_3475',
		label: 'Avery 3475 (3x8, 24 Etiketten)',
		kurz: 'Avery 3475',
		spalten: 3,
		zeilen: 8,
		breite: 70,
		hoehe: 37,
		randOben: 0.5,
		randLinks: 0,
		abstandX: 0,
		abstandY: 0
	},
	{
		value: 'standard_52',
		label: 'Kleine Barcodes (4x13, 52 Etiketten)',
		kurz: 'Standard 52',
		spalten: 4,
		zeilen: 13,
		breite: 48.3,
		hoehe: 21.2,
		randOben: 10.7,
		randLinks: 3.4,
		abstandX: 0,
		abstandY: 0
	}
];

/**
 * Das Format zu einer Kennung. Unbekanntes liefert den ersten Eintrag, damit Überschrift,
 * Startposition und Vorschau bedienbar bleiben.
 *
 * @param {string} formatId
 * @returns {EtikettFormat}
 */
export function etikettFormat(formatId) {
	return ETIKETT_FORMATE.find((f) => f.value === formatId) ?? ETIKETT_FORMATE[0];
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
