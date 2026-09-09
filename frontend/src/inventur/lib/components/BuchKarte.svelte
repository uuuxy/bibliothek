<!-- @component BuchKarte — eine Kachel des Medienkatalogs, gebaut wie Google Play Books
     (02.09.2026, mit Peter entschieden): Das COVER ist die Kachel, der Text steht darunter
     auf der Fläche. Kein Kartenrahmen, kein Schatten, kein Trennstrich, kein Kasten um
     das Prüfdatum — beim Zeigen nur die M3-Zustandsschicht (m3-state) als weiche
     Tonfläche um Bild und Text.

     Fünf Informationen, in dieser Reihenfolge, und keine weiteren: Titel, ISBN mit
     Kopieren (die Nummer liest man gegen das Buch in der Hand ab), Signatur (die
     Regaladresse), Bestand als Satz statt Ampelpunkt, Prüfdatum nur wenn es eines gibt.
     Fach-, Klassen- und Zweig-Chips sind bewusst weg: Das Suchfeld findet sie, der
     Reiter „Jahrgänge" gruppiert danach. Der Autor interessiert im Schulkatalog nicht.

     Vorher (bis 02.09.2026): weiße Karte mit Rahmen UND Schatten, sechs
     Informationsschichten, Stift in einem weißen Kästchen auf dem Cover, Ampelpunkt
     mit Leuchtschein — die Buchakte in klein, 455 px hoch, vier je Reihe. -->
<script>
	import { coverKandidaten } from '../../../lib/utils/coverSrc.js';
	import BuchKarteCover from './BuchKarteCover.svelte';
	import { formatDatum } from '../../../lib/utils/format.js';
	import { Copy, MapPin, SquarePen } from '@lucide/svelte';

	/**
	 * @type {{
	 *   book: {
	 *     id: string,
	 *     isbn: string,
	 *     title: string,
	 *     author: string,
	 *     subject: string,
	 *     verfuegbar?: number,
	 *     gesamt?: number,
	 *     coverUrl: string,
	 *     lastCounted?: string,
	 *     signatur?: string,
	 *     medientyp?: string
	 *   },
	 *   onclick?: (event: Event) => void,
	 *   onEditClick?: () => void
	 * }}
	 */
	let { book, onclick, onEditClick } = $props();

	/** @type {string[]} */
	let coverCandidates = $state([]);
	let currentCandidateIndex = $state(0);
	let coverSrc = $derived(coverCandidates[currentCandidateIndex] || '');
	let coverFailed = $state(false);
	let copied = $state(false);

	const nummerArt = $derived(book.medientyp === 'CD' || book.medientyp === 'DVD' ? 'EAN' : 'ISBN');

	/** Bestand als Satz: „27 von 28 verfügbar", „Keine Exemplare" bei einem Titel ohne
	 *  Exemplare — der frühere rote Punkt sah für beides gleich aus. Bewusst OHNE Rot bei
	 *  0: Im Schuljahr ist fast jedes Lernmittel komplett verliehen; ein Katalog, der
	 *  überall rot ist, sagt nichts mehr. */
	const bestand = $derived(
		book.gesamt ? `${book.verfuegbar ?? 0} von ${book.gesamt} verfügbar` : 'Keine Exemplare'
	);

	const geprueft = $derived(book.lastCounted ? formatDatum(book.lastCounted) : '');

	/** @param {Event} e */
	function copyIsbn(e) {
		e.stopPropagation();
		if (!book.isbn) return;
		navigator.clipboard.writeText(book.isbn);
		copied = true;
		setTimeout(() => (copied = false), 2000);
	}

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

	// Lokale Liste statt Lesen des eigenen $state im selben Effekt: Der Effekt hinge sonst
	// von coverCandidates ab, das er gerade schreibt — Endlosschleife, Svelte bricht mit
	// effect_update_depth_exceeded ab und die ganze Seite reagiert nicht mehr (E2E 02.09.).
	$effect(() => {
		const kandidaten = coverKandidaten(book?.coverUrl, book?.isbn);
		coverCandidates = kandidaten;
		currentCandidateIndex = 0;
		coverFailed = kandidaten.length === 0;
	});
