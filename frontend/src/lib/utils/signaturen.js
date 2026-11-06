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

/** Je Signatur eine Anfrage: Wer im Menü auf und ab fährt, zeigt mehrmals auf dieselbe. */
const coverJeSignatur = new Map();

/**
 * Bis zu zwei Titel mit gespeichertem Cover unter einer Signatur, für die Vorschau am
 * Vorschlagsmenü. Dieselbe Abfrage wie die Regalansicht, mit kleiner Grenze und nur Titeln mit
 * Cover. Antwortet der Server nicht, bleibt die Liste leer und wird beim nächsten Zeigen neu
 * erfragt.
 *
 * @param {string} signatur
 * @returns {Promise<{ titel_id: string, titel: string, isbn: string, cover_url: string }[]>}
 */
export function ladeCoverZurSignatur(signatur) {
	let antwort = coverJeSignatur.get(signatur);
	if (!antwort) {
		const frage = new URLSearchParams({ signatur, limit: '2', nur_mit_cover: 'true' });
		// Ein Fehlschlag bleibt nicht im Speicher: Das nächste Zeigen fragt neu.
		const vergessen = () => {
			coverJeSignatur.delete(signatur);
			return [];
		};
		antwort = apiFetch(`/api/signaturen/buecher?${frage}`)
			.then(async (res) => {
				if (!res.ok) return vergessen();
				return (await res.json())?.buecher ?? [];
			})
			.catch(vergessen);
		coverJeSignatur.set(signatur, antwort);
	}
	return antwort;
}
