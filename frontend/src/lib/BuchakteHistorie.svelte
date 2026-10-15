<script>
	import { Clock } from '@lucide/svelte';
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
	<div class="py-16 flex flex-col items-center text-on-surface-variant gap-3">
		<Clock class="w-10 h-10" aria-hidden="true" />
		<p class="font-semibold text-sm">Noch keine Ausleihen in der Datenbank vorhanden.</p>
	</div>
{:else}
	<div class="w-full">
		<div class="px-1 py-3 border-b border-outline-variant flex items-center justify-between">
			<p class="text-sm font-medium text-on-surface-variant">Letzte {history.length} Ausleihen</p>
		</div>
		<ul class="divide-y divide-outline-variant">
			{#each history as h, _i (_i)}
				<li class="px-5 py-3 flex items-center justify-between hover:bg-surface transition-colors">
					<div class="flex items-center gap-3 min-w-0">
						<div
							class="w-8 h-8 rounded-full bg-surface-container-highest text-on-surface-variant flex items-center justify-center font-bold text-xs shrink-0"
						>
							{h.schueler_name?.[0] ?? ''}{h.schueler_nachname?.[0] ?? ''}
						</div>
						<div class="min-w-0">
							<p class="text-sm font-semibold text-on-surface truncate">
								{h.schueler_name}
								{h.schueler_nachname}
								<span class="text-xs font-normal text-on-surface-variant">({h.klasse})</span>
							</p>
							<p class="text-xs text-on-surface-variant font-mono">
								Exemplar: {h.exemplar_barcode}
							</p>
						</div>
					</div>
					<div class="text-right shrink-0 ml-4 space-y-0.5">
						<p class="text-xs text-on-surface-variant">
							<span class="font-medium">Von</span>
							{fmtDate(h.ausgeliehen_am)}
						</p>
						<p class="text-xs {h.rueckgabe_am ? 'text-success' : 'text-warning'} font-semibold">
							{h.rueckgabe_am ? `Zurück ${fmtDate(h.rueckgabe_am)}` : 'Noch ausgeliehen'}
						</p>
					</div>
				</li>
			{/each}
		</ul>
	</div>
{/if}
