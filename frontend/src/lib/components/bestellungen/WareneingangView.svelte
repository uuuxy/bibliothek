<script>
	import { apiPost } from '../../apiFetch.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import WareneingangTable from './WareneingangTable.svelte';
	import PageContainer from '../layout/PageContainer.svelte';
	import Button from '../ui/Button.svelte';

	let { incomingShipments = [], onBack, onReceived } = $props();

	let isSubmitting = $state(false);
	/** @type {string[]} */
	let selectedExemplarIds = $state([]);

	let totalItems = $derived(
		incomingShipments.reduce(
			(/** @type {any} */ sum, /** @type {any} */ s) =>
				sum + s.items.reduce((/** @type {any} */ s2, /** @type {any} */ i) => s2 + i.menge, 0),
			0
		)
	);

	async function handleBulkReceive() {
		if (selectedExemplarIds.length === 0) return;

		isSubmitting = true;
		try {
			const data = await apiPost('/api/bestellungen/bulk-receive', {
				exemplar_ids: selectedExemplarIds
			});
			toastStore.addToast(`${data.received_count} Exemplare erfolgreich eingebucht!`, 'success');
			selectedExemplarIds = [];
			onReceived(data.received_items ?? []);
		} catch {
			// API handler shows error toast
		} finally {
			isSubmitting = false;
		}
	}
</script>

<div class="w-full h-full flex flex-col gap-6 bg-white p-6 animate-fade-in">
	<!-- Back Button & Header -->
	<div
		class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-4 border-b border-slate-100"
	>
		<div class="space-y-3">
			<button
				onclick={onBack}
				class="inline-flex items-center gap-1 text-xs font-bold text-slate-500 hover:text-slate-800 transition-colors cursor-pointer group"
			>
				<span class="transform group-hover:-translate-x-0.5 transition-transform">←</span> Zurück zur
				Übersicht
			</button>
			<div>
				<h2 class="text-xl sm:text-2xl font-black text-slate-900 tracking-tight">
					Wareneingang bearbeiten
				</h2>
				<p class="mt-1 text-sm text-slate-500">
					Wähle die Positionen aus, die eingetroffen sind und ins System aufgenommen werden sollen.
				</p>
			</div>
		</div>

		<div class="flex items-center justify-end sm:self-end">
			<Button
				size="lg"
				onclick={handleBulkReceive}
				disabled={isSubmitting || selectedExemplarIds.length === 0}
				class="px-6"
			>
				{#if isSubmitting}
					<div
						class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"
					></div>
				{/if}
				Ausgewählte Positionen einbuchen
			</Button>
		</div>
	</div>

	<!-- Content/Table Section -->
	<WareneingangTable {incomingShipments} {totalItems} bind:selectedExemplarIds />
</div>
