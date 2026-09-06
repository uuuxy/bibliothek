<!-- @component LmfPlanTabelle — der Plan in der Form, die das Kollegium kennt: je Art
     ein Block (Büchertausch vor den Sommerferien, Bücherausgabe nach den Sommerferien),
     darunter der eine Satz, was dort geschieht, darin Wochentag, Datum, Stunde, Klassen,
     Besonderheiten („nur Rückgabe" als Chip vor dem Vermerk). Lesend — der Portal-Reiter
     und jede Stelle, die den fertigen Plan zeigt. Bearbeitet wird er im Planer. -->
<script>
	import StatusChip from '../ui/StatusChip.svelte';
	import {
		ARTEN,
		artErklaerung,
		artLabel,
		datumKurz,
		stundeText,
		wochentag
	} from '../../lmfplanDienst.js';

	/** @type {{ termine: import('../../lmfplanDienst.js').LmfTermin[], eingangsjahrgaenge?: number[] }} */
	let { termine, eingangsjahrgaenge = [] } = $props();

	const bloecke = $derived(
		ARTEN.map((a) => ({
			art: a.wert,
			label: a.label,
			zeilen: termine.filter((t) => t.art === a.wert)
		})).filter((b) => b.zeilen.length > 0)
	);
</script>

{#each bloecke as block (block.art)}
	<section class="mt-6" aria-label={block.label}>
		<h2 class="text-title-medium font-medium text-on-surface px-4">{artLabel(block.art)}</h2>
		<p class="px-4 pb-2 text-sm text-on-surface-variant max-w-3xl">
			{artErklaerung(block.art, eingangsjahrgaenge)}
		</p>
		<div class="overflow-x-auto">
			<table class="w-full text-left text-base border-collapse">
				<thead>
					<tr class="border-b border-outline-variant text-on-surface-variant text-sm">
						<th class="py-2 px-4">Wochentag</th>
						<th class="py-2 px-4">Datum</th>
						<th class="py-2 px-4">Stunde</th>
						<th class="py-2 px-4">Klassen</th>
						<th class="py-2 px-4">Besonderheiten</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-outline-variant">
					{#each block.zeilen as t (t.id)}
						<tr class="hover:bg-surface-container-low transition-colors">
							<td class="py-2 px-4 text-on-surface-variant">{wochentag(t.datum)}</td>
							<td class="py-2 px-4 text-on-surface tabular-nums">{datumKurz(t.datum)}</td>
							<td class="py-2 px-4 text-on-surface-variant">{stundeText(t.stunde)}</td>
							<td class="py-2 px-4 font-medium text-on-surface">{t.klassen.join(' / ')}</td>
							<td class="py-2 px-4 text-on-surface-variant">
								<span class="inline-flex items-center gap-2">
									{#if t.nur_rueckgabe}
										<StatusChip text="nur Rückgabe" />
									{/if}
									{t.vermerk}
								</span>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</section>
{/each}
