<script>
	import Suchfeld from '../ui/Suchfeld.svelte';
	import { scanUebernehmen } from './scanTreffer.js';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import { apiPost } from '../../apiFetch.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import { orderStore } from '../../stores/orderStore.svelte.js';
	import Select from '../ui/Select.svelte';
	import OrderStaging from './OrderStaging.svelte';
	import BuchCover from '../ui/BuchCover.svelte';
	import AndereIsbnFormWahl from './AndereIsbnFormWahl.svelte';
	import { ausIsbnRumpf, istAndereFormFrage } from './ausIsbn.js';

	/** @type {any} */
	let stagedBook = $state(null);
	let resolvingDnb = $state(false);
	/** Die Frage, wenn die ISBN des DNB-Treffers in der anderen Länge im Katalog steht; bis dahin
	 *  ist nichts angelegt (ausIsbn.js). @type {{ vorschlag: any, book: any } | null} */
	let wahl = $state(null);

	let localResults = $derived(orderStore.searchResults.filter((r) => r.source === 'local'));
	let dnbResults = $derived(orderStore.searchResults.filter((r) => r.source === 'dnb'));

	/** @param {any} book @param {boolean} [neuAnlegen] true nach „Neu anlegen" */
	async function openStaging(book, neuAnlegen = false) {
		wahl = null;
		if (book.source !== 'dnb') return stageBook(book);
		resolvingDnb = true;
		try {
			const localBook = await apiPost('/api/buecher/aus-isbn', ausIsbnRumpf(book.isbn, neuAnlegen));
			if (istAndereFormFrage(localBook)) {
				wahl = { vorschlag: localBook.andere_form, book };
				orderStore.showDropdown = false;
			} else if (localBook && localBook.titel_id) {
				stageAusTuer(localBook, book);
			} else {
				toastStore.addToast('Fehler beim Anlegen des DNB-Buchs', 'error');
			}
		} catch {
			toastStore.addToast('Fehler beim Anlegen des DNB-Buchs', 'error');
		} finally {
			resolvingDnb = false;
		}
	}

	/** Das Fenster mit dem Titel aus der Tür. @param {any} localBook @param {any} book der DNB-Treffer */
	function stageAusTuer(localBook, book) {
		stageBook({
			id: localBook.titel_id,
			titel: localBook.titel,
			autor: localBook.autor,
			isbn: localBook.isbn,
			verlag: localBook.verlag,
			cover_url: localBook.cover_url,
			// exists=false: signatur ist nur ein VORSCHLAG aus der DNB-Heuristik.
			signatur: localBook.signatur ?? '',
			// Preisvorschlag vom DNB-Treffer — über /aus-isbn ginge er sonst verloren.
			preis_vorschlag: book.preis_vorschlag,
			// Ein eben angelegter Titel ist noch kein Lernmittel — OrderStaging fragt nach.
			ist_lernmittel: Boolean(localBook.ist_lernmittel),
			// Schlagworte der eigenen Liste, die der DNB-Satz nennt — nur angeboten,
			// eingetragen wird erst im Fenster (docs/OFFEN.md 4.20).
			schlagwort_vorschlaege: localBook.schlagwort_vorschlaege ?? []
		});
	}

	/** @param {any} book */
	function stageBook(book) {
		wahl = null;
		stagedBook = book;
		orderStore.resetSearch();
	}
</script>

