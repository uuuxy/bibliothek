<script>
	import { onMount } from 'svelte';
	import { apiGet } from '../../apiFetch.js';

	/** @type {any[]} */
	let bestellungen = $state([]);
	let loading = $state(true);
	/** @type {string|null} */
	let expandedId = $state(null);

	onMount(async () => {
		bestellungen = (await apiGet('/api/bestellhistorie')) || [];
		loading = false;
	});

	let gesamtsumme = $derived(bestellungen.reduce((sum, b) => sum + b.gesamtbetrag, 0));

	/** @param {number} n */
	function euro(n) {
		return n.toLocaleString('de-DE', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) + ' €';
	}

	/** @param {string} iso */
	function datum(iso) {
		return new Date(iso).toLocaleDateString('de-DE', {
			day: '2-digit',
			month: '2-digit',
			year: 'numeric'
		});
	}

	/** @param {string} id */
	function toggleExpand(id) {
		expandedId = expandedId === id ? null : id;
	}
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between border-b border-slate-200 pb-4">
		<div>
			<h2 class="text-base font-bold text-slate-800">Bestellhistorie</h2>
			<p class="text-sm text-slate-500 mt-0.5">
				Alle aufgegebenen Bestellungen — automatisch erfasst beim Bestellen
			</p>
		</div>
		{#if bestellungen.length > 0}
			<div class="text-right">
				<div class="text-xs text-slate-400 font-semibold">Gesamtausgaben</div>
				<div class="text-2xl font-black text-slate-800">{euro(gesamtsumme)}</div>
			</div>
		{/if}
	</div>

	{#if loading}
		<div class="py-16 text-center text-slate-400 text-base animate-pulse">
			Lade Bestellhistorie…
		</div>
	{:else if bestellungen.length === 0}
		<div class="py-16 text-center text-slate-400 text-base">
			Noch keine Bestellungen aufgegeben.<br />
			<span class="text-sm">Bestellungen werden hier automatisch gespeichert.</span>
		</div>
	{:else}
		<!-- Eine Kopfzeile für alle Bestellungen statt vier Beschriftungen je Zeile: Der Inhalt ist
		     ein gleichförmiger Datensatz mit festen Spalten, also eine Tabelle. Als Karten brauchte
		     dieselbe Information rund das Dreifache an Höhe. Die Zeile bleibt aufklappbar — sie
		     trägt dafür role="button" samt Tastaturbedienung, das Chevron ist nur noch Anzeige. -->
		<div class="overflow-x-auto rounded-xl border border-slate-200 bg-white shadow-xs">
			<table class="w-full border-collapse text-sm">
				<thead>
					<tr class="border-b border-slate-200 bg-slate-50/60 text-xs font-semibold text-slate-400">
						<th class="px-3 py-2 text-left font-semibold">Datum</th>
						<th class="px-3 py-2 text-left font-semibold">Lieferant</th>
						<th class="px-3 py-2 text-right font-semibold">Exemplare</th>
						<th class="px-3 py-2 text-right font-semibold">Betrag</th>
						<th class="w-8 px-3 py-2"><span class="sr-only">Positionen</span></th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100">
					{#each bestellungen as b (b.id)}
						<tr
							role="button"
							tabindex="0"
							aria-expanded={expandedId === b.id}
							aria-controls="positionen-{b.id}"
							onclick={() => toggleExpand(b.id)}
							onkeydown={(e) => {
								if (e.key === 'Enter' || e.key === ' ') {
									e.preventDefault();
									toggleExpand(b.id);
								}
							}}
							class="cursor-pointer transition-colors hover:bg-slate-50/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-blue-500"
						>
							<td class="px-3 py-2 font-semibold whitespace-nowrap text-slate-800 tabular-nums">
								{datum(b.bestelldatum)}
							</td>
							<td class="max-w-0 px-3 py-2">
								<span class="block truncate font-semibold text-slate-800">{b.lieferant_name}</span>
								<span class="block truncate text-xs text-slate-400">
									{b.kundennummer ? 'Kd.-Nr. ' + b.kundennummer : b.lieferant_email}
								</span>
							</td>
							<td class="px-3 py-2 text-right whitespace-nowrap text-slate-700 tabular-nums">
								{b.anzahl_exemplare}
							</td>
							<td
								class="px-3 py-2 text-right font-bold whitespace-nowrap text-slate-900 tabular-nums"
							>
								{euro(b.gesamtbetrag)}
							</td>
							<td class="px-3 py-2 text-right">
								<span
									aria-hidden="true"
									class="inline-block text-slate-400 transition-transform {expandedId === b.id
										? 'rotate-180'
										: ''}">▾</span
								>
							</td>
						</tr>

						<!-- Positionen -->
						{#if expandedId === b.id}
							<tr id="positionen-{b.id}">
								<td colspan="5" class="border-t border-slate-100 bg-slate-50/40 px-5 py-4">
									{#if b.positionen.length === 0}
										<p class="text-sm text-slate-400 italic">Keine Positionen gespeichert.</p>
									{:else}
										<table class="w-full border-collapse text-sm">
											<thead>
												<tr class="border-b border-slate-200 text-xs font-semibold text-slate-400">
													<th class="pb-1.5 text-left font-semibold">Titel</th>
													<th class="pb-1.5 text-left font-semibold">ISBN</th>
													<th class="pb-1.5 text-right font-semibold">Menge</th>
													<th class="pb-1.5 text-right font-semibold">Einzelpreis</th>
													<th class="pb-1.5 text-right font-semibold">Gesamt</th>
												</tr>
											</thead>
											<tbody class="divide-y divide-slate-100">
												{#each b.positionen as p, _i (_i)}
													<tr>
														<td class="py-1.5 pr-4 font-medium text-slate-800">{p.titel_name}</td>
														<td class="py-1.5 pr-4 font-mono text-xs text-slate-500"
															>{p.isbn || '—'}</td
														>
														<td class="py-1.5 text-right text-slate-700 tabular-nums">{p.menge}</td>
														<td class="py-1.5 text-right text-slate-700 tabular-nums"
															>{euro(p.einzelpreis)}</td
														>
														<td class="py-1.5 text-right font-bold text-slate-800 tabular-nums"
															>{euro(p.gesamtpreis)}</td
														>
													</tr>
												{/each}
											</tbody>
											<tfoot>
												<tr class="border-t-2 border-slate-200">
													<td colspan="4" class="pt-1.5 text-right text-sm font-bold text-slate-600"
														>Summe</td
													>
													<td class="pt-1.5 text-right font-black text-slate-900 tabular-nums"
														>{euro(b.gesamtbetrag)}</td
													>
												</tr>
											</tfoot>
										</table>
									{/if}
								</td>
							</tr>
						{/if}
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>
