<!-- @component BookAkteMeta — der Kopf der Buchakte wie eine Play-Store-Detailseite
     (mit Peter am 02.09.2026 entschieden): Titel groß links, darunter in ruhigem Grau die
     Einordnung (Fach · Jahrgang · Zweig · Medienart), dann ISBN, Signatur und Standort als
     Textzeilen, eine Zahlenreihe (verfügbar, Ausleiher, Exemplare) und die Aktionen als
     Knopfreihe. Das Cover steht rechts mit Luft.

     Vorher: Cover links als farbiger Block, daneben sieben bunte Chips mit Rahmen, vier
     getönte Zahlenkästen in vier Farben, Titel erst darunter. -->
<script>
	import { Copy, MapPin, SquarePen, Trash2 } from '@lucide/svelte';
	import Button from './components/ui/Button.svelte';
	import BuchKarteCover from '../inventur/lib/components/BuchKarteCover.svelte';

	/**
	 * @typedef {Object} Props
	 * @property {any} book
	 * @property {any[]} borrowers
	 * @property {any[]} exemplare
	 * @property {string} coverSrc
	 * @property {boolean} coverFailed
	 * @property {(e: Event) => void} onCoverError
	 * @property {(e: Event) => void} onCoverLoad
	 * @property {(() => void) | undefined} [onEdit] nur mit edit_books
	 * @property {(() => void) | undefined} [onDelete] nur mit delete_books
	 */

	/** @type {Props} */
	let {
		book,
		borrowers,
		exemplare,
		coverSrc,
		coverFailed,
		onCoverError,
		onCoverLoad,
		onEdit = undefined,
		onDelete = undefined
	} = $props();

	let copied = $state(false);

	/** Einordnung in einer Zeile, wie „Sept. 2026 · Buch 3 · Verlag" im Play Store. */
	const einordnung = $derived(
		[
			book.subject,
			book.jahrgangVon && book.jahrgangBis
				? book.jahrgangVon === book.jahrgangBis
					? `Jahrgang ${book.jahrgangVon}`
					: `Jahrgang ${book.jahrgangVon}–${book.jahrgangBis}`
				: book.gradeLevel
					? `Jahrgang ${book.gradeLevel}`
					: '',
			book.track,
			book.medientyp && book.medientyp !== 'Buch' ? book.medientyp : ''
		]
			.filter(Boolean)
			.join(' · ')
	);

	const nummerArt = $derived(book.medientyp === 'CD' || book.medientyp === 'DVD' ? 'EAN' : 'ISBN');
	const signatur = $derived(book.signatur || book.erweiterte_eigenschaften?.signatur || '');
	const standort = $derived(book.erweiterte_eigenschaften?.standort || '');

	function kopieren() {
		if (!book.isbn) return;
		navigator.clipboard.writeText(book.isbn);
		copied = true;
		setTimeout(() => (copied = false), 2000);
	}
</script>

<div class="flex flex-col-reverse gap-8 sm:flex-row sm:items-start">
	<div class="min-w-0 flex-1">
		<h1 class="text-3xl leading-tight font-semibold wrap-break-word text-on-surface">
			{book.title}
		</h1>
		{#if book.untertitel}
			<p class="mt-1 text-lg text-on-surface-variant">{book.untertitel}</p>
		{/if}
		{#if einordnung}
			<p class="mt-2 text-sm text-on-surface-variant">{einordnung}</p>
		{/if}

		<div class="mt-4 flex flex-col gap-1 text-sm text-on-surface-variant">
			<button
				class="flex w-fit cursor-pointer items-center gap-1.5 font-mono transition-colors hover:text-primary"
				onclick={kopieren}
				title="{nummerArt} kopieren"
				aria-label="{nummerArt} kopieren"
			>
				<span>{nummerArt}: {book.isbn || '–'}</span>
				{#if copied}
					<span class="font-sans text-xs font-medium text-primary">Kopiert!</span>
				{:else if book.isbn}
					<Copy class="h-3.5 w-3.5" aria-hidden="true" />
				{/if}
			</button>
			{#if signatur}
				<span class="flex items-center gap-1.5">
					<MapPin class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
					<span class="font-mono">{signatur}</span>
					{#if standort}<span>· {standort}</span>{/if}
				</span>
			{:else if standort}
				<span class="flex items-center gap-1.5">
					<MapPin class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
					<span>{standort}</span>
				</span>
			{/if}
		</div>

		<!-- Zahlenreihe wie „E-Book | 544 Seiten": Wert oben, Bedeutung darunter, Trennlinien. -->
		<dl class="mt-6 flex divide-x divide-outline-variant">
			<div class="pr-6">
				<dd class="text-lg font-semibold tabular-nums text-on-surface">
					{book.verfuegbar ?? 0} von {book.gesamt ?? 0}
				</dd>
				<dt class="text-xs text-on-surface-variant">verfügbar</dt>
			</div>
			<div class="px-6">
				<dd class="text-lg font-semibold tabular-nums text-on-surface">{borrowers.length}</dd>
				<dt class="text-xs text-on-surface-variant">Ausleiher</dt>
			</div>
			<div class="pl-6">
				<dd class="text-lg font-semibold tabular-nums text-on-surface">{exemplare.length}</dd>
				<dt class="text-xs text-on-surface-variant">Exemplare</dt>
			</div>
		</dl>

		{#if onEdit || onDelete}
			<div class="mt-6 flex flex-wrap items-center gap-3">
				{#if onEdit}
					<Button onclick={onEdit}>
						<SquarePen class="h-4 w-4" aria-hidden="true" />
						Titel bearbeiten
					</Button>
				{/if}
				{#if onDelete}
					<Button variant="ghost" onclick={onDelete} class="text-error">
						<Trash2 class="h-4 w-4" aria-hidden="true" />
						Titel löschen
					</Button>
				{/if}
			</div>
		{/if}
	</div>

	<div class="w-40 shrink-0 sm:w-48">
		<div class="aspect-2/3 w-full overflow-hidden rounded-lg bg-surface-container-low">
			{#if coverSrc && !coverFailed}
				<img
					src={coverSrc}
					alt={`Cover ${book.title}`}
					class="h-full w-full object-cover"
					onerror={onCoverError}
					onload={onCoverLoad}
				/>
			{:else}
				<BuchKarteCover subject={book.subject} title={book.title} />
			{/if}
		</div>
	</div>
</div>
