<script>
	import { getSubjectColor, getStockDotColor, formatDate } from '../bookHelpers.js';
	import { coverKandidaten } from '../../../lib/utils/coverSrc.js';
	import BuchKarteCover from './BuchKarteCover.svelte';
	import { Clock, Copy, SquarePen } from '@lucide/svelte';

	/**
	 * @type {{
	 *   book: {
	 *     id: string,
	 *     isbn: string,
	 *     title: string,
	 *     author: string,
	 *     subject: string,
	 *     gradeLevel: number,
	 *     track: string,
	 *     stock: number,
	 *     verfuegbar: number,
	 *     gesamt: number,
	 *     coverUrl: string,
	 *     lastCounted: string,
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

	/**
	 * @param {string} isbn
	 */
	function copyIsbn(isbn) {
		if (!isbn) return;
		navigator.clipboard.writeText(isbn);
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

	/**
	 * @param {Event} event
	 */
	function onCoverLoad(event) {
		const image = /** @type {HTMLImageElement} */ (event.currentTarget);
		if (image.naturalWidth < 10 || image.naturalHeight < 10) {
			onCoverError();
		}
	}

	$effect(() => {
		const candidates = [];
		candidates.push(...coverKandidaten(book?.coverUrl, book?.isbn));
		coverCandidates = candidates;
		currentCandidateIndex = 0;
		coverFailed = candidates.length === 0;
	});
</script>

<!-- role="button" + tabindex + Tastaturweg: Die ganze Karte ist anklickbar, war aber
     ausschliesslich mit der Maus erreichbar. Die Pruefung auf currentTarget haelt die
     inneren Schaltflaechen (Quick-Edit) heraus — deren Enter darf nicht zusaetzlich die
     Karte oeffnen. -->
<!-- <div> statt <article>: Ein <article> ist nicht-interaktiv und darf laut ARIA keine
     Button-Rolle tragen — der Svelte-Compiler weist das zu Recht ab. Zwischen "korrekte
     Dokumentgliederung" und "mit der Tastatur bedienbar" ist Letzteres wichtiger; die
     Karte IST eine Schaltflaeche, kein eigenstaendiger Artikel. -->
<div
	class="bg-white rounded-xl border border-slate-200 flex flex-col h-full group overflow-hidden hover:border-blue-300 hover:shadow-md transition-all duration-300 shadow-sm cursor-pointer relative"
	role="button"
	tabindex="0"
	{onclick}
	onkeydown={(e) => {
		if (e.target !== e.currentTarget) return;
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			onclick?.(e);
		}
	}}
