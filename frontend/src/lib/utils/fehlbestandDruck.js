// utils/fehlbestandDruck.js
// Baut das Druckdokument zum Fehlbestand einer Inventur: die Exemplare, die noch fehlen, in
// der Reihenfolge des Regals und mit einem Kästchen zum Abhaken. Titel und Autor laufen um,
// statt gekürzt zu werden: Auf dem Papier führt kein Weg zum abgeschnittenen Rest.

import { baueListenDruckHtml } from './listenDruck.js';

/**
 * @param {any[]} offene Die Exemplare, die noch fehlen, in der Reihenfolge des Berichts
 * @param {{ label?: string, gebucht: number }} bericht Name der Inventur und die Zahl der
 *   Exemplare, die sie als Verlust gebucht hat
 * @param {Date} [jetzt] Druckdatum (Tests setzen es)
 * @returns {string} Vollständiges HTML-Dokument
 */
export function baueFehlbestandDruckHtml(offene, { label = '', gebucht }, jetzt = new Date()) {
	const geklaert = gebucht - offene.length;
	// Die Zeile unter der Überschrift nennt, was nicht auf dem Blatt steht: Ein Blatt mit
	// weniger Zeilen, als der Bericht zählt, sähe sonst aus wie ein unvollständiger Ausdruck.
	const meta = [
		`Erstellt am: ${jetzt.toLocaleDateString('de-DE')}`,
		`Als Verlust gebucht: ${gebucht}`,
		geklaert > 0 ? `bereits geklärt: ${geklaert}` : '',
		`noch offen: ${offene.length}`,
		'nach Signatur sortiert'
	]
		.filter(Boolean)
		.join(' | ');

	return baueListenDruckHtml({
		ueberschrift: label ? `Fehlbestand — ${label}` : 'Fehlbestand',
		meta,
		// Titel und Autor teilen sich die Breite; die kurzen Spalten nehmen die ihres Inhalts.
		spalten: [
			{ text: 'Signatur', klasse: 'schmal' },
			'Titel',
			'Autor',
			{ text: 'Barcode', klasse: 'schmal' },
			{ text: 'Gefunden', klasse: 'schmal' }
		],
		zeilen: offene.map((e) => [
			e.signatur || '—',
			e.titel ?? '',
			e.autor ?? '',
			{ text: e.barcode_id ?? '', klasse: 'mono' },
			{ text: '', klasse: 'kaestchen' }
		])
	});
}
