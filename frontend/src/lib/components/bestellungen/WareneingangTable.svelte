<script>
	import { coverSrc } from '../../utils/coverSrc.js';
	import Tabelle from '../ui/Tabelle.svelte';
	import Kaestchen from '../ui/Kaestchen.svelte';

	/**
	 * @component WareneingangTable
	 * Rendert die Liste der erwarteten Lieferungen gruppiert nach Lieferant.
	 *
	 * @prop {any[]} incomingShipments - Array der erwarteten Lieferungen.
	 * @prop {number} totalItems - Gesamtanzahl der Exemplare.
	 * @prop {string[]} selectedExemplarIds - Bindable Array mit ausgewählten Exemplar-IDs.
	 */
	let { incomingShipments = [], totalItems = 0, selectedExemplarIds = $bindable([]) } = $props();

	let allSelected = $derived(
		selectedExemplarIds.length > 0 &&
			selectedExemplarIds.length ===
				incomingShipments.flatMap((/** @type {any} */ s) =>
					s.items.flatMap((/** @type {any} */ i) => i.exemplar_ids || [])
				).length
	);

	function toggleAll() {
		if (allSelected) {
			selectedExemplarIds = [];
		} else {
			selectedExemplarIds = incomingShipments.flatMap((s) =>
				s.items.flatMap((/** @type {any} */ i) => i.exemplar_ids || [])
			);
		}
	}

	/**
	 * @param {Event} e
	 * @param {string[]} ids
	 */
	function toggleItemSelection(e, ids) {
		const target = /** @type {HTMLInputElement} */ (e.target);
		if (target.checked) {
			selectedExemplarIds = [...selectedExemplarIds, ...ids];
		} else {
			selectedExemplarIds = selectedExemplarIds.filter((id) => !ids.includes(id));
		}
	}
</script>

{#snippet coverImage(item)}
	{@const quelle = coverSrc(item.cover_url, item.isbn)}
	{#if quelle}
		<img
			src={quelle}
			class="w-16 h-24 object-cover shadow-sm rounded border border-slate-200"
			alt="Cover"
			loading="lazy"
		/>
	{:else}
		<div
			class="w-16 h-24 bg-slate-100 rounded border border-slate-200 flex items-center justify-center text-slate-400 text-label-small text-center p-1 leading-tight"
		>
			Kein Cover
		</div>
	{/if}
{/snippet}

<div class="flex-1 flex flex-col min-h-0">
	<div class="flex items-center justify-between mb-3">
		<h3 class="text-base font-medium text-slate-500">
			Erwartete Positionen ({totalItems} Exemplare)
		</h3>
		{#if incomingShipments.length > 0}
			<button
				onclick={toggleAll}
				class="text-xs font-bold text-blue-600 hover:text-blue-700 cursor-pointer"
			>
				{allSelected ? 'Auswahl aufheben' : 'Alle auswählen'}
			</button>
		{/if}
	</div>

	<div class="flex-1 bg-slate-50/30 flex flex-col">
		<div class="overflow-y-auto max-h-[50vh] sm:max-h-[60vh] custom-scrollbar">
			{#if incomingShipments.length === 0}
				<div class="py-12 text-center text-sm font-medium text-slate-400">
					Keine Positionen im Zulauf.
				</div>
			{:else}
				{#each incomingShipments as group, _i (_i)}
					<div
						class="bg-slate-50/80 border-b border-slate-200 px-6 py-3 flex items-center justify-between sticky top-0 z-10 backdrop-blur-sm"
					>
						<div class="font-bold text-slate-800">{group.supplierName}</div>
						<div class="text-xs font-semibold text-slate-500">Bestellt am {group.date}</div>
					</div>
					<Tabelle beschriftung="Bestellte Exemplare im Zulauf">
						<tbody>
							{#each group.items as item, _i (_i)}
								{@const isSelected = item.exemplar_ids.every((/** @type {string} */ id) =>
									selectedExemplarIds.includes(id)
								)}
								<tr aria-selected={isSelected}>
									<td class="w-12">
										<Kaestchen
											checked={isSelected}
											onchange={(e) => toggleItemSelection(e, item.exemplar_ids || [])}
											aria-label="{item.titel} auswählen"
										/>
									</td>
									<td class="w-20 shrink-0">
										{@render coverImage(item)}
									</td>
									<td class="font-semibold">{item.titel}</td>
									<td class="text-right">
										<span
											class="inline-flex items-center justify-center min-w-14 h-14 px-2 rounded-xl bg-blue-50 text-blue-800 text-3xl font-extrabold shadow-inner border border-blue-200"
										>
											{item.menge}
										</span>
									</td>
								</tr>
							{/each}
						</tbody>
					</Tabelle>
				{/each}
			{/if}
		</div>
	</div>
</div>

<style>
	.custom-scrollbar::-webkit-scrollbar {
		width: 6px;
	}
	.custom-scrollbar::-webkit-scrollbar-track {
		background: transparent;
	}
	.custom-scrollbar::-webkit-scrollbar-thumb {
		background-color: #cbd5e1;
		border-radius: 6px;
	}
</style>
