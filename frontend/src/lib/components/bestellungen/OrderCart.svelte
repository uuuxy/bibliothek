<script>
	import { orderStore } from '../../stores/orderStore.svelte.js';
</script>

<div class="space-y-3">
	<span class="text-sm font-medium text-slate-600">Warenkorb</span>
	{#if !orderStore.cart.length}
		<div
			class="py-10 border border-dashed border-slate-200 rounded-lg text-center text-base text-slate-400"
		>
			Der Warenkorb ist leer. Suche nach Büchern zum Hinzufügen.
		</div>
	{:else}
		<div class="border border-slate-100 rounded-lg overflow-hidden divide-y divide-slate-100">
			{#each orderStore.cart as item, idx (idx)}
				<div class="p-3 bg-slate-50/30 flex items-center justify-between gap-4 text-base">
					<div class="flex items-center gap-3 min-w-0">
						{#if item.cover_url}<img
								src="/api/images/cover?isbn={item.isbn || ''}&url={encodeURIComponent(
									item.cover_url
								)}"
								class="w-8 aspect-3/4 object-cover rounded-sm"
								alt=""
							/>{:else}<div
								class="w-8 aspect-3/4 rounded bg-slate-200 flex items-center justify-center font-bold text-sm uppercase"
							>
								{item.titel.charAt(0)}
							</div>{/if}
						<div class="min-w-0">
							<h4 class="font-bold text-slate-800 truncate">{item.titel}</h4>
							<p class="text-sm text-slate-400 truncate">ISBN: {item.isbn}</p>
							{#if item.generate_barcodes}
								<div
									class="text-[10px] font-bold text-blue-600 mt-1 flex items-center gap-1 bg-blue-50 w-fit px-1.5 py-0.5 rounded-md"
								>
									🔖 {item.menge}
									{item.menge === 1 ? 'Barcode' : 'Barcodes'} reserviert
								</div>
							{/if}
						</div>
					</div>
					<div class="flex items-center gap-4">
						<div class="flex items-center gap-2">
							<span class="text-sm font-semibold text-slate-400">€</span>
							<input
								type="number"
								step="0.01"
								bind:value={item.preis}
								class="w-20 px-2 py-1 border border-slate-200 rounded-md text-right font-semibold text-slate-700 focus:outline-none focus:border-blue-400 focus:ring-1 focus:ring-blue-400"
							/>
						</div>
						<div
							class="flex items-center border border-slate-200 bg-white rounded-md overflow-hidden"
						>
							<button
								onclick={() => (item.menge = Math.max(1, item.menge - 1))}
								aria-label="Menge verringern"
								class="px-2 py-0.5 hover:bg-slate-50 font-bold text-slate-500">-</button
							><span class="px-3 font-bold text-slate-700 min-w-[20px] text-center"
								>{item.menge}</span
							><button
								onclick={() => (item.menge += 1)}
								aria-label="Menge erhöhen"
								class="px-2 py-0.5 hover:bg-slate-50 font-bold text-slate-500">+</button
							>
						</div>
						<button
							onclick={() => orderStore.removeFromCart(idx)}
							class="text-slate-400 hover:text-rose-500 cursor-pointer">Löschen</button
						>
					</div>
				</div>
			{/each}
		</div>
		<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mt-4">
			<div class="text-lg font-bold text-slate-800">
				Gesamtsumme: {orderStore.total.toFixed(2).replace('.', ',')} €
			</div>
			<div class="flex flex-col sm:flex-row items-end sm:items-center gap-4">
				<label
					class="flex items-center gap-2 cursor-pointer bg-white px-3 py-2 border border-slate-200 rounded-lg"
				>
					<input
						type="checkbox"
						bind:checked={orderStore.attachBarcodes}
						class="w-4 h-4 text-blue-600 rounded border-slate-300 focus:ring-blue-500"
					/>
					<span class="text-sm font-bold text-slate-700">Barcodes mitschicken</span>
				</label>
				<button
					onclick={() => orderStore.submitOrder()}
					disabled={orderStore.submitting || !orderStore.selectedSupplier}
					class="px-5 py-2.5 rounded-lg bg-blue-600 hover:bg-blue-700 text-white font-bold text-base cursor-pointer disabled:bg-slate-200 disabled:text-slate-400 flex items-center gap-2"
				>
					{#if orderStore.submitting}
						<div
							class="w-4 h-4 border-2 border-t-white border-white/20 rounded-full animate-spin"
						></div>
						Bestellung wird gesendet...
					{:else}
						📤 Bestellung auslösen ({orderStore.totalQty} Expl.)
					{/if}
				</button>
			</div>
		</div>
	{/if}
</div>
