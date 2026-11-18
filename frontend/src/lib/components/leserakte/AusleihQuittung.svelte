<script>
	import BuchCover from '../ui/BuchCover.svelte';
	import { istUeberfaellig } from '../../utils/ueberfaellig.js';
	import { formatDatum } from '../../utils/format.js';

	/**
	 * @typedef {Object} Props
	 * @property {any} leser
	 */
	/** @type {Props} */
	let { leser } = $props();

	function formatDate(dateString) {
		return formatDatum(dateString) || 'Keine Angabe';
	}
</script>

<!-- Print Container für Ausleihen -->
<div class="print-receipt-section" style="display:none">
	<div class="text-center mb-8 border-b border-outline-variant pb-4">
		<h1 class="text-2xl">Ausleih-Quittung</h1>
		<h2 class="text-lg text-on-surface-variant">Schulbibliothek</h2>
	</div>

	<div class="flex justify-between mb-8">
		<div>
			<p class="text-sm text-on-surface-variant">Schüler</p>
			<p class="font-bold text-lg">{leser.vorname} {leser.nachname}</p>
			<p class="text-sm">{leser.klasse || ''}</p>
		</div>
		<div class="text-right">
			<p class="text-sm text-on-surface-variant">Datum</p>
			<p class="text-body-large-emphasized">{formatDatum(new Date())}</p>
		</div>
	</div>

	<div class="mb-4">
		<h3 class="text-title-medium mb-2 border-b border-outline-variant pb-2">Offene Ausleihen</h3>
		{#if leser.entliehene_buecher && leser.entliehene_buecher.length > 0}
			<table class="w-full text-left text-sm border-collapse">
				<thead>
					<tr class="border-b border-outline-variant">
						<th class="py-2 px-2 w-12 text-title-small">Cover</th>
						<th class="py-2 px-2 text-title-small">Titel</th>
						<th class="py-2 px-2 text-center text-title-small">Barcode und Signatur</th>
						<th class="py-2 px-2 text-center text-title-small">Ausgeliehen am</th>
						<th class="py-2 px-2 text-right text-title-small">Rückgabe bis</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-outline-variant">
					{#each leser.entliehene_buecher as book, _i (_i)}
						<tr>
							<td class="py-3 px-2">
								<BuchCover
									coverUrl={book.cover_url}
									isbn={book.isbn}
									titel={book.titel}
									nurGespeichert
								/>
							</td>
							<td class="py-3 px-2">
								<div class="text-body-medium-emphasized">{book.titel}</div>
								<div class="text-xs text-on-surface-variant">{book.autor}</div>
							</td>
							<td class="py-3 px-2 text-center font-mono text-xs">{book.barcode_id || '–'}</td>
							<td class="py-3 px-2 text-center">{formatDate(book.ausgeliehen_am)}</td>
							<!-- Eine Dauerleihe hat keine Frist, wie in der Leserakte. -->
							<td
								class="py-3 px-2 text-right text-body-medium-emphasized {istUeberfaellig(book)
									? 'text-error'
									: ''}"
							>
								{book.ist_dauerleihe ? 'ohne Frist' : formatDate(book.rueckgabe_frist)}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{:else}
			<p class="text-on-surface-variant italic">Keine offenen Ausleihen</p>
		{/if}
	</div>
</div>
