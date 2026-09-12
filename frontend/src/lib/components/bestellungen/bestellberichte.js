/** Auswahl und Adresse der Bestellberichte — reine Textbausteine und URL-Bau.
 *
 *  Getrennt von BestellBerichte.svelte, weil hier nichts angezeigt wird: Was in der
 *  Liste steht und welche Adresse der Download bekommt, ist eine Frage von Wörtern
 *  und Parametern. */
import { lastOfMonth } from '../../utils/dates.js';

const MONATE = [
	'Januar',
	'Februar',
	'März',
	'April',
	'Mai',
	'Juni',
	'Juli',
	'August',
	'September',
	'Oktober',
	'November',
	'Dezember'
];

/** Die Beschriftungen folgen dem Schalter „Preise im Bestellwesen".
 *
 *  Ohne Preise wäre „Lieferantenabrechnung" schlicht falsch — abgerechnet wird
 *  nichts, der Bericht listet dann Mengen. Und ein „Monatsbericht … mit Summe"
 *  verspricht eine Summe, die im PDF nicht mehr steht. Die Auswahl muss
 *  beschreiben, was herauskommt.
 *  @param {boolean} mitPreisen */
export function berichtOptionen(mitPreisen) {
	return [
		{
			value: 'monat',
			label: 'Monatsbericht',
			desc: mitPreisen
				? 'Alle Bestellungen eines Monats mit Titeln und Summe'
				: 'Alle Bestellungen eines Monats mit Titeln und Exemplarzahlen'
		},
		{
			value: 'jahr',
			label: 'Jahresbericht',
			desc: mitPreisen
				? 'Monatliche Übersicht + Aufteilung nach Lieferant'
				: 'Monatliche Übersicht + Aufteilung nach Lieferant (Mengen)'
		},
		{
			value: 'lieferant',
			label: mitPreisen ? 'Lieferantenabrechnung' : 'Lieferantenübersicht',
			desc: 'Alle Bestellungen bei einem Lieferanten in einem Zeitraum'
		}
	];
}

/**
 * Adresse des Berichts-PDFs.
 *
 * `mittel` engt den Bericht auf einen Topf ein (leer = beide und die Alt-Bestellungen
 * ohne Zuordnung). Die Überschrift des Blattes hängt der Server an — dort steht die eine
 * Beschriftung, die auch Chip und Anschreiben tragen (api/mittel_vermerk.go,
 * Paritäts-Gate).
 *
 * @param {{
 *   typ: string, monatJahr: string, jahr: string, vonDatum: string, bisDatum: string,
 *   lieferantId: string, suppliers: Array<{ id: string, name: string }>,
 *   mitPreisen: boolean, mittel?: string
 * }} eingabe
 */
export function berichtURL(eingabe) {
	const base = '/api/bestellhistorie/bericht';
	/** @param {URLSearchParams} params */
	const mitTopf = (params) => {
		if (eingabe.mittel) params.set('mittel', eingabe.mittel);
		return params;
	};
	if (eingabe.typ === 'monat') {
		const [y, m] = eingabe.monatJahr.split('-');
		const params = new URLSearchParams({
			von: `${eingabe.monatJahr}-01`,
			bis: lastOfMonth(eingabe.monatJahr),
			titel: `Monatsbericht ${MONATE[Number(m) - 1] ?? ''} ${y}`
		});
		return `${base}?${mitTopf(params)}`;
	}
	if (eingabe.typ === 'jahr') {
		const params = new URLSearchParams({
			von: `${eingabe.jahr}-01-01`,
			bis: `${eingabe.jahr}-12-31`,
			jahresansicht: 'true',
			titel: `Jahresbericht ${eingabe.jahr}`
		});
		return `${base}?${mitTopf(params)}`;
	}
	const name = eingabe.suppliers.find((s) => s.id === eingabe.lieferantId)?.name ?? 'Lieferant';
	const params = new URLSearchParams({
		von: eingabe.vonDatum,
		bis: eingabe.bisDatum,
		lieferant_id: eingabe.lieferantId,
		titel: `${eingabe.mitPreisen ? 'Lieferantenabrechnung' : 'Lieferantenübersicht'}: ${name}`
	});
	return `${base}?${mitTopf(params)}`;
}