</script>

<!-- Die Kachel ist für Hilfstechnik KEIN Knopf mehr: Bis zum 09.09.2026 trug sie
     role="button" — mit Kopieren-Knopf und Stift darin, Bedienelemente im Bedienelement
     (nested-interactive, 50 Verstöße je Katalogseite). Jetzt ist der TITEL der Knopf
     (echter <button> in der Überschrift, per Tastatur und Screenreader erreichbar). Der
     Klick auf die ganze Fläche bleibt für Maus und Finger erhalten — als reiner
     Maus-Komfort ohne Rolle, deshalb die beiden Ausnahmen darunter; die Tastatur hat
     den Titel. Die inneren Knöpfe stoppen die Weitergabe wie zuvor. (Gemessen am
     09.09.2026: Der Stift öffnet trotzdem die Akte, weil dieser Flächen-Handler als
     direkter Listener vor Sveltes Delegation feuert — e2e/cover-aendern.spec.js
     schreibt genau das fest, also bleibt es so.) -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<!-- svelte-ignore a11y_click_events_have_key_events -->
<div class="m3-state group flex h-full cursor-pointer flex-col gap-3 rounded-2xl p-3" {onclick}>
	<div class="aspect-2/3 w-full overflow-hidden rounded-lg bg-surface-container-low">
		{#if coverSrc && !coverFailed}
			<img
				src={coverSrc}
				alt={`Cover von ${book.title}`}
				loading="lazy"
				class="h-full w-full object-cover"
				onerror={onCoverError}
				onload={onCoverLoad}
			/>
		{:else}
			<BuchKarteCover subject={book.subject} title={book.title} />
		{/if}
	</div>

	<div class="flex flex-col gap-1 text-sm text-on-surface-variant">
		<h2 class="text-base leading-snug font-semibold text-on-surface" title={book.title}>
			<!-- Ohne eigenen Handler: Enter/Leertaste erzeugen ein click-Ereignis, das zur
			     Fläche aufsteigt und dort die Akte öffnet — ein Handler hier navigierte doppelt.
			     Der Knopf trägt die Semantik (Tastatur, Screenreader), die Fläche das Verhalten. -->
			<button
				type="button"
				class="line-clamp-2 block w-full cursor-pointer text-left wrap-break-word hover:underline"
			>
				{book.title}
			</button>
		</h2>

		<button
			class="flex cursor-pointer items-center gap-1.5 text-left font-mono transition-colors hover:text-primary"
			onclick={copyIsbn}
			title="{nummerArt} kopieren"
			aria-label="{nummerArt} kopieren"
		>
			<span class="truncate" title={book.isbn}>{nummerArt}: {book.isbn || '–'}</span>
			{#if copied}
				<span class="font-sans text-xs font-medium text-primary">Kopiert!</span>
			{:else if book.isbn}
				<Copy class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
			{/if}
		</button>

		{#if book.signatur}
			<div class="flex items-center gap-1.5">
				<MapPin class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
				<span class="truncate font-mono">{book.signatur}</span>
			</div>
		{/if}

		<div class="flex items-center justify-between gap-2">
			<span>{bestand}</span>
			{#if onEditClick}
				<!-- 32×32 Trefferfläche, ruhig grau; sitzt im Text statt auf dem Cover. -->
				<button
					class="m3-state -my-1 flex h-8 w-8 shrink-0 cursor-pointer items-center justify-center rounded-full"
					onclick={(e) => {
						e.stopPropagation();
						onEditClick();
					}}
					title="Schnell bearbeiten"
					aria-label="Buch schnell bearbeiten"
				>
					<SquarePen class="h-4 w-4" aria-hidden="true" />
				</button>
			{/if}
		</div>

		{#if geprueft}
			<span>Geprüft {geprueft}</span>
		{/if}
	</div>
</div>
