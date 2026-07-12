<script>
	/**
	 * @typedef {Object} Props
	 * @property {any} profile
	 */
	/** @type {Props} */
	let { profile } = $props();

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
	<div class="text-center mb-8 border-b border-slate-300 pb-4">
		<h1 class="text-2xl font-bold">Ausleih-Quittung</h1>
		<h2 class="text-lg text-slate-600">Schulbibliothek</h2>
	</div>

	<div class="flex justify-between mb-8">
		<div>
			<p class="text-sm text-slate-500">Schüler/in</p>
			<p class="font-bold text-lg">{profile.vorname} {profile.nachname}</p>
			<p class="text-sm">{profile.klasse || ''}</p>
		</div>
		<div class="text-right">
			<p class="text-sm text-slate-500">Datum</p>
			<p class="font-bold">{new Date().toLocaleDateString('de-DE')}</p>
		</div>
	</div>

	<div class="mb-4">
		<h3 class="font-bold text-lg mb-2 border-b border-slate-300 pb-2">Offene Ausleihen</h3>
		{#if profile.entliehene_buecher && profile.entliehene_buecher.length > 0}
			<table class="w-full text-left text-sm border-collapse">
				<thead>
					<tr class="border-b border-slate-300">
						<th class="py-2 px-2 font-semibold w-12">Cover</th>
						<th class="py-2 px-2 font-semibold">Titel</th>
						<th class="py-2 px-2 font-semibold text-center">Barcode/Signatur</th>
						<th class="py-2 px-2 font-semibold text-center">Ausgeliehen am</th>
						<th class="py-2 px-2 font-semibold text-right">Rückgabe bis</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-200">
					{#each profile.entliehene_buecher as book, _i (_i)}
						<tr>
							<td class="py-3 px-2">
								{#if book.cover_url}
									<img
										src={book.cover_url}
										alt="Cover"
										class="w-8 h-12 object-cover rounded shadow-sm"
									/>
								{:else}
									<div
										class="w-8 h-12 bg-slate-100 rounded flex items-center justify-center text-xs text-slate-400"
									>
										📖
									</div>
								{/if}
							</td>
							<td class="py-3 px-2">
								<div class="font-bold">{book.titel}</div>
								<div class="text-xs text-slate-500">{book.autor}</div>
							</td>
							<td class="py-3 px-2 text-center font-mono text-xs"
								>{book.barcode || book.signatur || '-'}</td
							>
							<td class="py-3 px-2 text-center">{formatDate(book.ausleih_datum)}</td>
							<td
								class="py-3 px-2 text-right font-bold {new Date(book.rueckgabe_datum) < new Date()
									? 'text-red-600'
									: ''}"
							>
								{formatDate(book.rueckgabe_datum)}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{:else}
			<p class="text-slate-500 italic">Keine offenen Ausleihen.</p>
		{/if}
	</div>
</div>
