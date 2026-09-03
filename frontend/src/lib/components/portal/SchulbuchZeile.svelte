<script>
	/**
	 * @component SchulbuchZeile — eine Zeile der Schulbuch-Liste im Portal, nach der M3-
	 * Liste gebaut (Leading = Cover 2:3, Headline = Titel, Supporting = Autor · ISBN,
	 * Trailing = Zahlen). Cover-Kandidaten und Rückfall wie in KlassenBuchKachel; die
	 * Zeichnung (BuchKarteCover) springt ein, wenn kein Bild lädt.
	 * @property {any} book  Titel aus /api/portal/lernmittel (title, autor, subject, isbn, coverUrl, gesamt, verliehen)
	 * @property {boolean} [mitFach]  Fach als Beitext zeigen (Ansicht „Alle")
	 */
	import BuchKarteCover from '../../../inventur/lib/components/BuchKarteCover.svelte';
	import { coverKandidaten } from '../../utils/coverSrc.js';

	let { book, mitFach = false } = $props();

	/** @type {string[]} */
	let kandidaten = $state([]);
	let index = $state(0);
	let fehlgeschlagen = $state(false);
	const src = $derived(kandidaten[index] || '');

	// Lokale Liste statt Lesen des eigenen $state im selben Effekt (Endlosschleife, siehe
	// BuchKarte.svelte).
	$effect(() => {
		const k = coverKandidaten(book?.coverUrl, book?.isbn);
		kandidaten = k;
		index = 0;
		fehlgeschlagen = k.length === 0;
	});

	function naechster() {
		if (index < kandidaten.length - 1) index++;
		else fehlgeschlagen = true;
	}
	/** @param {Event} e */
	function geladen(e) {
		const img = /** @type {HTMLImageElement} */ (e.currentTarget);
		if (img.naturalWidth < 10 || img.naturalHeight < 10) naechster();
	}

	const beitext = $derived(
		[mitFach ? book.subject || 'Ohne Fach' : '', book.autor, book.isbn].filter(Boolean).join(' · ')
	);
</script>

<li class="flex items-center gap-4 border-b border-outline-variant/40 py-2">
	<div class="aspect-2/3 w-10 shrink-0 overflow-hidden rounded-sm bg-surface-container-low">
		{#if src && !fehlgeschlagen}
			<img
				{src}
				alt=""
				loading="lazy"
				class="h-full w-full object-cover"
				onerror={naechster}
				onload={geladen}
			/>
		{:else}
			<BuchKarteCover subject={book.subject} title={book.title} />
		{/if}
	</div>
	<div class="min-w-0 flex-1">
		<h3 class="truncate text-body-large font-medium text-on-surface">{book.title}</h3>
		{#if beitext}
			<p class="truncate text-body-medium text-on-surface-variant">{beitext}</p>
		{/if}
	</div>
	<div class="flex shrink-0 flex-col items-end tabular-nums">
		<span class="text-label-large font-medium text-on-surface">{book.gesamt} Exemplare</span>
		<span class="text-body-medium text-on-surface-variant">
			{book.verliehen ? `${book.verliehen} verliehen` : 'alle im Haus'}
		</span>
	</div>
</li>
