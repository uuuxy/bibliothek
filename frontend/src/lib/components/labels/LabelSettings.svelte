<script>
	import { Printer } from '@lucide/svelte';
	import { labelStore } from '../../stores/labels.svelte.js';
	import { printQueue } from '../../stores/printQueue.svelte.js';
	import LabelBarcodeSchritt from './LabelBarcodeSchritt.svelte';
	import LabelLayoutOptionen from './LabelLayoutOptionen.svelte';
	import Select from '../ui/Select.svelte';
</script>

<div class="lg:col-span-5 space-y-6 text-left">
	{#if (printQueue.copies?.length ?? 0) > 0}
		<div class="p-4 border-l-2 border-blue-300 bg-blue-50/50 space-y-4 text-left animate-fade-in">
			<div class="flex items-start gap-2.5">
				<Printer class="h-4 w-4" aria-hidden="true" />
				<div>
					<h3 class="text-xs font-medium text-blue-800">Aktiver Druckauftrag</h3>
					<p class="text-xs text-blue-700 font-medium leading-relaxed mt-1">
						Es werden {printQueue.copies?.length ?? 0} Etiketten aus der freigegebenen Lieferung geladen.
					</p>
				</div>
			</div>
			<button
				onclick={labelStore.resetPendingCopies}
				class="w-full py-2 bg-white hover:bg-slate-50 border border-slate-200 text-slate-700 font-bold rounded-xl text-xs transition-colors cursor-pointer"
			>
				Auswahl zurücksetzen / Anderes Buch wählen
			</button>
		</div>
	{:else}
		<!-- Step 1: Selection -->
		<div class="py-5 space-y-4 border-b border-slate-200">
			<h3 class="text-xs text-blue-600 font-medium">1. Titel / Klassensatz wählen</h3>

			<!-- Tab selector for search vs class set -->
			<div class="space-y-3">
				<!-- Autocomplete search -->
				<div class="space-y-1.5">
					<span class="text-xs font-medium text-slate-500 block">Buchtitel im Katalog suchen</span>
					<div class="relative">
						<input
							type="text"
							bind:value={labelStore.searchVal}
							oninput={labelStore.handleSearchInput}
							placeholder="Titel, Autor oder ISBN eingeben..."
							class="w-full bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-sm text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500"
						/>
						{#if labelStore.isSearching}
							<div class="absolute right-3 top-1/2 -translate-y-1/2">
								<div
									class="w-3.5 h-3.5 border-2 border-blue-600 border-t-transparent rounded-full animate-spin"
								></div>
							</div>
						{/if}
					</div>

					{#if labelStore.searchResults.length > 0}
						<div class="relative">
							<div
								class="absolute left-0 right-0 mt-1 bg-surface-container rounded-sm shadow-xl z-20 max-h-48 overflow-y-auto divide-y divide-slate-50"
							>
								{#each labelStore.searchResults as r, _i (_i)}
									<button
										onclick={() => labelStore.selectBookTitle(r)}
										class="w-full text-left px-3.5 py-2.5 hover:bg-slate-50 transition-colors flex flex-col gap-0.5 cursor-pointer"
									>
										<span class="text-xs font-bold text-slate-900">{r.titel}</span>
										<span class="text-label-small text-slate-500"
											>{r.autor || 'Unbekannt'} · {r.verlag || 'Kein Verlag'}</span
										>
									</button>
								{/each}
							</div>
						</div>
					{/if}
				</div>

				<!-- Divider -->
				<div class="relative flex py-1 items-center">
					<div class="grow border-t border-slate-100"></div>
					<span class="shrink mx-3 text-xs text-slate-400 font-medium">ODER</span>
					<div class="grow border-t border-slate-100"></div>
				</div>

				<!-- Class Selection -->
				<div class="grid grid-cols-2 gap-3">
					<div class="space-y-1.5">
						<span class="text-xs font-medium text-slate-500 block">Aus Klasse laden</span>
						<Select
							bind:value={labelStore.selectedClass}
							options={labelStore.classGroups.map((/** @type {any} */ g) => ({
								value: g.className,
								label: g.className
							}))}
							onchange={() => labelStore.handleClassChange()}
							placeholder="Klasse wählen"
							aria-label="Aus Klasse laden"
						/>
					</div>

					<div class="space-y-1.5">
						<span class="text-xs font-medium text-slate-500 block">Buch aus Klasse</span>
						<Select
							disabled={!labelStore.selectedClass}
							options={labelStore.classBooks.map((/** @type {any} */ b) => ({
								value: String(b.id),
								label: b.title
							}))}
							onchange={(/** @type {string} */ id) => {
								const book = labelStore.classBooks.find(
									(/** @type {any} */ b) => String(b.id) === id
								);
								if (book) {
									labelStore.selectBookTitle({
										id: String(book.id),
										titel: book.title,
										autor: book.author
									});
								}
							}}
							placeholder="Buch wählen"
							aria-label="Buch aus Klasse"
						/>
					</div>
				</div>
			</div>
		</div>

		<LabelBarcodeSchritt />
	{/if}

	<LabelLayoutOptionen />
</div>
