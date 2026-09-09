<!-- @component LmfPlanTabelle — der Plan in der Form, die das Kollegium kennt: je Art
     ein Block (Büchertausch vor den Sommerferien, Bücherausgabe nach den Sommerferien),
     darunter der eine Satz, was dort geschieht, darin Wochentag, Datum, Stunde, Klassen,
     Besonderheiten (Vermerk, wie die Bibliothek ihn geschrieben hat). Lesend — der Portal-Reiter
     und jede Stelle, die den fertigen Plan zeigt. Bearbeitet wird er im Planer. -->
<script>
	import Tabelle from '../ui/Tabelle.svelte';
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
			<Tabelle beschriftung="Termine des Lernmittelplans">
				<thead>
					<tr>
						<th>Wochentag</th>
						<th>Datum</th>
						<th>Stunde</th>
						<th>Klassen</th>
						<th>Besonderheiten</th>
					</tr>
				</thead>
				<tbody>
					{#each block.zeilen as t (t.id)}
						<tr>
							<td>{wochentag(t.datum)}</td>
							<td class="tabular-nums">{datumKurz(t.datum)}</td>
							<td>{stundeText(t.stunde)}</td>
							<td class="font-medium">{t.klassen.join(' / ')}</td>
							<td>
								{t.vermerk}
							</td>
						</tr>
					{/each}
				</tbody>
			</Tabelle>
		</div>
	</section>
{/each}
