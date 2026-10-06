<script>
	import Button from '../../../../lib/components/ui/Button.svelte';
	import BuchCover from '../../../../lib/components/ui/BuchCover.svelte';
	import { Save, Table, X } from '@lucide/svelte';
	let {
		selectedClasses = [],
		selectedBookIds = new Set(),
		selectedBooksList = [],
		isSaving = false,
		isUpdate = false,
		onToggleBook = () => {},
		onsave = () => {},
		// Abbrechen steht in derselben Aktionszeile wie Speichern (M3-Dialog: rechts
		// ausgerichtet, Textknopf vor gefülltem Knopf).
		oncancel = () => {}
	} = $props();
</script>

<div
	class="px-4 sm:px-6 py-4 sm:py-6 border-b border-outline-variant flex items-center justify-between"
>
	<h3 class="text-xl font-bold text-on-surface">Auswahl</h3>
	<div class="bg-surface-container-high px-3 py-1.5 rounded-full text-sm font-bold text-on-surface">
		{selectedBookIds.size}
	</div>
</div>

<div
	class="flex-1 overflow-y-auto [&::-webkit-scrollbar]:w-1.5 [&::-webkit-scrollbar-track]:bg-transparent [&::-webkit-scrollbar-thumb]:bg-outline-variant [&::-webkit-scrollbar-thumb]:rounded-full p-4 space-y-2"
>
	{#if selectedBooksList.length === 0}
		<div
			class="h-full flex flex-col items-center justify-center text-center p-8 text-on-surface-variant"
		>
			<Table class="mb-4" aria-hidden="true" />
			<p class="text-sm font-medium">Deine Auswahl ist noch leer</p>
		</div>
	{:else}
		{#each selectedBooksList as book (book.id)}
			<div
				class="flex items-center gap-3.5 hover:bg-surface-container p-2 rounded-xl transition-colors group"
			>
				<!-- Der Titel steht daneben, das Bild ist Schmuck. -->
				<BuchCover
					coverUrl={book.coverUrl}
					isbn={book.isbn}
					titel={book.title}
					groesse="liste"
					dekorativ
				/>
				<p class="font-medium text-on-surface grow truncate leading-tight">
					{book.title}
				</p>
				<button
					onclick={() => onToggleBook(book.id)}
					class="icon-btn text-on-surface-variant"
					data-tip="Buch entfernen"
					aria-label="Buch entfernen"
				>
					<X class="w-4 h-4" aria-hidden="true" />
				</button>
			</div>
		{/each}
	{/if}
</div>

<!-- Aktionszeile nach M3: rechtsbündig, Textknopf (verwerfen) vor gefülltem Knopf
     (bestätigen), Beschriftung in Satzschreibung. -->
<footer
	class="flex items-center justify-end gap-2 border-t border-outline-variant bg-surface-container-lowest p-4 sm:p-6"
>
	<Button variant="ghost" onclick={() => oncancel()}>Abbrechen</Button>
	<Button
		disabled={selectedClasses.length === 0 || (!isUpdate && selectedBookIds.size === 0) || isSaving}
		onclick={(e) => onsave(e)}
	>
		<Save class="h-4 w-4" aria-hidden="true" />
		<!-- Kurz halten: In der 340 px schmalen Spalte bräche „Auswahl speichern" auf zwei
		     Zeilen um. Was gespeichert wird, sagt die Überschrift des Dialogs. -->
		<span class="whitespace-nowrap">{isSaving ? 'Speichert …' : 'Speichern'}</span>
	</Button>
</footer>
