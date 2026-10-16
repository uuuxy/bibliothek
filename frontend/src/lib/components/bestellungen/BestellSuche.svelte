<script>
	import Suchfeld from '../ui/Suchfeld.svelte';
	import { scanUebernehmen } from './scanTreffer.js';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import { apiPost } from '../../apiFetch.js';
	import { snackbars } from '../../stores/snackbars.svelte.js';
	import { bestellStore } from '../../stores/bestellStore.svelte.js';
	import Select from '../ui/Select.svelte';
	import TrefferPruefung from './TrefferPruefung.svelte';
	import BuchCover from '../ui/BuchCover.svelte';
	import { fensterAusAntwort } from './ausIsbn.js';

	/** @type {any} */
	let stagedBook = $state(null);
	let resolvingDnb = $state(false);

	let localResults = $derived(bestellStore.searchResults.filter((r) => r.source === 'local'));
	let dnbResults = $derived(bestellStore.searchResults.filter((r) => r.source === 'dnb'));

	/** @param {any} book */
	async function waehleTitel(book) {
		if (book.source !== 'dnb') return stelleTitelBereit(book);
		resolvingDnb = true;
		try {
			// Die Tür nimmt den Titel aus dem Katalog oder legt ihn aus der DNB an (ausIsbn.js).
			const localBook = await apiPost('/api/buecher/aus-isbn', { isbn: book.isbn });
			if (localBook && localBook.titel_id) {
				stelleTitelBereit(fensterAusAntwort(localBook, book));
			} else {
				snackbars.addToast('Fehler beim Anlegen des DNB-Buchs', 'error');
			}
		} catch {
			snackbars.addToast('Fehler beim Anlegen des DNB-Buchs', 'error');
		} finally {
			resolvingDnb = false;
		}
	}

	/** @param {any} book */
	function stelleTitelBereit(book) {
		stagedBook = book;
		bestellStore.resetSearch();
	}
</script>

<div class="space-y-4">
	<div class="space-y-1.5">
		<label for="supplier" class="block text-xs font-medium text-on-surface-variant">Lieferant</label
		>
		<Select
			id="supplier"
			bind:value={bestellStore.selectedSupplierId}
			options={bestellStore.suppliers.map((s) => ({
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
			onscan={(code) => scanUebernehmen(code, bestellStore, waehleTitel)}
			autofokus
			id="book"
			bind:wert={bestellStore.searchQuery}
			oninput={() => bestellStore.handleSearchInput()}
			platzhalter="Titel, Autor oder ISBN …"
			etikett="Titel suchen & hinzufügen"
		/>
		{#if bestellStore.showDropdown && (bestellStore.searchResults.length > 0 || bestellStore.dnbAusfall)}
			<!-- M3 Lists: Titel in on-surface, Autor und ISBN sowie die Angabe rechts („trailing
			     text") in on-surface-variant, keine Trennlinien zwischen den Zeilen. Das Cover
			     kommt aus ui/BuchCover — mit dessen Platzhalter, wenn keine Quelle ein Bild hat.
			     Die Rückmeldung beim Zeigen gibt der State-Layer der Knöpfe. -->
			<div
				class="absolute z-10 w-full mt-1 bg-surface-container rounded-sm shadow-xl max-h-72 overflow-y-auto"
			>
				{#if bestellStore.dnbAusfall}
					<!-- Ohne die Zeile sähe ein Buch, das die DNB kennt, aus wie eines, das sie nicht
					     kennt. Sie steht oben: Unter vielen Treffern aus dem Bestand läse sie niemand.
					     Der Wortlaut ist der des Schlagwort-Vorschlags aus der DNB. -->
					<p class="px-3.5 py-2.5 text-sm text-error" role="alert">
						Die DNB ist nicht erreichbar. Bitte später erneut versuchen.
					</p>
				{/if}
				{#if localResults.length > 0}
					<div
						class="bg-surface-container px-3.5 py-2 text-xs font-medium text-on-surface-variant sticky top-0 z-5"
					>
						Im lokalen Bestand
					</div>
					{#each localResults as b, _i (_i)}
						<button
							onclick={() => waehleTitel(b)}
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
							onclick={() => !isDuplicate && waehleTitel(b)}
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
		{#if bestellStore.searchLoading}
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
	</div>
</div>

{#if stagedBook}
	<!-- {#key}: Ein neuer Treffer bekommt ein frisches Fenster mit frischen Feldern,
	     statt die Eingaben des vorigen zu erben. -->
	{#key stagedBook.id}
		<TrefferPruefung book={stagedBook} onDone={() => (stagedBook = null)} />
	{/key}
{/if}
