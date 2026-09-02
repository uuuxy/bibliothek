<!-- @component KlassenBuchKachel — ein Titel im Klassensatz, gebaut wie die Katalog-Kachel
     (BuchKarte, 02.09.2026): Cover ohne Rahmen, darunter Titel und Bestand. Der Bestand
     („82 von 82 verfügbar") stand bis dahin nirgends auf der Kachel, obwohl der
     Klassensatz-Endpunkt ihn je Titel liefert — Peter wollte ihn beim Zeigen sehen; er
     steht jetzt immer da, denn am Tablet gibt es kein Zeigen.

     Vorher: weiße Karte mit Rahmen, Schatten und Hochheben, blaues Overlay „Bearbeiten"
     erst bei :hover, Zweig-Abzeichen auf dem Cover, Autor auf der Attrappe. -->
<script>
	import { coverKandidaten } from '../../../../lib/utils/coverSrc.js';
	import BuchKarteCover from '../BuchKarteCover.svelte';

	/**
	 * @type {{
	 *   book: {
	 *     id: string,
	 *     isbn: string,
	 *     title: string,
	 *     subject: string,
	 *     coverUrl: string,
	 *     verfuegbar?: number,
	 *     gesamt?: number
	 *   },
	 *   onEdit?: (book: any) => void,
	 *   bearbeitbar?: boolean
	 * }}
	 * bearbeitbar=false: reine Ansicht (Kollegiums-Portal, 25.08.2026). Ohne den Schalter
	 * wäre jede Kachel dort ein Knopf, hinter dem für die Rolle ein 403 liegt.
	 */
	let { book, onEdit = undefined, bearbeitbar = true } = $props();

	/** @type {string[]} */
	let coverCandidates = $state([]);
	let currentCandidateIndex = $state(0);
	let coverSrc = $derived(coverCandidates[currentCandidateIndex] || '');
	let coverFailed = $state(false);

	/** Bestand als Satz wie auf der Katalog-Kachel; ohne Zahlen im Payload bleibt die Zeile leer. */
	const bestand = $derived(
		book.gesamt == null
			? ''
			: book.gesamt
				? `${book.verfuegbar ?? 0} von ${book.gesamt} verfügbar`
				: 'Keine Exemplare'
	);

	// Lokale Liste statt Lesen des eigenen $state im selben Effekt (sonst Endlosschleife,
	// siehe BuchKarte.svelte).
	$effect(() => {
		const kandidaten = coverKandidaten(book?.coverUrl, book?.isbn);
		coverCandidates = kandidaten;
		currentCandidateIndex = 0;
		coverFailed = kandidaten.length === 0;
	});

	function onCoverError() {
		if (currentCandidateIndex < coverCandidates.length - 1) {
			currentCandidateIndex++;
		} else {
			coverFailed = true;
		}
	}

	/** @param {Event} event */
	function onCoverLoad(event) {
		const image = /** @type {HTMLImageElement} */ (event.currentTarget);
		if (image.naturalWidth < 10 || image.naturalHeight < 10) onCoverError();
	}
</script>

<!-- Zwei Hüllen, ein Inhalt: Die Rolle „button" muss für svelte-check STATISCH am Element
     stehen (a11y_no_noninteractive_tabindex). Der Inhalt liegt einmal im Snippet. -->
{#snippet inhalt()}
	<div class="aspect-2/3 w-full overflow-hidden rounded-lg bg-surface-container-low">
		{#if coverSrc && !coverFailed}
			<img
				src={coverSrc}
				alt=""
				loading="lazy"
				class="h-full w-full object-cover"
				onerror={onCoverError}
				onload={onCoverLoad}
			/>
		{:else}
			<BuchKarteCover subject={book.subject} title={book.title} />
		{/if}
	</div>
	<div class="flex flex-col gap-0.5 text-sm">
		<h3
			class="line-clamp-2 leading-snug font-semibold wrap-break-word text-on-surface"
			title={book.title}
		>
			{book.title}
		</h3>
		{#if bestand}
			<span class="text-on-surface-variant">{bestand}</span>
		{/if}
	</div>
{/snippet}

{#if bearbeitbar}
	<div
		class="m3-state flex cursor-pointer flex-col gap-2 rounded-2xl p-2"
		onclick={() => onEdit?.(book)}
		role="button"
		tabindex="0"
		aria-label="{book.title} bearbeiten"
		onkeydown={(e) => {
			if (e.target !== e.currentTarget) return;
			if (e.key === 'Enter' || e.key === ' ') {
				e.preventDefault();
				onEdit?.(book);
			}
		}}
	>
		{@render inhalt()}
	</div>
{:else}
	<div class="flex flex-col gap-2 rounded-2xl p-2">
		{@render inhalt()}
	</div>
{/if}
