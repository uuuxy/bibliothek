<script>
	import { omniboxStore } from '../stores/omnibox.svelte.js';
	import Button from './ui/Button.svelte';
	import { ClipboardCheck } from '@lucide/svelte';
	import { escapeSchliesst } from './ui/escapeSchliesst.js';

	/**
	 * Zubehör-Checkliste beim Geräte-Scan: Der Server unterbricht mit
	 * type=geraet_check; erst die Bestätigung hier schickt den Scan mit
	 * confirmed_checklist erneut — dieselbe Mechanik wie der Sperr-Override.
	 * Gilt für Ausleihe UND Rückgabe (fehlt ein Teil bei der Rückgabe, wird
	 * abgebrochen und der Schaden am Profil gemeldet).
	 * @type {{ onReload: () => void }}
	 */
	let { onReload } = $props();

	const teile = $derived(
		(omniboxStore.checklistAnfrage?.geraet?.zubehoer ?? '')
			.split(',')
			.map((/** @type {string} */ t) => t.trim())
			.filter(Boolean)
	);

	function bestaetigen() {
		const anfrage = omniboxStore.checklistAnfrage;
		if (!anfrage) return;
		omniboxStore.checklistAnfrage = null;
		omniboxStore.queryVal = anfrage.query;
		omniboxStore.submitAction(null, onReload, false, true);
	}
</script>

{#if omniboxStore.checklistAnfrage}
	<div
		class="fixed inset-0 bg-black/40 backdrop-blur-sm z-100 flex items-center justify-center p-4"
	>
		<div
			class="bg-white rounded-3xl p-8 max-w-md w-full shadow-2xl"
			use:escapeSchliesst={() => (omniboxStore.checklistAnfrage = null)}
		>
			<div class="flex items-center gap-3 mb-2">
				<ClipboardCheck class="h-8 w-8 text-primary" aria-hidden="true" />
				<h2 class="text-xl font-bold text-on-surface">Zubehör prüfen</h2>
			</div>
			<p class="text-sm text-on-surface-variant mb-4">
				<strong>{omniboxStore.checklistAnfrage.geraet?.modellname}</strong>
				({omniboxStore.checklistAnfrage.geraet?.barcode_id}) — bitte prüfen, ob alles dabei ist:
			</p>

			<ul class="mb-6 space-y-2">
				{#each teile as teil, _i (_i)}
					<li class="flex items-center gap-2 text-sm text-on-surface">
						<span
							class="inline-flex h-5 w-5 items-center justify-center rounded-md bg-secondary-container text-on-secondary-container text-xs font-bold"
							aria-hidden="true">✓</span
						>
						<span>{teil}</span>
					</li>
				{/each}
			</ul>

			<div class="flex justify-end gap-2">
				<Button variant="secondary" onclick={() => (omniboxStore.checklistAnfrage = null)}>
					Abbrechen
				</Button>
				<Button variant="primary" onclick={bestaetigen}>Alles vollständig — weiter</Button>
			</div>
		</div>
	</div>
{/if}
