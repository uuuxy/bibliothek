<!-- @component BuchKarte — eine Kachel des Medienkatalogs, gebaut wie Google Play Books: Das
     Cover ist die Kachel, der Text steht darunter auf der Fläche. Kein Kartenrahmen, kein
     Schatten, kein Trennstrich, kein Kasten um das Prüfdatum; beim Zeigen nur die
     M3-Zustandsschicht (m3-state) als weiche Tonfläche um Bild und Text.

     Fünf Informationen, in dieser Reihenfolge, und keine weiteren: Titel, ISBN mit
     Kopieren (die Nummer liest man gegen das Buch in der Hand ab), Signatur (die
     Regaladresse), Bestand als Satz statt Ampelpunkt, Prüfdatum nur wenn es eines gibt.
     Fach-, Klassen- und Zweig-Chips fehlen mit Absicht, das Suchfeld findet sie. Der Autor
     interessiert im Schulkatalog nicht. Ein Buch in mehreren Auflagen zählt die Summe; eine
     Zeile darunter sagt, woraus sie besteht (startseiten_api.js, buecherJeBuch). -->
<script>
	import { coverKandidaten } from '../../../lib/utils/coverSrc.js';
	import BuchKarteCover from './BuchKarteCover.svelte';
	import { bestandSatz, formatDatum } from '../../../lib/utils/format.js';
	import { auflagenAufschluesselung } from '../../../lib/utils/auflagenText.js';
	import { Copy, MapPin, SquarePen } from '@lucide/svelte';

	/**
	 * @type {{
	 *   titel: {
	 *     id: string,
	 *     isbn: string,
	 *     title: string,
	 *     author: string,
	 *     subject: string,
	 *     verfuegbar?: number,
	 *     gesamt?: number,
	 *     imZulauf?: number,
	 *     coverUrl: string,
	 *     lastCounted?: string,
	 *     signatur?: string,
	 *     medientyp?: string,
	 *     auflage?: string,
	 *     buch?: { gesamt: number, verfuegbar: number, imZulauf: number, auflagen: any[] }
	 *   },
	 *   onclick?: (event: Event) => void,
	 *   onEditClick?: () => void
	 * }}
	 */
	let { titel, onclick, onEditClick } = $props();

	/** @type {string[]} */
	let coverCandidates = $state([]);
	let currentCandidateIndex = $state(0);
	let coverSrc = $derived(coverCandidates[currentCandidateIndex] || '');
	let coverFailed = $state(false);
	let copied = $state(false);

	const nummerArt = $derived(
		titel.medientyp === 'CD' || titel.medientyp === 'DVD' ? 'EAN' : 'ISBN'
	);

	/** Bestand als Satz: „27 von 28 verfügbar", „Keine Exemplare" bei einem Titel ohne
	 *  Exemplare; ein Farbpunkt sähe für beides gleich aus. Kein Rot bei 0: Im Schuljahr ist
	 *  fast jedes Lernmittel komplett verliehen, und ein Katalog, der überall rot ist, sagt
	 *  nichts mehr. Der Wortlaut kommt aus format.js, weil ihn auch die Theke braucht. */
	const zahlen = $derived(titel.buch ?? titel);
	const bestand = $derived(bestandSatz(zahlen.gesamt, zahlen.verfuegbar, zahlen.imZulauf));

	const geprueft = $derived(titel.lastCounted ? formatDatum(titel.lastCounted) : '');

	/** @param {Event} e */
	function copyIsbn(e) {
		e.stopPropagation();
		if (!titel.isbn) return;
		navigator.clipboard.writeText(titel.isbn);
		copied = true;
		setTimeout(() => (copied = false), 2000);
	}

	function naechsteQuelle() {
		if (currentCandidateIndex < coverCandidates.length - 1) {
			currentCandidateIndex++;
		} else {
			coverFailed = true;
		}
	}

	/** @param {Event} event */
	function pruefeGeladenes(event) {
		const image = /** @type {HTMLImageElement} */ (event.currentTarget);
		if (image.naturalWidth < 10 || image.naturalHeight < 10) naechsteQuelle();
	}

	// Lokale Liste statt Lesen des eigenen $state im selben Effekt: Der Effekt hinge sonst
	// von coverCandidates ab, das er gerade schreibt — Endlosschleife, Svelte bricht mit
	// effect_update_depth_exceeded ab und die ganze Seite reagiert nicht mehr.
	$effect(() => {
		const kandidaten = coverKandidaten(titel?.coverUrl, titel?.isbn);
		coverCandidates = kandidaten;
		currentCandidateIndex = 0;
		coverFailed = kandidaten.length === 0;
	});
