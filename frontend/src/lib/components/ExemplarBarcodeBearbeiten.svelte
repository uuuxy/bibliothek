<script>
	import { apiFetch, apiClient } from '../apiFetch.js';
	import Button from './ui/Button.svelte';
	import Feld from './ui/Feld.svelte';

	/**
	 * Barcode-Editor eines Exemplars: Feld, „Interne ID generieren" und „Speichern".
	 * Setzt bei Erfolg ex.barcode_id und schließt via onDone, wie ExemplarStatusBearbeiten.
	 * @type {{ ex: any, vorgabe: string, onDone: () => void }}
	 */
	let { ex, vorgabe, onDone } = $props();

	// Arbeitskopie, einmal aus dem Prop befüllt: Als $derived überschriebe jede Änderung an
	// vorgabe, was gerade getippt oder gescannt wird.
	// svelte-ignore state_referenced_locally
	let neuerBarcode = $state(vorgabe);
	let barcodeFehler = $state('');
	/** @type {HTMLInputElement | undefined} */
	let feld = $state();
	// „Speichern" bleibt bei leerem Feld bedienbar: Der Klick nennt am Feld, was fehlt, bis
	// etwas darin steht (wie `verlangt` in BuchFormular).
	let verlangt = $state(false);
	const hinweis = $derived(
		verlangt && !neuerBarcode.trim()
			? 'Bitte einen Barcode scannen oder eintragen. Gespeichert wird erst mit ihm.'
			: barcodeFehler
	);

	async function speichereBarcode() {
		const barcode = neuerBarcode.trim();
		verlangt = true;
		if (!barcode) return feld?.focus();
		if (barcode === ex.barcode_id) return onDone();
		barcodeFehler = '';
		try {
			const res = await apiClient.put(`/api/buecher/exemplare/${ex.id}/barcode`, { barcode });
			if (res.ok) {
				ex.barcode_id = barcode;
				onDone();
			} else {
				const fehlerAntwort = await res.json().catch(() => ({}));
				barcodeFehler = fehlerAntwort.error || 'Fehler beim Speichern';
			}
		} catch {
			barcodeFehler = 'Netzwerkfehler';
		}
	}

	async function generateInternalId() {
		try {
			const res = await apiFetch('/api/barcode/next');
			if (res.ok) {
				const daten = await res.json();
				neuerBarcode = daten.next_barcode;
				barcodeFehler = '';
			} else {
				barcodeFehler = 'Fehler beim Generieren der ID';
			}
		} catch {
			barcodeFehler = 'Netzwerkfehler';
		}
	}
</script>

<!-- autofocus bewusst: Das Feld erscheint erst auf Klick und ersetzt an dieser Stelle den
     Barcode. Wer es öffnet, will sofort tippen oder scannen. -->
<Feld
	bind:value={neuerBarcode}
	bind:element={feld}
	aria-label="Barcode des Exemplars"
	autofocus
	onfocus={(e) => e.currentTarget.select()}
	ungueltig={!!hinweis}
	hint={hinweis}
	feld="font-mono"
	onkeydown={(e) => {
		if (e.key === 'Enter') speichereBarcode();
		if (e.key === 'Escape') onDone();
	}}
/>
<div class="mt-1 flex gap-2">
	<Button variant="secondary" size="sm" onclick={generateInternalId} class="text-label-small">
		Interne ID generieren
	</Button>
	<Button size="sm" onclick={speichereBarcode} class="text-label-small">Speichern</Button>
</div>
