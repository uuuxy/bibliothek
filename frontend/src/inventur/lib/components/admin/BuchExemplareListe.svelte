<script>
	import { apiFetch } from '../../../../lib/apiFetch.js';
	import { loeschenBestaetigen } from '../../../../lib/stores/bestaetigung.svelte.js';
	import { showToast } from '$lib/store.svelte.js';
	import { onMount } from 'svelte';
	import { Trash2 } from '@lucide/svelte';
	import StatusChip from '../../../../lib/components/ui/StatusChip.svelte';

	let { formular = $bindable() } = $props();

	/** @type {any[]} */
	let exemplare = $state([]);
	let loading = $state(true);
	let error = $state('');

	onMount(() => {
		loadExemplare();
	});

	async function loadExemplare() {
		if (!formular.id) return;
		loading = true;
		error = '';
		try {
			const res = await apiFetch(`/api/buecher/titel/${formular.id}/exemplare`, {
				credentials: 'include'
			});
			if (!res.ok) {
				const err = await res.json().catch(() => ({}));
				throw new Error(err.error || 'Fehler beim Laden der Exemplare');
			}
			// Die Antwort ist das nackte Array (RespondJSON), wie Buchakte und Druck-Center
			// sie lesen — bis zum 23.09.2026 stand hier `json.data`, und die Liste blieb leer.
			exemplare = (await res.json()) ?? [];
		} catch (err) {
			error = err instanceof Error ? err.message : String(err);
		} finally {
			loading = false;
		}
	}

	/** @param {any} ex */
	async function deleteCopy(ex) {
		if (!(await loeschenBestaetigen(`Exemplar ${ex.barcode_id} löschen?`))) return;
		try {
			const res = await apiFetch(`/api/buecher/exemplare/${ex.id}`, {
				method: 'DELETE',
				credentials: 'include'
			});
			if (res.ok) {
				exemplare = exemplare.filter((e) => e.id !== ex.id);
				// Also decrement stock in the main form so it's accurate!
				formular.stock = Math.max(0, Number(formular.stock) - 1);
				showToast('Exemplar erfolgreich gelöscht', 'success');
			} else {
				const err = await res.json().catch(() => ({}));
				showToast(err.error || 'Fehler beim Löschen des Exemplars.', 'error');
			}
		} catch {
			showToast('Netzwerkfehler beim Löschen.', 'error');
		}
	}
</script>

<div class="mt-8 border-t border-outline-variant pt-6">
	<h3 class="text-lg font-semibold text-on-surface mb-4">Exemplare ({exemplare.length})</h3>

	{#if loading}
		<div class="text-sm text-on-surface-variant py-4 flex items-center justify-center">
			Lade Exemplare...
		</div>
	{:else if error}
		<div class="text-sm text-error py-4">{error}</div>
	{:else if exemplare.length === 0}
		<div class="text-sm text-on-surface-variant py-4 italic text-center">
			Keine Exemplare in der Datenbank vorhanden. (Gesamtbestand: {formular.stock})
		</div>
	{:else}
		<div class="space-y-2 max-h-64 overflow-y-auto pr-2 custom-scrollbar">
			{#each exemplare as ex, _i (_i)}
				<!-- Dieselben Exemplare zeigt die Buchakte (BookExemplarCard): umrandete Fläche in
				     outline-variant, Barcode als getönte Chip-Form, Zustand über StatusChip. Zwei
				     Ansichten desselben Exemplars sollen nicht zwei Farbsprachen sprechen. -->
				<div class="flex items-center justify-between p-3 rounded-lg border border-outline-variant">
					<div class="flex items-center gap-3">
						<span
							class="rounded-md px-2 py-0.5 font-mono text-xs font-bold whitespace-nowrap bg-primary-container text-on-primary-container"
						>
							{ex.barcode_id}
						</span>
						<StatusChip
							ton={!ex.ist_ausleihbar ? 'fehler' : !ex.ist_verfuegbar ? 'warten' : 'erfolg'}
							text={!ex.ist_ausleihbar
								? 'Gesperrt'
								: !ex.ist_verfuegbar
									? 'Ausgeliehen'
									: 'Verfügbar'}
						/>
						{#if ex.zustand_notiz}
							<span
								class="text-label-small text-on-surface-variant truncate max-w-37.5"
								title={ex.zustand_notiz}>{ex.zustand_notiz}</span
							>
						{/if}
					</div>
					<button
						title="Exemplar löschen"
						aria-label="Exemplar löschen"
						class="icon-btn text-on-surface-variant hover:text-error focus-visible:ring-2 focus-visible:ring-primary focus:outline-none"
						onclick={() => deleteCopy(ex)}
					>
						<Trash2 class="w-4 h-4" aria-hidden="true" />
					</button>
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
	.custom-scrollbar::-webkit-scrollbar {
		width: 4px;
	}
	.custom-scrollbar::-webkit-scrollbar-track {
		background: transparent;
	}
	.custom-scrollbar::-webkit-scrollbar-thumb {
		background: var(--color-outline-variant);
		border-radius: 4px;
	}
	.custom-scrollbar::-webkit-scrollbar-thumb:hover {
		background: var(--color-outline);
	}
</style>
