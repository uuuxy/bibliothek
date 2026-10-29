// Der Druck eines Bogens im Druck-Center: Auftrag schicken, PDF öffnen, Druck gegenbuchen.

import { apiFetch, leseApiFehler } from '../../apiFetch.js';
import { snackbars } from '../../stores/snackbars.svelte.js';
import { fehlertext } from '../../utils/fehlertext.js';
import { zeigePdf, PDF_BLOCKIERT } from '../../utils/pdfImFenster.js';
import { startfeldAufBogen } from './etikettformate.js';

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
 * Etikett, das niemand mehr nachdruckt. Die Warnung sagt es, sonst druckte es jemand zweimal.
 * @param {DruckEtikett[]} gedruckt
 */
async function vermerkeGedruckt(gedruckt) {
	const barcodes = gedruckt.map((l) => l.barcode_id).filter(Boolean);
	if (barcodes.length === 0) return;
	let grund = '';
	try {
		const res = await apiFetch('/api/exemplare/etiketten-gedruckt', {
			method: 'POST',
			headers: JSON_KOPF,
			body: JSON.stringify({ barcode_ids: barcodes })
		});
		if (!res.ok) grund = await leseApiFehler(res);
	} catch (err) {
		grund = fehlertext(err);
	}
	if (!grund) return;
	snackbars.addToast(
		`Nicht als gedruckt vermerkt: ${grund.replace(/\.$/, '')}. Die Etiketten stehen weiter unter „Fehlende Etiketten".`,
		'warning'
	);
}

/**
 * Der Auftrag an den Server für das PDF des Bogens. Die Vorschau schickt denselben Auftrag, damit
 * sie das Blatt zeigt, das gedruckt wird.
 * @param {DruckBogen} bogen @param {DruckEtikett[]} etiketten
 */
export function sendeBogen(bogen, etiketten) {
	return apiFetch('/api/print/labels', {
		method: 'POST',
		headers: JSON_KOPF,
		body: JSON.stringify({
			formatId: bogen.formatId,
			startPosition: startfeldAufBogen(bogen.startFeld, bogen.formatId),
			isQR: bogen.alsQR,
			items: etiketten.map((l) => ({ BarcodeID: l.barcode_id, Titel: l.titel, Autor: l.autor }))
		})
	});
}

/**
 * Druckt die Etiketten als Bogen im PDF und bucht den Druck gegen.
 * @param {DruckBogen} bogen
 * @param {DruckEtikett[]} etiketten
 */
export async function druckeBogen(bogen, etiketten) {
	if (etiketten.length === 0) return;
	try {
		const res = await zeigePdf(() => sendeBogen(bogen, etiketten));
		if (!res) {
			// Ohne offenes PDF ist nichts gedruckt; die Etiketten bleiben auf der Liste.
			snackbars.addToast(PDF_BLOCKIERT, 'warning');
		} else if (res.ok) {
			await vermerkeGedruckt(etiketten);
		} else {
			console.error('Fehler beim Erstellen des PDFs');
			snackbars.addToast(await leseApiFehler(res, 'Fehler beim Erstellen des PDFs'), 'error');
		}
	} catch (err) {
		console.error('Netzwerkfehler beim Drucken', err);
		snackbars.addToast('Fehler beim Senden der Daten', 'error');
	}
}
