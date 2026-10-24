/**
 * @file format.js
 * Zahlen, Prozent und Zeitpunkte in einer deutschen Schreibweise: Formatiert jeder
 * Bildschirm selbst, stehen „23.94 %" neben „0,00 €" und drei Formate für dasselbe Datum.
 * Hier ist die eine Stelle, an der sie entstehen.
 */

/** @param {number | null | undefined} wert */
export function formatZahl(wert) {
	return (wert ?? 0).toLocaleString('de-DE');
}

/**
 * „23,9 %" — mit geschütztem Leerzeichen vor dem Zeichen (DIN 5008).
 * @param {number | string | null | undefined} wert
 */
export function formatProzent(wert) {
	const zahl = typeof wert === 'string' ? Number.parseFloat(wert) : (wert ?? 0);
	return (
		(Number.isFinite(zahl) ? zahl : 0).toLocaleString('de-DE', { maximumFractionDigits: 1 }) + ' %'
	);
}

/**
 * „19,92 €" — Beträge stehen in Forderungen und Bescheiden, deshalb immer zwei
 * Nachkommastellen und immer dieselbe Schreibweise.
 * @param {number | null | undefined} wert
 */
export function formatEuro(wert) {
	return (wert ?? 0).toLocaleString('de-DE', { style: 'currency', currency: 'EUR' });
}

/**
 * „TT.MM.JJJJ" — immer zweistellig, damit Spalten bündig bleiben.
 * @param {string | number | Date | null | undefined} wert
 */
export function formatDatum(wert) {
	if (!wert) return '';
	const d = new Date(wert);
	if (Number.isNaN(d.getTime())) return '';
	return d.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit', year: 'numeric' });
}

/**
 * „TT.MM.JJJJ, hh:mm" — Datum wie formatDatum, Uhrzeit ohne Sekunden.
 * @param {string | number | Date | null | undefined} wert
 */
export function formatZeitpunkt(wert) {
	if (!wert) return '';
	const d = new Date(wert);
	if (Number.isNaN(d.getTime())) return '';
	return d.toLocaleString('de-DE', {
		day: '2-digit',
		month: '2-digit',
		year: 'numeric',
		hour: '2-digit',
		minute: '2-digit'
	});
}

/**
 * Der Bestand eines Titels als Satz: „3 von 5 verfügbar", „Keine Exemplare",
 * „2 bestellt", und nichts, wenn niemand gezählt hat.
 *
 * Die drei Fälle sind der Grund für diese Funktion: Katalog (BuchKarte,
 * KlassenBuchKachel) und Theke zeigen dieselbe Zahl über denselben Titel, und jede eigene
 * Auslegung des dritten Falls wäre eine zu viel.
 *
 * `null`/`undefined` heißt nicht null: Nur die Suchabfragen liefern die Zahlen mit
 * (siehe repository/titel.go, Bestand/Verfuegbar als Zeiger); alle anderen lassen
 * sie weg. Über einen Titel, dessen Bestand niemand gezählt hat, schreibt die
 * Oberfläche gar nichts statt „Keine Exemplare".
 *
 * Steht nichts im Regal, aber etwas im Zulauf, heißt der Satz „2 bestellt" statt „Keine
 * Exemplare": Der Titel steht in der Trefferliste, weil der Zulauf als vorhanden zählt, und
 * „Keine Exemplare" schickte den Kollegen ins Regal.
 *
 * @param {number | null | undefined} gesamt
 * @param {number | null | undefined} verfuegbar
 * @param {number | null | undefined} [imZulauf] bestellt, noch nicht eingetroffen
 * @returns {string} leer, wenn keine Zahl vorliegt
 */
export function bestandSatz(gesamt, verfuegbar, imZulauf) {
	if (gesamt == null) return '';
	if (gesamt === 0 && imZulauf) return zulaufSatz(imZulauf);
	if (gesamt === 0) return 'Keine Exemplare';
	return `${verfuegbar ?? 0} von ${gesamt} verfügbar`;
}

/**
 * Der Zulauf eines Titels als Satz: „2 bestellt". Wo er zur Frage gehört, steht er neben dem
 * Bestand in denselben Worten wie beim Titel, der nur bestellt ist.
 *
 * @param {number | null | undefined} imZulauf bestellt, noch nicht eingetroffen
 * @returns {string} leer, wenn nichts bestellt ist
 */
export function zulaufSatz(imZulauf) {
	return imZulauf ? `${imZulauf} bestellt` : '';
}

/**
 * Der Jahrgang eines Titels als Text: „7" für ein Jahr, „7–10" für eine Spanne. Ohne Eintrag
 * ist er unbekannt und der Text leer. Dieselbe Regel wie im Ausdruck der Schulbuchliste
 * (pdf/schulbuecher.go, jahrgangText); beide Seiten lesen dieselben Prüffälle
 * (jahrgangSpanne.faelle.json).
 *
 * @param {number | string | null | undefined} von
 * @param {number | string | null | undefined} bis
 * @returns {string} leer, wenn kein Jahrgang eingetragen ist
 */
export function jahrgangSpanne(von, bis) {
	const v = Number(von);
	const b = Number(bis);
	if (!v || !b) return '';
	return v === b ? String(v) : `${v}–${b}`;
}
