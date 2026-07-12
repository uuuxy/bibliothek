<script>
	/** @type {{ history: any[] }} */
	let { history } = $props();

	/** @param {string} d */
	function fmtDate(d) {
		if (!d) return '-';
		try {
			return new Date(d).toLocaleDateString('de-DE');
		} catch {
			return d;
		}
	}
</script>

{#if history.length === 0}
	<div class="py-16 flex flex-col items-center text-slate-400 gap-3">
		<svg class="w-10 h-10" fill="none" stroke="currentColor" viewBox="0 0 24 24"
			><path
				stroke-linecap="round"
				stroke-linejoin="round"
				stroke-width="1.5"
				d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
			/></svg
		>
		<p class="font-semibold text-sm">Noch keine Ausleihen in der Datenbank vorhanden.</p>
	</div>
{:else}
	<div class="w-full">
		<div class="px-1 py-3 border-b border-gray-200 flex items-center justify-between">
			<p class="text-sm font-medium text-gray-600">Letzte {history.length} Ausleihen</p>
		</div>
		<ul class="divide-y divide-slate-50">
			{#each history as h, _i (_i)}
				<li class="px-5 py-3 flex items-center justify-between hover:bg-slate-50 transition-colors">
					<div class="flex items-center gap-3 min-w-0">
						<div
							class="w-8 h-8 rounded-full bg-slate-100 text-slate-500 flex items-center justify-center font-bold text-xs shrink-0"
						>
							{h.schueler_name?.[0] ?? ''}{h.schueler_nachname?.[0] ?? ''}
						</div>
						<div class="min-w-0">
							<p class="text-sm font-semibold text-slate-800 truncate">
								{h.schueler_name}
								{h.schueler_nachname}
								<span class="text-xs font-normal text-slate-400">({h.klasse})</span>
							</p>
							<p class="text-xs text-slate-400 font-mono">Exemplar: {h.exemplar_barcode}</p>
						</div>
					</div>
					<div class="text-right shrink-0 ml-4 space-y-0.5">
						<p class="text-xs text-slate-500">
							<span class="font-medium text-slate-400">Von</span>
							{fmtDate(h.ausgeliehen_am)}
						</p>
						<p
							class="text-xs {h.rueckgabe_am ? 'text-emerald-600' : 'text-amber-600'} font-semibold"
						>
							{h.rueckgabe_am ? `Zurück ${fmtDate(h.rueckgabe_am)}` : 'Noch ausgeliehen'}
						</p>
					</div>
				</li>
			{/each}
		</ul>
	</div>
{/if}
