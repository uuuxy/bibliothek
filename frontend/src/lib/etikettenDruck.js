// Der Druck eines Bogens im Druck-Center: Auftrag schicken, PDF öffnen, Druck gegenbuchen. Eine
// LMF-Nummer passt nicht auf das Etikett; lehnt der Server sie ab, fragt der Druck, ob das
// Exemplar eine kurze Nummer bekommt, und druckt mit ihr.

import { apiFetch } from './apiFetch.js';
import { bestaetigen } from './stores/bestaetigung.svelte.js';
import { snackbars } from './stores/snackbars.svelte.js';

/** @typedef {{ barcode_id?: string, titel?: string, autor?: string }} DruckEtikett */
/** @typedef {{ formatId: string, startFeld: number, alsQR: boolean }} DruckBogen */

const JSON_KOPF = { 'Content-Type': 'application/json' };

/**
 * Bucht den Druck gegen, damit die Nachdruck-Liste die Exemplare loswird.
 *
 * Gemessen wird am erzeugten PDF, nicht am Papier — mehr weiss der Browser nicht. Wer
 * den Ausdruck abbricht, findet das Exemplar über die Suche im Nachdruck wieder.
 *
 * Scheitert der Vermerk, bleibt der Druck trotzdem gültig: Das PDF ist bereits offen,
 * und ein Fehler an dieser Stelle darf ihn nicht als fehlgeschlagen erscheinen lassen.
 * Der Preis ist ein Exemplar, das erneut auf der Liste steht — harmlos gegenüber einem
 * Etikett, das niemand mehr nachdruckt.
 * @param {DruckEtikett[]} gedruckt
 */
async function vermerkeGedruckt(gedruckt) {
	const barcodes = gedruckt.map((l) => l.barcode_id).filter(Boolean);
	if (barcodes.length === 0) return;
	try {
		await apiFetch('/api/exemplare/etiketten-gedruckt', {
			method: 'POST',
			headers: JSON_KOPF,
			body: JSON.stringify({ barcode_ids: barcodes })
		});
	} catch (err) {
		console.error('Etiketten konnten nicht als gedruckt vermerkt werden', err);
	}
}

/** @param {DruckBogen} bogen @param {DruckEtikett[]} etiketten */
function sendeBogen(bogen, etiketten) {
	return apiFetch('/api/print/labels', {
		method: 'POST',
		headers: JSON_KOPF,
		body: JSON.stringify({
			formatId: bogen.formatId,
			startPosition: bogen.startFeld,
			isQR: bogen.alsQR,
			items: etiketten.map((l) => ({ BarcodeID: l.barcode_id, Titel: l.titel, Autor: l.autor }))
		})
	});
}

/**
 * Fragt, ob die Exemplare mit diesen LMF-Nummern kurze Nummern bekommen, und lässt sie mit Ja
 * vergeben. M3, Dialogs: „Basic dialogs can give people the ability to provide confirmation of
 * a choice before committing to it“; das alte Etikett ist danach nicht mehr zu scannen.
 * @param {string[]} lang
 * @returns {Promise<Record<string, string> | null>} zu jeder alten Nummer die neue; null, wenn
 *   abgebrochen oder gescheitert (dann gemeldet)
 */
async function vergibNeueNummern(lang) {
	const eins = lang.length === 1;
	const ja = await bestaetigen({
		titel: eins ? 'Neue Nummer vergeben?' : `${lang.length} neue Nummern vergeben?`,
		text: eins
			? `Die Nummer ${lang[0]} passt nicht auf das Etikett. Das Exemplar bekommt eine kurze Nummer; sein altes Etikett lässt sich danach nicht mehr scannen. Das neue Etikett gehört über das alte.`
			: `${lang.length} Exemplare tragen eine LMF-Nummer, die nicht auf das Etikett passt. Sie bekommen kurze Nummern; ihre alten Etiketten lassen sich danach nicht mehr scannen. Die neuen Etiketten gehören über die alten.`,
		aktion: eins ? 'Neue Nummer vergeben' : 'Neue Nummern vergeben'
	});
	if (!ja) return null;
	const res = await apiFetch('/api/exemplare/neue-nummern', {
		method: 'POST',
		headers: JSON_KOPF,
		body: JSON.stringify({ barcode_ids: lang })
	});
	const antwort = await res.json().catch(() => ({}));
	if (!res.ok || !antwort.nummern) {
		snackbars.addToast(antwort.error || 'Die neuen Nummern ließen sich nicht vergeben.', 'error');
		return null;
	}
	return antwort.nummern;
}

/**
 * Druckt die Etiketten als Bogen im PDF und bucht den Druck gegen. Lehnt der Server
 * LMF-Nummern ab (409, lange_nummern), fragt er nach kurzen; mit ihnen meldet er die neuen
 * Nummern über nummernErsetzt zurück und druckt.
 * @param {DruckBogen} bogen
 * @param {DruckEtikett[]} etiketten
 * @param {(zuordnung: Record<string, string>) => void} nummernErsetzt
 */
export async function druckeBogen(bogen, etiketten, nummernErsetzt) {
	if (etiketten.length === 0) return;
	try {
		let auftrag = etiketten;
		let res = await sendeBogen(bogen, auftrag);
		if (res.status === 409) {
			const abgelehnt = await res.json().catch(() => ({}));
			const lang = Array.isArray(abgelehnt.lange_nummern) ? abgelehnt.lange_nummern : [];
			if (lang.length === 0) {
				snackbars.addToast(abgelehnt.error || 'Fehler beim Erstellen des PDFs', 'error');
				return;
			}
			const zuordnung = await vergibNeueNummern(lang);
			if (!zuordnung) return;
			nummernErsetzt(zuordnung);
			auftrag = auftrag.map((l) =>
				l.barcode_id && zuordnung[l.barcode_id] ? { ...l, barcode_id: zuordnung[l.barcode_id] } : l
			);
			res = await sendeBogen(bogen, auftrag);
		}

		if (res.ok) {
			const blob = await res.blob();
			window.open(window.URL.createObjectURL(blob), '_blank');
			await vermerkeGedruckt(auftrag);
		} else {
			console.error('Fehler beim Erstellen des PDFs');
			snackbars.addToast('Fehler beim Erstellen des PDFs', 'error');
		}
	} catch (err) {
		console.error('Netzwerkfehler beim Drucken', err);
		snackbars.addToast('Fehler beim Senden der Daten', 'error');
	}
}
