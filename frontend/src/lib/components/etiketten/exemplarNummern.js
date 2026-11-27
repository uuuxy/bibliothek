// Welches Exemplar eines Titels eine Eingabe im Nummernfeld des Etikettendrucks meint.
import { dekodiereLitteraEtikett } from '../../utils/litteraEtikett.js';
import { ohnePruefzeichen } from '../../utils/code39Pruefzeichen.js';

/**
 * Die Nummern, unter denen das Nummernfeld ein Exemplar findet: seine eigene und ein altes
 * LMF-Etikett, klein geschrieben. Groß und klein zählen nicht: Das Feld sucht nur Nummern eines
 * Titels.
 * @param {{ barcode_id?: unknown, lmf_barcode?: unknown }} exemplar
 * @returns {string[]}
 */
export function nummernDes(exemplar) {
	return [exemplar.barcode_id, exemplar.lmf_barcode]
		.filter(Boolean)
		.map((nummer) => String(nummer).toLowerCase());
}

/**
 * Das Exemplar zu einem Scan nach der Regel der Theke (am Server service.ExemplarKandidaten und
 * FindeExemplar): der Scan, das zurückgerechnete Littera-Etikett, zuletzt der Scan ohne
 * Prüfzeichen, jeder Kandidat gegen die Nummer und ein altes LMF-Etikett. Ein Teil trifft nichts.
 * @template {{ barcode_id?: unknown, lmf_barcode?: unknown }} T
 * @param {T[]} exemplare
 * @param {string} scan schon durch normalisiereScan
 * @returns {T | undefined}
 */
export function exemplarZumScan(exemplare, scan) {
	for (const kandidat of [scan, dekodiereLitteraEtikett(scan), ohnePruefzeichen(scan)]) {
		const gesucht = kandidat?.toLowerCase();
		const exemplar = gesucht ? exemplare.find((e) => nummernDes(e).includes(gesucht)) : undefined;
		if (exemplar) return exemplar;
	}
	return undefined;
}
