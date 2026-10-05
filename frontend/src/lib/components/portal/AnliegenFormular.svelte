<!-- @component AnliegenFormular — das Formular zur gewählten Art eines Anliegens.

     Ein Wunsch gilt oft einem Buch, das es im Bestand noch nicht gibt, deshalb freier Text.
     Ein Problem betrifft meist ein vorhandenes Buch, deshalb schlägt das Feld dort Titel
     aus dem Katalog vor. Gespeichert wird in beiden Fällen der Text. -->
<script>
	import { apiFetch } from '../../apiFetch.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import Abschnitt from '../ui/Abschnitt.svelte';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import BuchVorschlagFeld from './BuchVorschlagFeld.svelte';

	/** @type {{ art: 'wunsch' | 'meldung', onabgeschickt: () => void | Promise<void>, onabbrechen: () => void }} */
	let { art, onabgeschickt, onabbrechen } = $props();

	const istWunsch = $derived(art === 'wunsch');
	let titelText = $state('');
	let klasse = $state('');
	let kommentar = $state('');
	let sending = $state(false);
	/** @type {HTMLInputElement | undefined} */
	let erstesFeld = $state();

	// Der Knopf, der das Formular geöffnet hat, ist nicht mehr da; der Fokus geht ins erste Feld.
	$effect(() => erstesFeld?.focus());

	// Ein Problem ohne Beschreibung nennt nur ein Buch; beim Wunsch ist die Anmerkung freiwillig.
	const vollstaendig = $derived(titelText.trim() !== '' && (istWunsch || kommentar.trim() !== ''));

	async function absenden() {
		if (!vollstaendig || sending) return;
		sending = true;
		try {
			const res = await apiFetch('/api/anliegen', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					art,
					titel_text: titelText.trim(),
					klasse: klasse.trim(),
					kommentar: kommentar.trim()
				})
			});
			if (!res.ok) {
				const data = await res.json().catch(() => null);
				throw new Error(data?.error || 'Anliegen konnte nicht gesendet werden.');
			}
			toastStore.addToast(
				istWunsch ? 'Wunsch ist bei der Bibliothek.' : 'Meldung ist bei der Bibliothek.',
				'success'
			);
			await onabgeschickt();
		} catch (err) {
			toastStore.addToast(/** @type {any} */ (err).message || String(err), 'error');
		} finally {
			sending = false;
		}
	}
</script>

<div>
	<Abschnitt titel={istWunsch ? 'Buchwunsch' : 'Problem melden'} />
	<div class="flex flex-col gap-4">
		{#if istWunsch}
			<Feld
				bind:value={titelText}
				bind:element={erstesFeld}
				label="Welches Buch?"
				type="text"
				maxlength={300}
				placeholder="z. B. Markl Biologie 2, ISBN falls bekannt"
			/>
		{:else}
			<BuchVorschlagFeld
				bind:value={titelText}
				bind:element={erstesFeld}
				label="Welches Buch?"
				maxlength={300}
				placeholder="Titel, Autor oder ISBN"
			/>
		{/if}
		<div class="grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-3">
			<Feld
				bind:value={klasse}
				label="Klasse / Kurs"
				type="text"
				maxlength={50}
				placeholder="z. B. 8G3"
			/>
			<Feld
				bind:value={kommentar}
				label={istWunsch ? 'Anmerkung (optional)' : 'Was stimmt nicht?'}
				type="text"
				maxlength={1000}
				placeholder={istWunsch
					? 'Was die Bibliothek sonst noch wissen sollte'
					: 'z. B. falsche Auflage bekommen, Seiten fehlen'}
				class="sm:col-span-2"
			/>
		</div>
		<div class="flex justify-end gap-2">
			<Button variant="secondary" onclick={onabbrechen} disabled={sending}>Abbrechen</Button>
			<Button onclick={absenden} disabled={sending || !vollstaendig}>
				{sending ? 'Wird gesendet …' : 'Absenden'}
			</Button>
		</div>
	</div>
</div>
