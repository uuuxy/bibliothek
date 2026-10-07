// utils/ausleiherDruck.js
// Baut das Druckdokument der Ausleiher-Liste.
//
// Steht bewusst als eigene Funktion neben der Komponente und nicht in ihr: Nur so
// lässt sich das fertige Dokument prüfen. Ein Test des Maskier-Helfers allein belegt
// nichts über diesen Pfad — er belegt nur, dass der Helfer maskiert, nicht, dass er
// an jeder Einsetzstelle auch aufgerufen wird (ausleiherDruck.test.js hält das zu).

import { baueListenDruckHtml } from './listenDruck.js';
import { fmtDateDE } from './dates.js';
import { istUeberfaellig } from './ueberfaellig.js';

/**
 * @param {any[]} ausleiher Bereits gefilterte Zeilen
 * @param {any} buch
 * @param {string} filterKlasse
 * @param {Date} [jetzt] Vergleichszeitpunkt für die Überfälligkeit (Tests setzen ihn)
 * @returns {string} Vollständiges HTML-Dokument
 */
export function baueAusleiherDruckHtml(ausleiher, buch, filterKlasse, jetzt = new Date()) {
	const zeilen = ausleiher.map((b) => {
		// Dauerleihe (Kollegium): keine Frist, nie überfällig — wie in der Akte.
		const dauerleihe = !!b.ist_dauerleihe;
		const ueberfaellig = istUeberfaellig(b, jetzt);
		return [
			`${b.schueler_name ?? ''} ${b.schueler_nachname ?? ''}`,
			b.klasse || '-',
			{ text: b.exemplar_barcode ?? '', klasse: 'mono' },
			fmtDateDE(b.ausgeliehen_am),
			{
				text: dauerleihe ? 'ohne Frist' : fmtDateDE(b.rueckgabe_frist),
				klasse: ueberfaellig ? 'overdue' : ''
			}
		];
	});

	return baueListenDruckHtml({
		// Das Blatt nennt alle Ausleiher des Titels, nicht nur die mit überschrittener Frist.
		ueberschrift: `Ausleiher-Liste: ${buch?.title || 'Buch'}`,
		meta: `Erstellt am: ${jetzt.toLocaleDateString('de-DE')} | Filter: Klasse ${filterKlasse}`,
		// Nur der Name bricht um; die kurzen Spalten nehmen die Breite ihres Inhalts.
		spalten: [
			'Schüler/in',
			{ text: 'Klasse', klasse: 'schmal' },
			{ text: 'Exemplar', klasse: 'schmal' },
			{ text: 'Ausgeliehen am', klasse: 'schmal' },
			{ text: 'Rückgabe bis', klasse: 'schmal' }
		],
		zeilen
	});
}
