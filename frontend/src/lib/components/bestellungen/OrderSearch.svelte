<script>
	import { apiPost, apiPut } from '../../apiFetch.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import { orderStore } from '../../stores/orderStore.svelte.js';
	import Button from '../ui/Button.svelte';
	import Select from '../ui/Select.svelte';
	import { coverSrc } from '../../utils/coverSrc.js';

	/** @type {any} */
	let stagedBook = $state(null);
	let stagedMenge = $state(1);
	let stagedGenerateBarcodes = $state(true);
	let resolvingDnb = $state(false);
	// Vergleichswert, um beim Bestätigen zu erkennen, ob die Signatur tatsächlich
	// bearbeitet wurde — unverändert übernommen wird nie ein zusätzlicher Request
	// ausgelöst (weder für einen unangetasteten Vorschlag noch für eine bereits
	// vorhandene Signatur).
	let stagedSignaturBeiStart = $state('');

	let localResults = $derived(orderStore.searchResults.filter((r) => r.source === 'local'));
	let dnbResults = $derived(orderStore.searchResults.filter((r) => r.source === 'dnb'));

	/** @param {any} book */
	async function openStaging(book) {
		if (book.source === 'dnb') {
			resolvingDnb = true;
			try {
				const localBook = await apiPost('/api/buecher/aus-isbn', { isbn: book.isbn });
				if (localBook && localBook.titel_id) {
					stageBook({
						id: localBook.titel_id,
						titel: localBook.titel,
						autor: localBook.autor,
						isbn: localBook.isbn,
						verlag: localBook.verlag,
						cover_url: localBook.cover_url,
						// exists=false: signatur ist hier nur ein VORSCHLAG aus der DNB-
						// Genre-/Altersheuristik (leer, wenn keine Kategorie erkannt wurde).
						signatur: localBook.signatur ?? '',
						// Der Preisvorschlag steht am DNB-Treffer, nicht am eben angelegten
						// lokalen Titel — sonst ginge er beim Umweg über /aus-isbn verloren.
						preis_vorschlag: book.preis_vorschlag
					});
				} else {
					toastStore.addToast('Fehler beim Anlegen des DNB-Buchs', 'error');
				}
			} catch {
				toastStore.addToast('Fehler beim Anlegen des DNB-Buchs', 'error');
			} finally {
				resolvingDnb = false;
			}
		} else {
			stageBook(book);
		}
	}

	/** @param {any} book */
	function stageBook(book) {
		stagedBook = book;
		stagedMenge = 1;
		stagedGenerateBarcodes = true;
		stagedSignaturBeiStart = book.signatur ?? '';
		orderStore.resetSearch();
	}

	async function confirmAddToCart() {
		// Signatur nur speichern, wenn tatsächlich bearbeitet — ein unangetasteter
		// Vorschlag steht bereits so in der DB (aus /aus-isbn), eine unangetastete
		// vorhandene Signatur soll erst recht nicht neu geschrieben werden.
		const neueSignatur = (stagedBook.signatur ?? '').trim();
		if (neueSignatur !== stagedSignaturBeiStart) {
			try {
				await apiPut(`/api/buecher/titel/${stagedBook.id}/signatur`, { signatur: neueSignatur });
			} catch {
				toastStore.addToast(
					'Signatur konnte nicht gespeichert werden — Titel wird trotzdem bestellt.',
					'error'
				);
			}
		}
		orderStore.addToCart(stagedBook, stagedMenge, stagedGenerateBarcodes);
		stagedBook = null;
	}
</script>

