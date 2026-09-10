<script>
	import Suchfeld from '../ui/Suchfeld.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import { apiPost } from '../../apiFetch.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import { orderStore } from '../../stores/orderStore.svelte.js';
	import Select from '../ui/Select.svelte';
	import OrderStaging from './OrderStaging.svelte';
	import { coverSrc } from '../../utils/coverSrc.js';

	/** @type {any} */
	let stagedBook = $state(null);
	let resolvingDnb = $state(false);

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
						preis_vorschlag: book.preis_vorschlag,
						// Ein eben angelegter Titel ist noch kein Lernmittel — das Staging-
						// Fenster fragt nach (OrderStaging).
						ist_lernmittel: Boolean(localBook.ist_lernmittel)
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
		orderStore.resetSearch();
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
		<Suchfeld
			id="book"
			bind:wert={orderStore.searchQuery}
			oninput={() => orderStore.handleSearchInput()}
			platzhalter="Titel, Autor oder ISBN …"
			etikett="Titel suchen & hinzufügen"
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
				<Ladekreis size="sm" />
				Suche läuft...
			</div>
		{:else if resolvingDnb}
			<div
				class="absolute z-10 w-full mt-1 bg-surface-container rounded-sm shadow-xl px-4 py-3 flex items-center gap-2 text-sm text-slate-500"
			>
				<Ladekreis size="sm" />
				Titel wird im Katalog angelegt...
			</div>
		{/if}
	</div>
</div>

{#if stagedBook}
	<!-- {#key}: Ein neuer Treffer bekommt ein frisches Fenster mit frischen Feldern,
	     statt die Eingaben des vorigen zu erben. -->
	{#key stagedBook.id}
		<OrderStaging book={stagedBook} onDone={() => (stagedBook = null)} />
	{/key}
{/if}