>
	<!-- Quick-Edit Stift-Icon.
	     Stand bis zum 10.08.2026 auf `opacity-0 group-hover:opacity-100` und war damit aus
	     drei unabhängigen Gründen unerreichbar: Auf einem Tablet gibt es kein :hover, ein
	     Fingertipp löst stattdessen den Klick der Karte aus; per Tastatur bekam der Knopf
	     zwar den Fokus, blieb dabei aber unsichtbar (WCAG 2.4.7); und Material 3 versteckt
	     Aktionen nicht hinter dem Zeiger. Gefunden hat es der Rundgang
	     (e2e/rundgang-alle-routen.spec.js) — 50 Fundstellen auf einer Seite, dieselbe
	     Bauart wie die Karussell-Pfeile zwei Commits zuvor.
	     Die Zurückhaltung trägt jetzt die Farbe, nicht die Sichtbarkeit: ruhiges Grau im
	     Ruhezustand, Blau beim Zeigen. -->
	{#if onEditClick}
		<button
			class="absolute top-2 right-2 z-10 p-2 rounded-lg bg-white/80 backdrop-blur-sm border border-slate-200 text-slate-500 hover:text-blue-600 hover:border-blue-300 hover:bg-blue-50 transition-all duration-200 shadow-sm cursor-pointer"
			onclick={(e) => {
				e.stopPropagation();
				onEditClick();
			}}
			title="Schnell bearbeiten"
			aria-label="Buch schnell bearbeiten"
		>
			<!-- p-2 + 16 px Symbol = 32×32 Trefferfläche. Vorher p-1.5 + 14 px = 28×28 und damit
			     unter der Mindestgröße — fiel nicht auf, solange das Gate den öffentlichen OPAC
			     statt des internen Katalogs vermaß (Pfadkollision /katalog). -->
			<SquarePen class="w-4 h-4" aria-hidden="true" />
		</button>
	{/if}
	{#if coverSrc && !coverFailed}
		<div
			class="w-full h-56 rounded-t-2xl overflow-hidden bg-slate-50 flex items-center justify-center border-b border-slate-100 relative"
		>
			<img
				src={coverSrc}
				alt={`Cover von ${book.title}`}
				loading="lazy"
				class="object-contain h-full w-full p-3 transition-all duration-500 group-hover:scale-105"
				onerror={onCoverError}
				onload={onCoverLoad}
			/>
		</div>
	{:else}
		<BuchKarteCover
			subject={book.subject}
			title={book.title}
			author={book.author}
			medientyp={book.medientyp}
			isbn={book.isbn}
		/>
	{/if}

	<div class="grow p-5 pt-4 flex flex-col justify-between">
		<div>
			<h2
				class="text-base font-bold text-slate-900 leading-snug mb-1 line-clamp-2"
				title={book.title}
			>
				{book.title}
			</h2>
			<!-- Die ISBN ist keine Fußnote, sondern eine Nummer, die man ABLIEST: gegen das
			     Buch in der Hand, gegen das Lieferantenformular, gegen die Bestellliste.
			     Sie stand auf text-label-small (11 px) und damit auf derselben Stufe wie
			     die Fach-Abzeichen und „Zuletzt geprüft" — also so klein wie das
			     Unwichtigste auf der Karte.
			     text-sm + font-mono ist keine neue Erfindung: Genau so zeigt
			     BookAkteMeta.svelte dieselbe ISBN einen Klick tiefer, und genau so trägt
			     die Profilkarte den Schüler-Barcode. Feste Zeichenbreite trennt die 13
			     Ziffern, ohne dass man mitzählen muss. Die Farbe bleibt: slate-400 ist seit
			     dem WCAG-Durchgang #5a5f66 und damit klar über der AA-Grenze. -->
			<button
				class="text-sm font-mono text-slate-400 mb-4 tracking-wide group/isbn flex items-center gap-2 text-left transition-colors hover:text-blue-600 cursor-pointer"
				onclick={(e) => {
					e.stopPropagation();
					copyIsbn(book.isbn);
				}}
				title={(book.medientyp === 'CD' || book.medientyp === 'DVD' ? 'EAN' : 'ISBN') + ' kopieren'}
				aria-label={(book.medientyp === 'CD' || book.medientyp === 'DVD' ? 'EAN' : 'ISBN') +
					' kopieren'}
			>
				<span
					>{book.medientyp === 'CD' || book.medientyp === 'DVD' ? 'EAN' : 'ISBN'}: {book.isbn ||
						'-'}</span
				>
				{#if book.isbn}
					{#if copied}
						<!-- Bleibt eine Stufe unter der Nummer: Die Rückmeldung soll bestätigen,
						     nicht mit dem konkurrieren, was man gerade abliest. -->
						<span class="text-blue-600 text-xs font-sans font-bold">Kopiert!</span>
					{:else}
						<Copy
							class="w-3.5 h-3.5 text-slate-300 opacity-0 group-hover/isbn:opacity-100 transition-opacity"
							aria-hidden="true"
						/>
					{/if}
				{/if}
			</button>

			<div class="flex flex-wrap gap-1.5 mb-4">
				{#if book.subject}
					<span
						class="{getSubjectColor(
							book.subject
						)} text-label-small font-bold px-2 py-0.5 rounded-md"
					>
						{book.subject}
					</span>
				{/if}
				<!-- Klasse 0 = nicht zugeordnet: kein Badge statt einer sinnlosen „Klasse 0". -->
				{#if book.gradeLevel}
					<span
						class="bg-slate-50 border border-slate-200 text-slate-600 text-label-small font-bold px-2 py-0.5 rounded-md"
					>
						Klasse {book.gradeLevel}
					</span>
				{/if}
				{#if book.track}
					<span
						class="bg-cyan-50 border border-cyan-200 text-cyan-700 text-label-small font-bold px-2 py-0.5 rounded-md"
					>
						{book.track}
					</span>
				{/if}
			</div>
		</div>

		<div class="space-y-4">
			<div
				class="inline-flex items-center gap-1.5 w-full px-2.5 py-1.5 rounded-lg bg-slate-50 border border-slate-100 text-label-small text-slate-500 font-medium"
			>
				<Clock class="w-3.5 h-3.5 text-slate-400" aria-hidden="true" />
				<span>
					Zuletzt geprüft: {formatDate(book.lastCounted) || 'Unbekannt'}
				</span>
			</div>

			<div class="pt-3 border-t border-slate-100 flex justify-between items-center">
				<span class="text-xs font-semibold text-slate-400">Verfügbar</span>
				<div class="flex items-center gap-2">
					<span class="w-2 h-2 rounded-full {getStockDotColor(book.verfuegbar || 0)}"></span>
					<span class="text-lg font-extrabold text-slate-800">{book.verfuegbar || 0}</span>
					{#if book.gesamt !== undefined}
						<span class="text-xs text-slate-500 font-medium">/ {book.gesamt}</span>
					{/if}
				</div>
			</div>
		</div>
	</div>
</div>
