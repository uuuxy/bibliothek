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

/** Trennt Regaladresse und Kürzel in der Aufschrift, wie am Server (repository.SignaturTrenner). */
const SIGNATUR_TRENNER = ' / ';

/**
 * Die Aufschrift zu einer gewählten Regaladresse, wie Littera sie in der Vorgabe bildet: dazu
 * die ersten drei Buchstaben des Nachnamens („LMF Deu 7 / Bie", „Sk / McC"). Den Nachnamen
 * liest sie nur aus der Form „Nachname, Vorname", die DNB und Littera schreiben; ohne Komma
 * bleibt es bei der Regaladresse, das Kürzel tippt dann jemand dazu.
 * @param {string} regaladresse
 * @param {string | null | undefined} autor
 * @returns {string}
 */
export function signaturMitKuerzel(regaladresse, autor) {
	const komma = (autor ?? '').indexOf(',');
	const buchstaben = komma > 0 ? (autor?.slice(0, komma).match(/\p{L}/gu) ?? []) : [];
	if (buchstaben.length === 0) {
		return regaladresse;
	}
	return regaladresse + SIGNATUR_TRENNER + buchstaben.slice(0, 3).join('');
}

/** Je Signatur eine Anfrage in LEBENSDAUER_MS: Wer im Menü auf und ab fährt, zeigt mehrmals auf
 *  dieselbe. Danach fragt das nächste Zeigen neu, sonst fehlte ein neues Cover oder zeigte ein
 *  umgezogener Titel bis zum Neuladen der Seite. */
const LEBENSDAUER_MS = 60_000;
/** @type {Map<string, { seit: number, antwort: Promise<any[]> }>} */
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
	const eintrag = coverJeSignatur.get(signatur);
	let antwort = eintrag && Date.now() - eintrag.seit < LEBENSDAUER_MS ? eintrag.antwort : null;
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
		coverJeSignatur.set(signatur, { seit: Date.now(), antwort });
	}
	return antwort;
}
