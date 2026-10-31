import { tick } from 'svelte';
import { apiFetch, leseApiFehler } from '../apiFetch.js';
import { fehlertext } from '../utils/fehlertext.js';
import { snackbars } from '../stores/snackbars.svelte.js';
import {
	applyDesign,
	designWurdeGeladen,
	idStore,
	PLATZHALTER_ADRESSE,
	PLATZHALTER_SCHULNAME
} from './idDesignerStore.svelte.js';

/** @type {Promise<boolean> | null} */
let lauf = null;
let druckWirdVorbereitet = false;
// Der Grund des letzten gescheiterten Abrufs, für die Meldung dessen, der drucken will.
let grund = '';

/**
 * Lädt das zentral gespeicherte Ausweis-Design beim ersten Bedarf der Sitzung, für jeden
 * Weg, der Ausweise druckt. Bauteile derselben Seite teilen sich den laufenden Abruf; nach
 * einem gescheiterten versucht es der nächste Aufruf erneut. Still, weil die Akte auch an
 * der Theke lädt: Dort ginge die Meldung bei jedem Leser auf, ohne dass jemand druckt.
 * @returns {Promise<boolean>} ob das gespeicherte Design im Store steht
 */
export function ladeAusweisDesign() {
	if (designWurdeGeladen()) return Promise.resolve(true);
	return holeDesign();
}

/** @param {{ timeoutMs: number }} [optionen] */
function holeDesign(optionen) {
	lauf ??= abruf(optionen).finally(() => {
		lauf = null;
	});
	return lauf;
}

/** @param {{ timeoutMs: number }} [optionen] */
async function abruf(optionen) {
	try {
		const res = await apiFetch('/api/ausweis-layout', optionen);
		if (!res.ok) {
			grund = await leseApiFehler(res);
			return false;
		}
		applyDesign(await res.json());
		grund = '';
		return true;
	} catch (e) {
		grund = fehlertext(e);
		return false;
	}
}

/** „: <Grund>" für die Meldungen, ohne doppelten Punkt; leer, wenn keiner bekannt ist. */
const mitGrund = () => (grund ? `: ${grund.replace(/\.$/, '')}` : '');

/** Die Meldung einer Seite, deren Bedienung vom Design abhängt (Karte oder Etikett). */
export function meldeDesignNichtGeladen() {
	snackbars.addToast(
		`Ausweis-Design nicht geladen${mitGrund()}. Ausweise lassen sich gerade nicht drucken.`,
		'error'
	);
}

/** Steht auf der Vorderseite noch die Musterschule? Dann fehlt die Schule in den Einstellungen. */
function tragtMusterkopf() {
	return idStore.front.elements.some(
		(e) =>
			e.show &&
			((e.id === 'header' && e.content === PLATZHALTER_SCHULNAME) ||
				(e.id === 'address' && e.content === PLATZHALTER_ADRESSE))
	);
}

/**
 * Vor jedem Druck von Ausweisen: Ohne das gespeicherte Design käme die Karte mit den
 * Standardwerten aus dem Drucker. Geholt wird es jedes Mal: Eine Seite bleibt an der Theke
 * tagelang offen, und Design und Schule ändern sich an anderen Plätzen. Bleibt die Antwort
 * aus, gilt der Stand, der schon da ist; gibt es keinen, steht die Meldung da und der Aufrufer
 * druckt nicht.
 * @returns {Promise<boolean>} ob gedruckt werden kann
 */
export async function designFuerDruck() {
	// Das Holen dauert: Ein zweiter Klick davor brächte nach dem ersten Druckdialog einen zweiten.
	if (druckWirdVorbereitet) return false;
	druckWirdVorbereitet = true;
	try {
		return await bereiteDruckVor();
	} finally {
		druckWirdVorbereitet = false;
	}
}

async function bereiteDruckVor() {
	// Mit einem Stand im Speicher wartet der Druck nicht die lange Frist von apiFetch ab.
	const optionen = designWurdeGeladen() ? { timeoutMs: 3000 } : undefined;
	if (!(await holeDesign(optionen)) && !designWurdeGeladen()) {
		snackbars.addToast(
			`Nicht gedruckt, das Ausweis-Design ist nicht geladen${mitGrund()}.`,
			'error'
		);
		return false;
	}
	// Die Karten zeichnen sich neu und holen Logo und Strichcode. Der Druckdialog nähme sonst die
	// Karte ohne ihre Bilder auf.
	await tick();
	/** @type {NodeListOf<HTMLImageElement>} */
	const bilder = document.querySelectorAll('.print-card-box img');
	await Promise.all([...bilder].map((bild) => bild.decode().catch(() => {})));
	// Gedruckt wird trotzdem: Der Druckdialog zeigt die Karte, und wer gerade drucken muss, soll es
	// können. Ohne den Satz fiele die falsche Schule erst an der fertigen Karte auf.
	if (tragtMusterkopf()) {
		snackbars.addToast(
			'Auf dem Ausweis steht noch die Musterschule. Name und Anschrift fehlen in den Einstellungen unter „Schule“.',
			'warning'
		);
	}
	return true;
}