<div class="space-y-4">
	<div class="space-y-1.5">
		<label for="supplier" class="block text-xs font-medium text-on-surface-variant">Lieferant</label
		>
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
		<label for="book" class="block text-xs font-medium text-on-surface-variant"
			>Titel suchen &amp; hinzufügen</label
		>
		<Suchfeld
			kamera
			onscan={(code) => ((wahl = null), scanUebernehmen(code, orderStore, openStaging))}
			autofokus
			id="book"
			bind:wert={orderStore.searchQuery}
			oninput={() => ((wahl = null), orderStore.handleSearchInput())}
			platzhalter="Titel, Autor oder ISBN …"
			etikett="Titel suchen & hinzufügen"
		/>
		{#if orderStore.showDropdown && (localResults.length > 0 || dnbResults.length > 0)}
			<!-- M3 Lists: Titel in on-surface, Autor und ISBN sowie die Angabe rechts („trailing
			     text") in on-surface-variant, keine Trennlinien zwischen den Zeilen. Das Cover
			     kommt aus ui/BuchCover — mit dessen Platzhalter, wenn keine Quelle ein Bild hat.
			     Die Rückmeldung beim Zeigen gibt der State-Layer der Knöpfe. -->
			<div
				class="absolute z-10 w-full mt-1 bg-surface-container rounded-sm shadow-xl max-h-72 overflow-y-auto"
			>
				{#if localResults.length > 0}
					<div
						class="bg-surface-container px-3.5 py-2 text-xs font-medium text-on-surface-variant sticky top-0 z-5"
					>
						Im lokalen Bestand
					</div>
					{#each localResults as b, _i (_i)}
						<button
							onclick={() => openStaging(b)}
							class="w-full text-left px-3.5 py-2.5 flex items-center gap-3 text-base"
						>
							<BuchCover coverUrl={b.cover_url} isbn={b.isbn} titel={b.titel} dekorativ />
							<div class="min-w-0 flex-1">
								<div class="font-bold text-on-surface truncate">{b.titel}</div>
								<div class="text-sm text-on-surface-variant truncate">{b.autor} · {b.isbn}</div>
							</div>
							<span class="shrink-0 text-label-small text-on-surface-variant">
								Bestand: {b.current_stock || 0}
							</span>
						</button>
					{/each}
				{/if}

				{#if dnbResults.length > 0}
					<div
						class="bg-surface-container px-3.5 py-2 text-xs font-medium text-on-surface-variant sticky top-0 z-5"
					>
						Neu aus DNB (Externe Suche)
					</div>
					{#each dnbResults as b, _i (_i)}
						{@const isDuplicate =
							b.is_duplicate ||
							localResults.some(
								(l) => (l.isbn || '').replace(/-/g, '') === (b.isbn || '').replace(/-/g, '')
							)}
						<button
							onclick={() => !isDuplicate && openStaging(b)}
							disabled={isDuplicate}
							class="w-full text-left px-3.5 py-2.5 flex items-center gap-3 text-base {isDuplicate
								? 'opacity-50 cursor-not-allowed'
								: ''}"
						>
							<BuchCover coverUrl={b.cover_url} isbn={b.isbn} titel={b.titel} dekorativ />
							<div class="min-w-0 flex-1">
								<div class="font-bold text-on-surface truncate">{b.titel}</div>
								<div class="text-sm text-on-surface-variant truncate">{b.autor} · {b.isbn}</div>
							</div>
							<span class="shrink-0 text-label-small text-on-surface-variant">
								{isDuplicate ? 'Vorhanden' : 'Neu'}
							</span>
						</button>
					{/each}
				{/if}
			</div>
		{/if}
		{#if orderStore.searchLoading}
			<div
				class="absolute z-10 w-full mt-1 bg-surface-container rounded-sm shadow-xl px-4 py-3 flex items-center gap-2 text-sm text-on-surface-variant"
			>
				<Ladekreis size="sm" />
				Suche läuft...
			</div>
		{:else if resolvingDnb}
			<div
				class="absolute z-10 w-full mt-1 bg-surface-container rounded-sm shadow-xl px-4 py-3 flex items-center gap-2 text-sm text-on-surface-variant"
			>
				<Ladekreis size="sm" />
				Titel wird im Katalog angelegt...
			</div>
		{/if}
		{#if wahl}
			<AndereIsbnFormWahl
				isbn={wahl.book.isbn}
				vorschlag={wahl.vorschlag}
				neuTitel={wahl.book.titel}
				onnehmen={() => wahl && stageAusTuer(wahl.vorschlag, wahl.book)}
				onneu={() => wahl && openStaging(wahl.book, true)}
			/>
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