</script>

<!-- Die Kachel ist für Hilfstechnik kein Knopf: Mit Kopieren-Knopf und Stift darin wäre sie
     ein Bedienelement im Bedienelement (axe: nested-interactive). Der Knopf ist der Titel,
     ein echter <button> in der Überschrift. Der Klick auf die ganze Fläche bleibt als Komfort
     für Maus und Finger, ohne Rolle, deshalb die beiden Ausnahmen darunter. Die inneren
     Knöpfe stoppen die Weitergabe: Der Stift öffnet nur die Bearbeitung, nicht auch die Akte
     (BuchKarte.test.js). -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<!-- svelte-ignore a11y_click_events_have_key_events -->
<div class="m3-state group flex h-full cursor-pointer flex-col gap-3 rounded-2xl p-3" {onclick}>
	<div class="aspect-2/3 w-full overflow-hidden rounded-lg bg-surface-container-low">
		{#if coverSrc && !coverFailed}
			<img
				src={coverSrc}
				alt={`Cover von ${titel.title}`}
				loading="lazy"
				class="h-full w-full object-cover"
				onerror={naechsteQuelle}
				onload={pruefeGeladenes}
			/>
		{:else}
			<BuchKarteCover subject={titel.subject} title={titel.title} />
		{/if}
	</div>

	<div class="flex flex-col gap-1 text-sm text-on-surface-variant">
		<h2 class="text-base leading-snug font-semibold text-on-surface" title={titel.title}>
			<!-- Ohne eigenen Handler: Enter/Leertaste erzeugen ein click-Ereignis, das zur
			     Fläche aufsteigt und dort die Akte öffnet — ein Handler hier navigierte doppelt.
			     Der Knopf trägt die Semantik (Tastatur, Screenreader), die Fläche das Verhalten. -->
			<button
				type="button"
				class="line-clamp-2 block w-full cursor-pointer text-left wrap-break-word hover:underline"
			>
				{titel.title}
			</button>
		</h2>

		<!-- Die Auflage steht direkt unter dem Titel: Zwei Zeilen „Lambacher Schweizer 7"
		     im Katalog sind zwei verschiedene Bücher, und das sieht man sonst nirgends. -->
		{#if titel.auflage}
			<p class="truncate text-xs" title={titel.auflage}>{titel.auflage}</p>
		{/if}

		<button
			class="flex cursor-pointer items-center gap-1.5 text-left font-mono transition-colors hover:text-primary"
			onclick={copyIsbn}
			title="{nummerArt} kopieren"
		>
			<span class="truncate" title={titel.isbn}>{nummerArt}: {titel.isbn || '–'}</span>
			{#if copied}
				<span class="font-sans text-xs font-medium text-primary">Kopiert!</span>
			{:else if titel.isbn}
				<Copy class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
			{/if}
		</button>

		{#if titel.signatur}
			<div class="flex items-center gap-1.5">
				<MapPin class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
				<span class="truncate font-mono">{titel.signatur}</span>
			</div>
		{/if}

		<!-- mt-2: Die Trefferfläche des Stifts ragt 8 px über ihn hinaus und berührt so die
		     Zeile darüber nicht. -->
		<div class="mt-2 flex items-center justify-between gap-2">
			<span>{bestand}</span>
			{#if onEditClick}
				<!-- Ruhig grau; sitzt im Text statt auf dem Cover. -->
				<button
					class="icon-btn -my-1 shrink-0"
					onclick={(e) => {
						e.stopPropagation();
						onEditClick();
					}}
					data-tip="Schnell bearbeiten"
					aria-label="Buch schnell bearbeiten"
				>
					<SquarePen class="h-4 w-4" aria-hidden="true" />
				</button>
			{/if}
		</div>
		{#if titel.buch}
			<p class="text-xs">{auflagenAufschluesselung(titel.buch.auflagen)}</p>
		{/if}

		{#if geprueft}
			<span>Geprüft {geprueft}</span>
		{/if}
	</div>
</div>
