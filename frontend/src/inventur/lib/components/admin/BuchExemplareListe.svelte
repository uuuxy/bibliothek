<script>
	import { apiFetch, apiClient } from '../../../../lib/apiFetch.js';
	import { showToast } from '$lib/store.svelte.js';
	import { onMount } from 'svelte';

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
			const json = await res.json();
			exemplare = json.data || [];
		} catch (err) {
			error = err instanceof Error ? err.message : String(err);
		} finally {
			loading = false;
		}
	}

	/** @param {any} ex */
	async function deleteCopy(ex) {
		if (!confirm(`Möchtest du das Exemplar ${ex.barcode_id} wirklich unwiderruflich löschen?`))
			return;
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
		} catch (e) {
			showToast('Netzwerkfehler beim Löschen.', 'error');
		}
	}
</script>

<div class="mt-8 border-t border-gray-100 pt-6">
	<h3 class="text-lg font-semibold text-gray-900 mb-4">Exemplare ({exemplare.length})</h3>

	{#if loading}
		<div class="text-sm text-gray-500 py-4 flex items-center justify-center">Lade Exemplare...</div>
	{:else if error}
		<div class="text-sm text-red-600 py-4">{error}</div>
	{:else if exemplare.length === 0}
		<div class="text-sm text-gray-500 py-4 italic text-center">
			Keine Exemplare in der Datenbank vorhanden. (Gesamtbestand: {formular.stock})
		</div>
	{:else}
		<div class="space-y-2 max-h-64 overflow-y-auto pr-2 custom-scrollbar">
			{#each exemplare as ex, _i (_i)}
				<div
					class="flex items-center justify-between p-3 bg-gray-50 rounded-lg border border-gray-100"
				>
					<div class="flex items-center gap-3">
						<span
							class="text-xs font-bold text-blue-700 bg-blue-50 border border-blue-100 px-2 py-1 rounded font-mono"
						>
							{ex.barcode_id}
						</span>
						<span
							class="text-[10px] font-bold px-2 py-0.5 rounded-full {!ex.ist_ausleihbar
								? 'bg-rose-50 text-rose-700 border border-rose-100'
								: !ex.ist_verfuegbar
									? 'bg-amber-50 text-amber-700 border border-amber-100'
									: 'bg-emerald-50 text-emerald-700 border border-emerald-100'}"
						>
							{!ex.ist_ausleihbar ? 'Gesperrt' : !ex.ist_verfuegbar ? 'Ausgeliehen' : 'Verfügbar'}
						</span>
						{#if ex.zustand_notiz}
							<span
								class="text-[10px] text-gray-500 truncate max-w-[150px]"
								title={ex.zustand_notiz}>{ex.zustand_notiz}</span
							>
						{/if}
					</div>
					<button
						title="Exemplar löschen"
						class="p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded transition-colors cursor-pointer"
						onclick={() => deleteCopy(ex)}
					>
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"
							><path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
							/></svg
						>
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
		background: #cbd5e1;
		border-radius: 4px;
	}
	.custom-scrollbar::-webkit-scrollbar-thumb:hover {
		background: #94a3b8;
	}
</style>
