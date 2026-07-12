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
				<div class="text-xs text-slate-400 uppercase tracking-wide font-semibold">
					Gesamtausgaben
				</div>
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
		<div class="space-y-3">
			{#each bestellungen as b (b.id)}
				<div class="border border-slate-200 rounded-xl overflow-hidden bg-white shadow-xs">
					<!-- Kopfzeile -->
					<button
						onclick={() => toggleExpand(b.id)}
						class="w-full text-left px-5 py-4 flex items-center justify-between gap-4 hover:bg-slate-50/60 transition-colors cursor-pointer"
					>
						<div class="flex items-center gap-6 flex-1 min-w-0">
							<div class="shrink-0">
								<div class="text-xs text-slate-400 font-semibold uppercase tracking-wide">
									Datum
								</div>
								<div class="font-bold text-slate-800 text-sm">{datum(b.bestelldatum)}</div>
							</div>
							<div class="min-w-0">
								<div class="text-xs text-slate-400 font-semibold uppercase tracking-wide">
									Lieferant
								</div>
								<div class="font-bold text-slate-800 text-sm truncate">{b.lieferant_name}</div>
								<div class="text-xs text-slate-400 truncate">
									{b.kundennummer ? 'Kd.-Nr. ' + b.kundennummer : b.lieferant_email}
								</div>
							</div>
							<div class="shrink-0">
								<div class="text-xs text-slate-400 font-semibold uppercase tracking-wide">
									Exemplare
								</div>
								<div class="font-bold text-slate-800 text-sm">{b.anzahl_exemplare}</div>
							</div>
						</div>
						<div class="shrink-0 text-right">
							<div class="text-xs text-slate-400 font-semibold uppercase tracking-wide">Betrag</div>
							<div class="text-lg font-black text-slate-900">{euro(b.gesamtbetrag)}</div>
						</div>
						<div
							class="text-slate-400 shrink-0 text-lg transition-transform {expandedId === b.id
								? 'rotate-180'
								: ''}"
						>
							▾
						</div>
					</button>

					<!-- Positionen -->
					{#if expandedId === b.id}
						<div class="border-t border-slate-100 bg-slate-50/40 px-5 py-4">
							{#if b.positionen.length === 0}
								<p class="text-sm text-slate-400 italic">Keine Positionen gespeichert.</p>
							{:else}
								<table class="w-full text-sm border-collapse">
									<thead>
										<tr
											class="text-xs font-semibold text-slate-400 uppercase tracking-wide border-b border-slate-200"
										>
											<th class="pb-2 text-left font-semibold">Titel</th>
											<th class="pb-2 text-left font-semibold">ISBN</th>
											<th class="pb-2 text-right font-semibold">Menge</th>
											<th class="pb-2 text-right font-semibold">Einzelpreis</th>
											<th class="pb-2 text-right font-semibold">Gesamt</th>
										</tr>
									</thead>
									<tbody class="divide-y divide-slate-100">
										{#each b.positionen as p, _i (_i)}
											<tr>
												<td class="py-2 font-medium text-slate-800 pr-4">{p.titel_name}</td>
												<td class="py-2 text-slate-500 font-mono text-xs pr-4">{p.isbn || '—'}</td>
												<td class="py-2 text-right text-slate-700">{p.menge}</td>
												<td class="py-2 text-right text-slate-700">{euro(p.einzelpreis)}</td>
												<td class="py-2 text-right font-bold text-slate-800"
													>{euro(p.gesamtpreis)}</td
												>
											</tr>
										{/each}
									</tbody>
									<tfoot>
										<tr class="border-t-2 border-slate-200">
											<td colspan="4" class="pt-2 text-right font-bold text-slate-600 text-sm"
												>Summe</td
											>
											<td class="pt-2 text-right font-black text-slate-900"
												>{euro(b.gesamtbetrag)}</td
											>
										</tr>
									</tfoot>
								</table>
							{/if}
						</div>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>
