import { apiFetch } from '../apiFetch.js';

/**
 * Die im Bestand vorkommenden Signaturen — die Regaladressen der Schülerbücherei und der
 * Lernmittel, wie sie aus Littera übernommen wurden („Sk", „JF", „MANGA", „LMF Deu 7").
 *
 * Ein Abruf für alle Felder, die eine Signatur entgegennehmen (Buchformular, Bestellkorb),
 * aus dem Bestand: Ein Vorschlag wie „BIB {Kategorie}" wäre ein zweites Vokabular neben dem,
 * das physisch auf den Büchern klebt.
 *
 * Antwortet der Server nicht, bleibt die Liste leer: Das Feld nimmt weiter freien Text an,
 * es fehlen nur die Vorschläge.
 *
 * @returns {Promise<{ signatur: string, titel: number, exemplare: number }[]>}
 */
export async function ladeSignaturen() {
	try {
		const res = await apiFetch('/api/signaturen');
		if (!res.ok) {
			return [];
		}
		return (await res.json()) ?? [];
	} catch {
		return [];
	}
}
