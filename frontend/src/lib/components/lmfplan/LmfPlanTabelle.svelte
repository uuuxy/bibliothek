<!-- @component LmfPlanTabelle — der Plan in der Form, die das Kollegium kennt: je Art
     ein Block (Bücherrückgabe, Bücherausgabe), darin Wochentag, Datum, Stunde, Klassen,
     Besonderheiten. Verwaltungsseite und Portal-Reiter zeigen DIESELBE Tabelle; nur die
     Verwaltung bekommt die Aktionsspalte. -->
<script>
	import { Pencil, Trash2 } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import { ARTEN, artLabel, datumKurz, stundeText, wochentag } from '../../lmfplanDienst.js';

	/** @type {{ termine: import('../../lmfplanDienst.js').LmfTermin[], bearbeitbar?: boolean, onBearbeiten?: (t: any) => void, onLoeschen?: (t: any) => void }} */
	let { termine, bearbeitbar = false, onBearbeiten = () => {}, onLoeschen = () => {} } = $props();

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
						{#if bearbeitbar}<th class="py-2 px-4 text-right">Aktionen</th>{/if}
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
							{#if bearbeitbar}
								<td class="py-1 px-4 text-right whitespace-nowrap">
									<Button
										variant="ghost"
										size="sm"
										onclick={() => onBearbeiten(t)}
										title="Termin bearbeiten"
									>
										<Pencil class="h-4 w-4" aria-hidden="true" />
										Bearbeiten
									</Button>
									<Button
										variant="ghost"
										size="sm"
										onclick={() => onLoeschen(t)}
										title="Termin löschen"
									>
										<Trash2 class="h-4 w-4" aria-hidden="true" />
										Löschen
									</Button>
								</td>
							{/if}
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</section>
{/each}
