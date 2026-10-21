<script>
	import BuchCover from './components/ui/BuchCover.svelte';
	import { istUeberfaellig } from './utils/ueberfaellig.js';

	/**
	 * @typedef {Object} Props
	 * @property {any} leser
	 */
	/** @type {Props} */
	let { leser } = $props();

	function formatDate(dateString) {
		if (!dateString) return 'Keine Angabe';
		try {
			const d = new Date(dateString);
			return d.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit', year: 'numeric' });
		} catch {
			return dateString;
		}
	}
</script>

<!-- Print Container für Ausleihen -->
<div class="print-receipt-section" style="display:none">
	<div class="text-center mb-8 border-b border-outline-variant pb-4">
		<h1 class="text-2xl font-bold">Ausleih-Quittung</h1>
		<h2 class="text-lg text-on-surface-variant">Schulbibliothek</h2>
	</div>

	<div class="flex justify-between mb-8">
		<div>
			<p class="text-sm text-on-surface-variant">Schüler/in</p>
			<p class="font-bold text-lg">{leser.vorname} {leser.nachname}</p>
			<p class="text-sm">{leser.klasse || ''}</p>
		</div>
		<div class="text-right">
			<p class="text-sm text-on-surface-variant">Datum</p>
			<p class="font-bold">{new Date().toLocaleDateString('de-DE')}</p>
		</div>
	</div>

	<div class="mb-4">
		<h3 class="font-bold text-lg mb-2 border-b border-outline-variant pb-2">Offene Ausleihen</h3>
		{#if leser.entliehene_buecher && leser.entliehene_buecher.length > 0}
			<table class="w-full text-left text-sm border-collapse">
				<thead>
					<tr class="border-b border-outline-variant">
						<th class="py-2 px-2 font-semibold w-12">Cover</th>
						<th class="py-2 px-2 font-semibold">Titel</th>
						<th class="py-2 px-2 font-semibold text-center">Barcode/Signatur</th>
						<th class="py-2 px-2 font-semibold text-center">Ausgeliehen am</th>
						<th class="py-2 px-2 font-semibold text-right">Rückgabe bis</th>
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
								<div class="font-bold">{book.titel}</div>
								<div class="text-xs text-on-surface-variant">{book.autor}</div>
							</td>
							<td class="py-3 px-2 text-center font-mono text-xs">{book.barcode_id || '-'}</td>
							<td class="py-3 px-2 text-center">{formatDate(book.ausgeliehen_am)}</td>
							<!-- Eine Dauerleihe hat keine Frist, wie in der Leserakte. -->
							<td
								class="py-3 px-2 text-right font-bold {istUeberfaellig(book) ? 'text-error' : ''}"
							>
								{book.ist_dauerleihe ? 'ohne Frist' : formatDate(book.rueckgabe_frist)}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{:else}
			<p class="text-on-surface-variant italic">Keine offenen Ausleihen.</p>
		{/if}
	</div>
</div>
