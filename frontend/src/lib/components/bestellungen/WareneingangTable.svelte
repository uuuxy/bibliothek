<script>
	import BuchCover from '../ui/BuchCover.svelte';
	import Button from '../ui/Button.svelte';
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

<div>
	<div class="flex items-center justify-between mb-3">
		<h3 class="text-base font-medium text-on-surface-variant">
			Erwartete Positionen ({totalItems} Exemplare)
		</h3>
		{#if incomingShipments.length > 0}
			<Button variant="ghost" size="sm" onclick={toggleAll}>
				{allSelected ? 'Auswahl aufheben' : 'Alle auswählen'}
			</Button>
		{/if}
	</div>

	<!-- Ohne eigenen Scrollkasten: Die Positionen scrollen mit der Seite, der Name des
	     Lieferanten bleibt dabei oben stehen. -->
	<div>
		{#if incomingShipments.length === 0}
			<div class="py-12 text-center text-sm font-medium text-on-surface-variant">
				Keine Positionen im Zulauf.
			</div>
		{:else}
			{#each incomingShipments as group, _i (_i)}
				<div
					class="sticky top-0 z-10 flex items-center justify-between border-b border-outline-variant bg-surface/80 px-6 py-3 backdrop-blur-sm"
				>
					<div class="font-bold text-on-surface">{group.supplierName}</div>
					<div class="text-xs font-semibold text-on-surface-variant">Bestellt am {group.date}</div>
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
									<BuchCover
										coverUrl={item.cover_url}
										isbn={item.isbn}
										titel={item.titel}
										groesse="gross"
									/>
								</td>
								<td class="font-semibold">{item.titel}</td>
								<td class="text-right">
									<span
										class="inline-flex h-14 min-w-14 items-center justify-center rounded-xl bg-primary-container px-2 text-3xl font-extrabold text-on-primary-container"
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
