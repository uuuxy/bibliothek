<!-- @component BuchListeStartseite — die Buch-Suche als dichte Liste: eine Zeile je
     Titel mit kleinem Cover, Fach, Klasse, Zweig, Signatur und Bestand. Die Alternative
     zum Cover-Raster (BuchRasterStartseite), wenn man viele Titel überblicken will:
     Eine Karte ist rund 430 px hoch, acht Titel je Bildschirm — hier sind es dreißig.
     Nur lesen; jede Zeile öffnet wie die Karte die Buchakte. -->
<script>
	import { getSubjectColor } from '../bookHelpers.js';
	import { coverKandidaten } from '../../../lib/utils/coverSrc.js';

	/** @type {{ filteredBooks: any[], onBookClick: (book: any) => void }} */
	let { filteredBooks, onBookClick } = $props();

	/** Jahrgangsspanne, sonst gradeLevel — dieselbe Lesart wie der Reiter „Jahrgänge". */
	function klasseVon(/** @type {any} */ b) {
		if (b.jahrgangVon && b.jahrgangBis) {
			return b.jahrgangVon === b.jahrgangBis
				? String(b.jahrgangVon)
				: `${b.jahrgangVon}–${b.jahrgangBis}`;
		}
		return b.gradeLevel ? String(b.gradeLevel) : '';
	}

	/** @param {Event} e */
	function coverVersteckenBeiFehler(e) {
		/** @type {HTMLImageElement} */ (e.currentTarget).style.visibility = 'hidden';
	}
</script>

<div class="overflow-x-auto">
	<!-- table-fixed: Spalten springen sonst mit dem Inhalt (leere Signatur-Spalte → alles rutscht). -->
	<table class="w-full table-fixed text-left text-sm">
		<thead class="border-b border-outline-variant text-xs font-medium text-on-surface-variant">
			<tr>
				<th class="w-12 py-2 pr-2"><span class="sr-only">Cover</span></th>
				<th class="py-2 pr-3">Titel</th>
				<th class="w-36 py-2 pr-3">Fach</th>
				<th class="w-20 py-2 pr-3">Klasse</th>
				<th class="w-32 py-2 pr-3">Zweig</th>
				<th class="w-40 py-2 pr-3">Signatur</th>
				<th class="w-36 py-2 text-right">Verfügbar</th>
			</tr>
		</thead>
		<tbody>
			{#each filteredBooks as book (book.id)}
				<tr
					class="m3-state cursor-pointer border-b border-outline-variant"
					role="button"
					tabindex="0"
					onclick={() => onBookClick(book)}
					onkeydown={(e) => {
						if (e.key === 'Enter' || e.key === ' ') {
							e.preventDefault();
							onBookClick(book);
						}
					}}
				>
					<td class="py-1.5 pr-2">
						{#if coverKandidaten(book.coverUrl, book.isbn)[0]}
							<img
								src={coverKandidaten(book.coverUrl, book.isbn)[0]}
								alt=""
								loading="lazy"
								class="h-10 w-7 rounded-xs object-cover"
								onerror={coverVersteckenBeiFehler}
							/>
						{/if}
					</td>
					<td class="py-1.5 pr-3">
						<div class="truncate font-medium text-on-surface" title={book.title}>{book.title}</div>
						{#if book.author}
							<div class="truncate text-sm text-on-surface-variant">{book.author}</div>
						{/if}
					</td>
					<td class="py-1.5 pr-3">
						{#if book.subject}
							<span
								class="{getSubjectColor(book.subject)} rounded-md px-2 py-0.5 text-xs font-bold"
								data-chip>{book.subject}</span
							>
						{/if}
					</td>
					<td class="py-1.5 pr-3 text-on-surface-variant">{klasseVon(book)}</td>
					<td class="py-1.5 pr-3 text-on-surface-variant">{book.track || ''}</td>
					<td class="py-1.5 pr-3 font-mono text-on-surface-variant">{book.signatur || ''}</td>
					<td class="py-1.5 text-right whitespace-nowrap">
						{#if !book.gesamt}
							<span class="text-on-surface-variant">Keine Exemplare</span>
						{:else}
							<span class="font-semibold text-on-surface">{book.verfuegbar ?? 0}</span>
							<span class="text-on-surface-variant">/ {book.gesamt}</span>
						{/if}
					</td>
				</tr>
			{/each}
		</tbody>
	</table>
</div>
