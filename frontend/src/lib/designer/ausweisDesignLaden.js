import { tick } from 'svelte';
import { apiFetch } from '../apiFetch.js';
import { toastStore } from '../stores/toastStore.svelte.js';
import { applyDesign, designWurdeGeladen } from './idDesignerStore.svelte.js';

/** @type {Promise<boolean> | null} */
let lauf = null;

/**
 * Lädt das zentral gespeicherte Ausweis-Design beim ersten Bedarf der Sitzung, für jeden
 * Weg, der Ausweise druckt. Bauteile derselben Seite teilen sich den laufenden Abruf; nach
 * einem gescheiterten versucht es der nächste Aufruf erneut. Still, weil die Akte auch an
 * der Theke lädt: Dort ginge die Meldung bei jedem Leser auf, ohne dass jemand druckt.
 * @returns {Promise<boolean>} ob das gespeicherte Design im Store steht
 */
export function ladeAusweisDesign() {
	if (designWurdeGeladen()) return Promise.resolve(true);
	lauf ??= abruf().finally(() => {
		lauf = null;
	});
	return lauf;
}

async function abruf() {
	try {
		const res = await apiFetch('/api/ausweis-layout');
		if (!res.ok) return false;
		applyDesign(await res.json());
		return true;
	} catch {
		return false;
	}
}

/** Die Meldung einer Seite, deren Bedienung vom Design abhängt (Karte oder Etikett). */
export function meldeDesignNichtGeladen() {
	toastStore.addToast(
		'Ausweis-Design nicht geladen: Ausweise lassen sich gerade nicht drucken.',
		'error'
	);
}

/**
 * Vor jedem Druck von Ausweisen: Ohne das gespeicherte Design käme die Karte mit den
 * Standardwerten aus dem Drucker. Fehlt es noch, wird es jetzt geladen; scheitert das, steht
 * die Meldung da und der Aufrufer druckt nicht.
 * @returns {Promise<boolean>} ob gedruckt werden kann
 */
export async function designFuerDruck() {
	const standSchon = designWurdeGeladen();
	if (!(await ladeAusweisDesign())) {
		toastStore.addToast('Nicht gedruckt: Das Ausweis-Design ist nicht geladen.', 'error');
		return false;
	}
	// Erst jetzt geladen: Die Karten zeichnen sich neu und holen Logo und Strichcode. Der
	// Druckdialog nähme sonst die Karte ohne ihre Bilder auf.
	if (!standSchon) {
		await tick();
		/** @type {NodeListOf<HTMLImageElement>} */
		const bilder = document.querySelectorAll('.print-card-box img');
		await Promise.all([...bilder].map((bild) => bild.decode().catch(() => {})));
	}
	return true;
}
