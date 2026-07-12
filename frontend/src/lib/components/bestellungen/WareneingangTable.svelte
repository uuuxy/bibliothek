<script>
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
	{#if item.isbn && item.cover_url}
		<img
			src="/api/images/cover?isbn={item.isbn}&url={encodeURIComponent(item.cover_url)}"
			class="w-16 h-24 object-cover shadow-sm rounded border border-slate-200"
			alt="Cover"
			loading="lazy"
		/>
	{:else}
		<div
			class="w-16 h-24 bg-slate-100 rounded border border-slate-200 flex items-center justify-center text-slate-400 text-[10px] text-center p-1 leading-tight"
		>
			Kein Cover
		</div>
	{/if}
{/snippet}

<div class="flex-1 flex flex-col min-h-0">
	<div class="flex items-center justify-between mb-3">
		<h3 class="text-xs font-bold text-slate-400 uppercase tracking-wider">
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
					<table class="w-full text-left text-sm border-collapse">
						<tbody class="divide-y divide-slate-100 bg-white">
							{#each group.items as item, _i (_i)}
								{@const isSelected = item.exemplar_ids.every((/** @type {string} */ id) =>
									selectedExemplarIds.includes(id)
								)}
								<tr
									class="hover:bg-blue-50/30 transition-colors {isSelected ? 'bg-blue-50/50' : ''}"
								>
									<td class="pl-6 pr-3 py-4 w-12">
										<input
											type="checkbox"
											class="w-4 h-4 text-blue-600 border-slate-300 rounded focus:ring-blue-500 cursor-pointer"
											checked={isSelected}
											onchange={(e) => toggleItemSelection(e, item.exemplar_ids || [])}
										/>
									</td>
									<td class="px-3 py-4 w-20 shrink-0">
										{@render coverImage(item)}
									</td>
									<td class="px-3 py-4 text-slate-800 font-semibold text-base">{item.titel}</td>
									<td class="px-6 py-4 text-right">
										<span
											class="inline-flex items-center justify-center min-w-14 h-14 px-2 rounded-xl bg-blue-50 text-blue-800 text-3xl font-extrabold shadow-inner border border-blue-200"
										>
											{item.menge}
										</span>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
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
