<script>
	import { orderStore } from '../../stores/orderStore.svelte.js';
</script>

<div class="space-y-3">
	<div class="flex items-center justify-between">
		<span class="text-xs font-bold text-slate-500 uppercase tracking-wider">Warenkorb</span>
		{#if orderStore.cart.length}
			<span class="text-xs font-bold text-slate-500 bg-slate-100 rounded-full px-2 py-0.5 tabular-nums"
				>{orderStore.totalQty} Expl.</span
			>
		{/if}
	</div>

	{#if !orderStore.cart.length}
		<div
			class="py-10 px-4 border border-dashed border-slate-200 rounded-xl text-center text-sm text-slate-400"
		>
			<div class="text-2xl mb-1.5">🛒</div>
			Noch nichts ausgewählt.<br />
			Tippe links bei einem Titel auf <span class="font-bold text-slate-500">+</span> oder suche oben.
		</div>
	{:else}
		<div class="space-y-2">
			{#each orderStore.cart as item, idx (idx)}
				<div class="rounded-xl border border-slate-200 bg-white p-3 space-y-2.5">
					<div class="flex items-start gap-2.5">
						{#if item.cover_url}<img
								src="/api/images/cover?isbn={item.isbn || ''}&url={encodeURIComponent(item.cover_url)}"
								class="w-8 aspect-3/4 object-cover rounded-sm shrink-0 ring-1 ring-slate-200/70"
								alt=""
							/>{:else}<div
								class="w-8 aspect-3/4 rounded-sm bg-slate-200 flex items-center justify-center font-bold text-xs uppercase shrink-0"
							>
								{item.titel.charAt(0)}
							</div>{/if}
						<div class="min-w-0 flex-1">
							<h4 class="font-semibold text-slate-900 text-sm truncate leading-snug">{item.titel}</h4>
							<p class="text-xs text-slate-400 truncate font-mono">{item.isbn || '—'}</p>
							{#if item.generate_barcodes}
								<div
									class="text-[10px] font-bold text-blue-600 mt-1 flex items-center gap-1 bg-blue-50 w-fit px-1.5 py-0.5 rounded-md"
								>
									🔖 {item.menge} {item.menge === 1 ? 'Barcode' : 'Barcodes'}
								</div>
							{/if}
						</div>
						<button
							onclick={() => orderStore.removeFromCart(idx)}
							aria-label="Entfernen"
							class="shrink-0 w-6 h-6 rounded-full text-slate-400 hover:text-rose-500 hover:bg-rose-50 flex items-center justify-center cursor-pointer transition-colors"
						>
							<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
								<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
							</svg>
						</button>
					</div>

					<div class="flex items-center justify-between gap-2 pl-10">
						<div class="flex items-center border border-slate-200 bg-white rounded-lg overflow-hidden">
							<button
								aria-label="Menge verringern"
								onclick={() => (item.menge = Math.max(1, item.menge - 1))}
								class="px-2.5 py-1 hover:bg-slate-50 font-bold text-slate-500 cursor-pointer">−</button
							><span class="px-2 font-bold text-slate-800 text-sm min-w-6 text-center tabular-nums"
								>{item.menge}</span
							><button
								aria-label="Menge erhöhen"
								onclick={() => (item.menge += 1)}
								class="px-2.5 py-1 hover:bg-slate-50 font-bold text-slate-500 cursor-pointer">+</button
							>
						</div>
						<div class="flex items-center gap-1.5">
							<input
								type="number"
								step="0.01"
								bind:value={item.preis}
								aria-label="Preis"
								class="w-20 px-2 py-1 border border-slate-200 rounded-lg text-right text-sm font-semibold text-slate-700 focus:outline-none focus:border-blue-400 focus:ring-1 focus:ring-blue-400"
							/>
							<span class="text-sm font-semibold text-slate-400">€</span>
						</div>
					</div>
				</div>
			{/each}
		</div>

		<!-- Footer: Summe + CTA -->
		<div class="pt-3 mt-1 border-t border-slate-100 space-y-3">
			<div class="flex items-center justify-between">
				<span class="text-sm font-semibold text-slate-500">Gesamt</span>
				<span class="text-xl font-bold text-slate-900 tabular-nums"
					>{orderStore.total.toFixed(2).replace('.', ',')} €</span
				>
			</div>
			<label
				class="flex items-center gap-2 cursor-pointer bg-slate-50 px-3 py-2 border border-slate-200 rounded-xl select-none"
			>
				<input
					type="checkbox"
					bind:checked={orderStore.attachBarcodes}
					class="w-4 h-4 text-blue-600 rounded border-slate-300 focus:ring-blue-500"
				/>
				<span class="text-sm font-semibold text-slate-700">Barcodes mitschicken</span>
			</label>
			<button
				onclick={() => orderStore.submitOrder()}
				disabled={orderStore.submitting || !orderStore.selectedSupplier}
				class="w-full px-5 py-3 rounded-xl bg-blue-600 hover:bg-blue-700 text-white font-bold text-sm cursor-pointer disabled:bg-slate-200 disabled:text-slate-400 disabled:cursor-not-allowed flex items-center justify-center gap-2 active:scale-[0.99] transition-all shadow-sm"
			>
				{#if orderStore.submitting}
					<div class="w-4 h-4 border-2 border-t-white border-white/20 rounded-full animate-spin"></div>
					Wird gesendet …
				{:else}
					Bestellung auslösen · {orderStore.totalQty} Expl.
				{/if}
			</button>
			{#if !orderStore.selectedSupplier}
				<p class="text-[11px] text-center text-amber-600 font-medium">Bitte zuerst einen Lieferanten wählen.</p>
			{/if}
		</div>
	{/if}
</div>
