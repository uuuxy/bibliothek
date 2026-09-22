/**
 * @file format.js
 * Zahlen, Prozent und Zeitpunkte in EINER deutschen Schreibweise.
 *
 * Anlass (Rundgang 01.09.2026): Die Statistik zeigte „23.94 %" (roh vom Backend)
 * neben „0,00 €" (de-DE), das Logbuch „1.9.2026, 21:05:03", die Inventur
 * „01.09.2026". Drei Formate für dieselbe Sache auf drei Bildschirmen — hier ist
 * die eine Stelle, an der sie entstehen.
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
	const zahl = typeof wert === 'string' ? parseFloat(wert) : (wert ?? 0);
	return (
		(Number.isFinite(zahl) ? zahl : 0).toLocaleString('de-DE', { maximumFractionDigits: 1 }) + ' %'
	);
}

/**
 * „19,92 €" — Beträge stehen in Forderungen und Bescheiden, deshalb immer zwei
 * Nachkommastellen und immer dieselbe Schreibweise. Bis zum 17.09.2026 trugen
 * StudentGebuehrenCard und StatsDashboard je eine eigene Kopie dieser Zeile.
 * @param {number | null | undefined} wert
 */
export function formatEuro(wert) {
	return (wert ?? 0).toLocaleString('de-DE', { style: 'currency', currency: 'EUR' });
}

/**
 * „01.09.2026" — immer zweistellig, damit Spalten bündig bleiben.
 * @param {string | number | Date | null | undefined} wert
 */
export function formatDatum(wert) {
	if (!wert) return '';
	const d = new Date(wert);
	if (Number.isNaN(d.getTime())) return '';
	return d.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit', year: 'numeric' });
}

/**
 * „01.09.2026, 21:05" — Datum wie formatDatum, Uhrzeit ohne Sekunden.
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
 * Die drei Fälle sind der Grund für diese Funktion. Bis zum 17.09.2026 stand die
 * Zeile zweimal im Katalog (BuchKarte, KlassenBuchKachel) — mit zwei verschiedenen
 * Antworten auf den dritten Fall. Die Theke braucht sie jetzt als dritte Stelle
 * (Protokoll des Medienzentrums vom 16.09.2026, Punkt 4), und drei Auslegungen
 * derselben Zahl über denselben Titel wären zwei zu viel.
 *
 * `null`/`undefined` heißt NICHT null: Nur die Suchabfragen liefern die Zahlen mit
 * (siehe repository/models.go, Bestand/Verfuegbar als Zeiger); alle anderen lassen
 * sie weg. Über einen Titel, dessen Bestand niemand gezählt hat, schreibt die
 * Oberfläche gar nichts statt „Keine Exemplare".
 *
 * Steht nichts im Regal, aber etwas im Zulauf, heißt der Satz „2 bestellt" statt „Keine
 * Exemplare" (docs/OFFEN.md 5.5, 22.09.2026): Der Titel steht in der Trefferliste, WEIL
 * der Zulauf als vorhanden zählt — und „Keine Exemplare" schickte den Kollegen ins Regal.
 *
 * @param {number | null | undefined} gesamt
 * @param {number | null | undefined} verfuegbar
 * @param {number | null | undefined} [imZulauf] bestellt, noch nicht eingetroffen
 * @returns {string} leer, wenn keine Zahl vorliegt
 */
export function bestandSatz(gesamt, verfuegbar, imZulauf) {
	if (gesamt == null) return '';
	if (gesamt === 0 && imZulauf) return `${imZulauf} bestellt`;
	if (gesamt === 0) return 'Keine Exemplare';
	return `${verfuegbar ?? 0} von ${gesamt} verfügbar`;
}
