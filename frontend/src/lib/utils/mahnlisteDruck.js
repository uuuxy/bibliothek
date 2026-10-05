// utils/mahnlisteDruck.js
// Baut das Druckdokument der Mahnliste: je überfälligem Buch eine Zeile, geordnet nach
// Klasse und Name. Gedruckt wird, was Reiter, Klassenfilter und Suche gerade zeigen; das
// Blatt zählt keine Mahnung.

import { baueListenDruckHtml } from './listenDruck.js';
import { ANSICHTEN, gemahntSatz } from '../mahnungen.js';

// Zahlen in der Klasse zählen als Zahl: 9A steht vor 10R.
const ORDNUNG = new Intl.Collator('de', { numeric: true });

/** @param {number} anzahl @param {string} eins @param {string} viele */
const zahlwort = (anzahl, eins, viele) => `${anzahl} ${anzahl === 1 ? eins : viele}`;

/**
 * @param {any[]} kinder Zeilen der Mahnliste, bereits gefiltert
 * @param {{ ansicht: string, klasse: string, suche: string }} sicht Was die Liste gerade zeigt
 * @param {Date} [jetzt] Druckdatum (Tests setzen es)
 * @returns {string} Vollständiges HTML-Dokument
 */
export function baueMahnlisteDruckHtml(kinder, sicht, jetzt = new Date()) {
	const geordnet = [...kinder].sort(
		(a, b) =>
			ORDNUNG.compare(a.klasse ?? '', b.klasse ?? '') || ORDNUNG.compare(a.name ?? '', b.name ?? '')
	);
	const zeilen = geordnet.flatMap((kind) =>
		(kind.medien ?? []).map((/** @type {any} */ m) => [
			kind.klasse ?? '',
			kind.name ?? '',
			m.titel ?? '',
			m.faellig_am ?? '',
			gemahntSatz(m.mahnstufe, m.letztes_mahndatum)
		])
	);

	// Die Zeile unter der Überschrift nennt den Ausschnitt: Ein Blatt aus „Eskaliert" oder
	// aus einer Klasse sähe sonst aus wie die ganze Liste.
	const meta = [
		`Erstellt am: ${jetzt.toLocaleDateString('de-DE')}`,
		`Ansicht: ${ANSICHTEN[sicht.ansicht] ?? sicht.ansicht}`,
		sicht.klasse ? `Klasse ${sicht.klasse}` : 'alle Klassen',
		sicht.suche.trim() ? `Suche: ${sicht.suche.trim()}` : '',
		`${zahlwort(geordnet.length, 'Kind', 'Kinder')}, ${zahlwort(zeilen.length, 'Buch', 'Bücher')}`
	]
		.filter(Boolean)
		.join(' | ');

	return baueListenDruckHtml({
		ueberschrift: 'Mahnliste',
		meta,
		spalten: [
			{ text: 'Klasse', klasse: 'schmal' },
			'Schüler/in',
			'Buch',
			{ text: 'Fällig seit', klasse: 'schmal' },
			{ text: 'Gemahnt', klasse: 'schmal' }
		],
		zeilen,
		// Je Buch eine Zeile: Die Liste einer Schule hat schnell einige hundert davon.
		dicht: true
	});
}
