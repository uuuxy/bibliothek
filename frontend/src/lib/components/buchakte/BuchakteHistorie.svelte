<script>
	import SuchZustand from '../ui/SuchZustand.svelte';
	import { formatDatum, formatZahl } from '../../utils/format.js';
	/** @type {{ verlauf: any[] }} */
	let { verlauf } = $props();
</script>

{#if verlauf.length === 0}
	<SuchZustand symbol="schedule" titel="Noch keine Ausleihen" />
{:else}
	<div class="w-full">
		<div class="px-1 py-3 border-b border-outline-variant flex items-center justify-between">
			<p class="text-sm font-medium text-on-surface-variant">
				{verlauf.length === 1
					? 'Letzte Ausleihe'
					: `Letzte ${formatZahl(verlauf.length)} Ausleihen`}
			</p>
		</div>
		<ul class="divide-y divide-outline-variant">
			{#each verlauf as h, _i (_i)}
				<li class="px-5 py-3 flex items-center justify-between hover:bg-surface transition-colors">
					<div class="flex items-center gap-3 min-w-0">
						<div
							class="w-8 h-8 rounded-full bg-surface-container-highest text-on-surface-variant flex items-center justify-center text-label-medium shrink-0"
						>
							{h.schueler_name?.[0] ?? ''}{h.schueler_nachname?.[0] ?? ''}
						</div>
						<div class="min-w-0">
							<p class="text-body-medium-emphasized text-on-surface truncate">
								{h.schueler_name}
								{h.schueler_nachname}
								<span class="text-xs font-normal text-on-surface-variant">· {h.klasse}</span>
							</p>
							<p class="text-xs text-on-surface-variant font-mono">
								Exemplar: {h.exemplar_barcode}
							</p>
						</div>
					</div>
					<div class="text-right shrink-0 ml-4 space-y-0.5">
						<p class="text-xs text-on-surface-variant">
							<span class="font-medium">Von</span>
							{formatDatum(h.ausgeliehen_am)}
						</p>
						<p
							class="text-body-small-emphasized {h.rueckgabe_am ? 'text-success' : 'text-warning'}"
						>
							{h.rueckgabe_am ? `Zurück ${formatDatum(h.rueckgabe_am)}` : 'Noch ausgeliehen'}
						</p>
					</div>
				</li>
			{/each}
		</ul>
	</div>
{/if}
