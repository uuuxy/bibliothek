<!-- @component LmfPlanTabelle — der Plan in der Form, die das Kollegium kennt: je Art
     ein Block (Bücherrückgabe, Bücherausgabe), darin Wochentag, Datum, Stunde, Klassen,
     Besonderheiten. Lesend — der Portal-Reiter und jede Stelle, die den fertigen Plan
     zeigt. Bearbeitet wird er im Planer (LmfPlanReihenfolge). -->
<script>
	import { ARTEN, artLabel, datumKurz, stundeText, wochentag } from '../../lmfplanDienst.js';

	/** @type {{ termine: import('../../lmfplanDienst.js').LmfTermin[] }} */
	let { termine } = $props();

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
		<h2 class="text-title-medium font-medium text-on-surface px-4 pb-2">{artLabel(block.art)}</h2>
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
							<td class="py-2 px-4 text-on-surface-variant">{t.vermerk}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</section>
{/each}
