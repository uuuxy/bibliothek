<script>
	import { erzeugeAuswahl } from './bookTableAuswahl.svelte.js';
	import Tabelle from '../../../../lib/components/ui/Tabelle.svelte';
	import BookTableToolbar from '$lib/components/admin/BookTableToolbar.svelte';
	import BookTableZeile from '$lib/components/admin/BookTableZeile.svelte';
	import Button from '../../../../lib/components/ui/Button.svelte';
	import Kaestchen from '../../../../lib/components/ui/Kaestchen.svelte';

	/**
	 * @type {{
	 *   books: any[],
	 *   loading: boolean,
	 *   onOpenDetail: (book: any) => void,
	 *   onCreateNew: () => void,
	 *   onScan: () => void,
	 *   onDelete: (ids: string[]) => void,
	 *   onAssignClass: (ids: string[]) => void,
	 *   onRetryCovers: () => void
	 * }}
	 */
	let {
		books,
		loading,
		onOpenDetail,
		onCreateNew,
		onScan,
		onDelete,
		onAssignClass,
		onRetryCovers
	} = $props();

	// Auswahl für die Massenaktionen: bookTableAuswahl.svelte.js. Der Effekt hält sie auf
	// dem, was in der Liste steht — sonst trifft „Löschen" nach dem Filtern Unsichtbares.
	const auswahl = erzeugeAuswahl();
	$effect(() => auswahl.angleichen(books));

	// Performance: Begrenzung der gerenderten DOM-Elemente
	let maxVisible = $state(50);

	function handleDelete() {
		onDelete(auswahl.ids);
		auswahl.leeren();
	}

	function handleAssignClass() {
		// Auswahl bleibt bestehen, bis der Dialog abgeschlossen/abgebrochen ist —
		// der Picker hält die IDs bereits über seine Prop.
		onAssignClass(auswahl.ids);
	}
</script>

<div class="w-full">
	<BookTableToolbar
		booksLength={books.length}
		selectedCount={auswahl.anzahl}
		onDelete={handleDelete}
		onAssignClass={handleAssignClass}
		{onScan}
		{onCreateNew}
		{onRetryCovers}
	/>

	<div class="overflow-x-auto">
		<Tabelle beschriftung="Titel im Bestand">
			<thead class="font-medium">
				<tr>
					<th class="w-10">
						<Kaestchen
							aria-label="Alle Bücher auswählen"
							checked={auswahl.alleGewaehlt(books)}
							onclick={() => auswahl.alleUmschalten(books)}
						/>
					</th>
					<th class="w-20">Cover</th>
					<th>Titel</th>
					<th>Fach</th>
					<th>Klasse</th>
					<th>Art</th>
					<th>Standort</th>
					<th class="text-right">Zuletzt geprüft</th>
					<th class="text-right">Bestand</th>
					<th class="w-10"></th>
				</tr>
			</thead>

			<tbody>
				<!-- Die Reihenfolge kommt vom Server: nach dem Titel (GET /api/books). -->
				{#each books.slice(0, maxVisible) as book (book.id)}
					<BookTableZeile
						{book}
						isSelected={auswahl.enthaelt(book.id)}
						{onOpenDetail}
						onToggleSelect={(id) => auswahl.umschalten(id)}
					/>
				{/each}

				{#if books.length === 0 && !loading}
					<tr>
						<td colspan="10" class="text-center">Keine Bücher gefunden.</td>
					</tr>
				{/if}
			</tbody>
		</Tabelle>

		{#if books.length > maxVisible}
			<div class="flex justify-center p-4">
				<Button variant="secondary" onclick={() => (maxVisible += 50)}>
					Weitere Bücher laden ({books.length - maxVisible} verbleibend)
				</Button>
			</div>
		{/if}
	</div>
</div>