<div class="space-y-4">
	<div class="space-y-1.5">
		<label for="supplier" class="block text-xs font-medium text-slate-500">Lieferant</label>
		<Select
			id="supplier"
			bind:value={orderStore.selectedSupplierId}
			options={orderStore.suppliers.map((s) => ({
				value: s.id,
				label: `${s.name} (${s.customerNumber})`
			}))}
			placeholder="Kein Lieferant angelegt"
		/>
	</div>
	<div class="space-y-1.5 relative">
		<label for="book" class="block text-xs font-medium text-slate-500"
			>Titel suchen &amp; hinzufügen</label
		>
		<input
			id="book"
			type="text"
			bind:value={orderStore.searchQuery}
			oninput={() => orderStore.handleSearchInput()}
			placeholder="Titel, Autor oder ISBN …"
			class="w-full px-3 py-2.5 rounded-xl border border-slate-200 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400"
		/>
		{#if orderStore.showDropdown && (localResults.length > 0 || dnbResults.length > 0)}
			<div
				class="absolute z-10 w-full mt-1 bg-surface-container rounded-sm shadow-xl max-h-72 overflow-y-auto divide-y divide-slate-100"
			>
				{#if localResults.length > 0}
					<div
						class="bg-slate-50/80 px-3.5 py-2 text-xs font-medium text-slate-500 sticky top-0 backdrop-blur-xs z-5"
					>
						Im lokalen Bestand
					</div>
					{#each localResults as b, _i (_i)}
						{@const quelle = coverSrc(b.cover_url, b.isbn)}
						<button
							onclick={() => openStaging(b)}
							class="w-full text-left px-3.5 py-2.5 hover:bg-slate-50 border-b border-slate-100 last:border-0 flex items-center gap-3 text-base"
						>
							{#if quelle}<img
									src={quelle}
									class="w-7 aspect-3/4 object-cover rounded-sm"
									alt=""
								/>{:else}<div
									class="w-7 aspect-3/4 rounded bg-slate-200 flex items-center justify-center font-bold text-sm uppercase"
								>
									{b.titel.charAt(0)}
								</div>{/if}
							<div class="min-w-0 flex-1">
								<div class="font-bold text-slate-800 truncate">{b.titel}</div>
								<div class="text-sm text-slate-400 truncate">{b.autor} · {b.isbn}</div>
							</div>
							<span
								class="shrink-0 text-xs bg-emerald-50 text-emerald-700 px-2 py-0.5 rounded-full font-bold"
							>
								Bestand: {b.current_stock || 0}
							</span>
						</button>
					{/each}
				{/if}

				{#if dnbResults.length > 0}
					<div
						class="bg-slate-50/80 px-3.5 py-2 text-xs font-medium text-slate-500 sticky top-0 backdrop-blur-xs z-5"
					>
						Neu aus DNB (Externe Suche)
					</div>
					{#each dnbResults as b, _i (_i)}
						{@const isDuplicate =
							b.is_duplicate ||
							localResults.some(
								(l) => (l.isbn || '').replace(/-/g, '') === (b.isbn || '').replace(/-/g, '')
							)}
						{@const quelle = coverSrc(b.cover_url, b.isbn)}
						<button
							onclick={() => !isDuplicate && openStaging(b)}
							disabled={isDuplicate}
							class="w-full text-left px-3.5 py-2.5 flex items-center gap-3 text-base border-b border-slate-100 last:border-0 {isDuplicate
								? 'opacity-50 cursor-not-allowed bg-slate-50/30'
								: 'hover:bg-slate-50'}"
						>
							{#if quelle}<img
									src={quelle}
									class="w-7 aspect-3/4 object-cover rounded-sm"
									alt=""
								/>{:else}<div
									class="w-7 aspect-3/4 rounded bg-slate-200 flex items-center justify-center font-bold text-sm uppercase"
								>
									{b.titel.charAt(0)}
								</div>{/if}
							<div class="min-w-0 flex-1">
								<div class="font-bold text-slate-800 truncate">{b.titel}</div>
								<div class="text-sm text-slate-400 truncate">{b.autor} · {b.isbn}</div>
							</div>
							{#if isDuplicate}
								<span
									class="shrink-0 text-xs bg-slate-100 text-slate-500 px-2 py-0.5 rounded font-medium"
								>
									Vorhanden
								</span>
							{:else}
								<span
									class="shrink-0 text-label-small bg-amber-50 text-amber-700 px-2 py-0.5 rounded font-bold uppercase"
								>
									NEU
								</span>
							{/if}
						</button>
					{/each}
				{/if}
			</div>
		{/if}
		{#if orderStore.searchLoading}
			<div
				class="absolute z-10 w-full mt-1 bg-surface-container rounded-sm shadow-xl px-4 py-3 flex items-center gap-2 text-sm text-slate-500"
			>
				<div
					class="w-4 h-4 border-2 border-t-blue-500 border-blue-500/20 rounded-full animate-spin shrink-0"
				></div>
				Suche läuft...
			</div>
		{:else if resolvingDnb}
			<div
				class="absolute z-10 w-full mt-1 bg-surface-container rounded-sm shadow-xl px-4 py-3 flex items-center gap-2 text-sm text-slate-500"
			>
				<div
					class="w-4 h-4 border-2 border-t-blue-500 border-blue-500/20 rounded-full animate-spin shrink-0"
				></div>
				Titel wird im Katalog angelegt...
			</div>
		{/if}
	</div>
</div>

{#if stagedBook}
	{@const stagedQuelle = coverSrc(stagedBook.cover_url, stagedBook.isbn)}
	<div class="mt-3 p-4 rounded-xl border border-blue-200 bg-blue-50/60 space-y-3.5 animate-fade-in">
		<div class="flex items-center gap-3 min-w-0">
			{#if stagedQuelle}
				<img
					src={stagedQuelle}
					class="w-10 aspect-3/4 object-cover rounded shadow-sm border border-white shrink-0"
					alt=""
				/>
			{:else}
				<div
					class="w-10 aspect-3/4 rounded bg-slate-200 flex items-center justify-center font-bold text-sm uppercase shrink-0"
				>
					{stagedBook.titel.charAt(0)}
				</div>
			{/if}
			<div class="min-w-0">
				<div class="font-bold text-slate-900 text-sm truncate">{stagedBook.titel}</div>
				<div class="text-xs text-slate-500 truncate">{stagedBook.autor}</div>
			</div>
		</div>

		<div class="space-y-1">
			<label for="stagedSignaturInput" class="text-xs font-medium text-slate-500">
				Signatur
				{#if !stagedSignaturBeiStart}
					<span class="text-amber-600 font-normal">(Vorschlag, bitte prüfen)</span>
				{/if}
			</label>
			<input
				id="stagedSignaturInput"
				type="text"
				bind:value={stagedBook.signatur}
				placeholder="z. B. BIB Jugendbuch"
				class="w-full px-2.5 py-1.5 border border-slate-200 bg-white rounded-xl text-sm font-medium text-slate-700 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
			/>
		</div>

		<div class="flex items-center justify-between gap-3">
			<div class="flex items-center gap-2">
				<label for="stagedMengeInput" class="text-xs font-medium text-slate-500">Menge</label>
				<input
					id="stagedMengeInput"
					type="number"
					min="1"
					bind:value={stagedMenge}
					class="w-16 px-2 py-1.5 border border-slate-200 bg-white rounded-xl text-center font-bold text-slate-700 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
				/>
			</div>
			<label class="flex items-center gap-2 cursor-pointer select-none">
				<input
					type="checkbox"
					bind:checked={stagedGenerateBarcodes}
					class="w-4 h-4 text-blue-600 rounded border-slate-300 focus:ring-blue-500"
				/>
				<span class="text-xs font-semibold text-slate-700">Barcodes generieren</span>
			</label>
		</div>

		<div class="flex items-center gap-2">
			<Button variant="ghost" onclick={() => (stagedBook = null)}>Abbrechen</Button>
			<Button onclick={confirmAddToCart} class="flex-1">In den Warenkorb</Button>
		</div>
	</div>
{/if}
