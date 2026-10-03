<script>
	import Button from '../ui/Button.svelte';
	import Suchfeld from '../ui/Suchfeld.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import { Printer } from '@lucide/svelte';
	import { labelStore } from '../../stores/labels.svelte.js';
	import { printQueue } from '../../stores/printQueue.svelte.js';
	import LabelBarcodeSchritt from './LabelBarcodeSchritt.svelte';
	import LabelLayoutOptionen from './LabelLayoutOptionen.svelte';
	import Select from '../ui/Select.svelte';
</script>

<div class="lg:col-span-7 space-y-6 text-left">
	{#if (printQueue.copies?.length ?? 0) > 0}
		{@const anzahl = printQueue.copies?.length ?? 0}
		<div
			class="animate-fade-in space-y-4 border-l-2 border-primary bg-primary-container/50 p-4 text-left text-on-primary-container"
		>
			<div class="flex items-start gap-2.5">
				<Printer class="h-4 w-4" aria-hidden="true" />
				<div>
					<h3 class="text-base font-medium">Aktiver Druckauftrag</h3>
					<p class="mt-1 text-xs leading-relaxed font-medium">
						<!-- Ohne Herkunft: Die Übergabe kommt aus dem Wareneingang, dem Nachdruck oder
						     der Buchakte — „aus der freigegebenen Lieferung" stimmte nur für den ersten. -->
						{anzahl === 1 ? '1 Etikett' : `${anzahl} Etiketten`} zum Drucken übernommen.
					</p>
				</div>
			</div>
			<Button variant="secondary" class="w-full" onclick={labelStore.resetPendingCopies}>
				Auswahl zurücksetzen / Anderes Buch wählen
			</Button>
		</div>
	{:else}
		<!-- Step 1: Selection -->
		<div class="space-y-4 border-b border-outline-variant py-5">
			<h3 class="text-base font-semibold text-on-surface-variant">1. Titel / Klassensatz wählen</h3>

			<!-- Tab selector for search vs class set -->
			<div class="space-y-3">
				<!-- Autocomplete search -->
				<div class="space-y-1.5">
					<span class="block text-xs font-medium text-on-surface-variant"
						>Buchtitel im Katalog suchen</span
					>
					<!-- Suchfeld-Rolle (Autocomplete), Spinner im nachlaufend-Snippet. -->
					<Suchfeld
						bind:wert={labelStore.searchVal}
						oninput={labelStore.handleSearchInput}
						platzhalter="Titel, Autor oder ISBN eingeben..."
						etikett="Buchtitel im Katalog suchen"
					>
						{#snippet nachlaufend()}
							{#if labelStore.isSearching}
								<Ladekreis size="sm" />
							{/if}
						{/snippet}
					</Suchfeld>

					{#if labelStore.searchResults.length > 0}
						<div class="relative">
							<div
								class="absolute right-0 left-0 z-20 mt-1 max-h-48 overflow-y-auto rounded-sm bg-surface-container shadow-xl"
							>
								{#each labelStore.searchResults as r, _i (_i)}
									<button
										onclick={() => labelStore.selectBookTitle(r)}
										class="flex w-full cursor-pointer flex-col gap-0.5 px-3.5 py-2.5 text-left"
									>
										<span class="text-xs font-bold text-on-surface">{r.titel}</span>
										<span class="text-label-small text-on-surface-variant"
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
					<div class="grow border-t border-outline-variant"></div>
					<span class="mx-3 shrink text-xs font-medium text-on-surface-variant">ODER</span>
					<div class="grow border-t border-outline-variant"></div>
				</div>

				<!-- Class Selection -->
				<div class="grid grid-cols-2 gap-3">
					<div class="space-y-1.5">
						<span class="block text-xs font-medium text-on-surface-variant">Aus Klasse laden</span>
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
						<span class="block text-xs font-medium text-on-surface-variant">Buch aus Klasse</span>
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
