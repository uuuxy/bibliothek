<!-- @component LmfPlanTabelle — der Plan in der Form, die das Kollegium kennt: je Art
     ein Block (Büchertausch vor den Sommerferien, Bücherausgabe nach den Sommerferien),
     darunter der eine Satz, was dort geschieht, darin Wochentag, Datum, Stunde, Klassen,
     Besonderheiten (Vermerk, wie die Bibliothek ihn geschrieben hat). Lesend — der Portal-Reiter
     und jede Stelle, die den fertigen Plan zeigt. Bearbeitet wird er im Planer; Spaltenbreiten
     und Linien sind dieselben wie dort, eine Linie steht nur zwischen zwei Tagen.
     `heute` (JJJJ-MM-TT) hebt die Zeilen dieses Tages hervor, damit man sofort sieht, welche
     Klassen heute dran sind (Tabelle: aria-current="date"). `aktion` steht rechts in der Zeile
     der ersten Überschrift (im Portal „Als PDF"). -->
<script>
	import Tabelle from '../ui/Tabelle.svelte';
	import {
		ARTEN,
		artErklaerung,
		artBeschriftung,
		datumKurz,
		stundeText,
		wochentag
	} from './lmfplanDienst.js';
	import { tagGehtWeiter } from './lmfplanZeilen.js';

	/** @type {{ termine: import('./lmfplanDienst.js').LmfTermin[], eingangsjahrgaenge?: number[], heute?: string, aktion?: import('svelte').Snippet }} */
	let { termine, eingangsjahrgaenge = [], heute = '', aktion } = $props();

	const bloecke = $derived(
		ARTEN.map((a) => ({
			art: a.wert,
			label: a.label,
			zeilen: termine.filter((t) => t.art === a.wert)
		})).filter((b) => b.zeilen.length > 0)
	);
</script>

{#each bloecke as block, i (block.art)}
	<section class="mt-6" aria-label={block.label}>
		<div class="flex flex-wrap items-center justify-between gap-3 pl-4">
			<h2 class="text-base font-medium text-on-surface">{artBeschriftung(block.art)}</h2>
			{#if i === 0}{@render aktion?.()}{/if}
		</div>
		<p class="px-4 pb-2 text-sm text-on-surface-variant max-w-3xl">
			{artErklaerung(block.art, eingangsjahrgaenge)}
		</p>
		<div class="overflow-x-auto">
			<Tabelle beschriftung="Termine des Lernmittelplans">
				<thead>
					<tr>
						<th class="w-px whitespace-nowrap">Wochentag</th>
						<th class="w-px whitespace-nowrap">Datum</th>
						<th class="w-px whitespace-nowrap">Stunde</th>
						<th class="w-48">Klassen</th>
						<th>Besonderheiten</th>
					</tr>
				</thead>
				<tbody>
					{#each block.zeilen as t, j (t.id)}
						<tr
							aria-current={heute !== '' && t.datum === heute ? 'date' : undefined}
							data-gruppe-weiter={tagGehtWeiter(block.zeilen, j) || undefined}
						>
							<td class="whitespace-nowrap">{wochentag(t.datum)}</td>
							<td class="tabular-nums whitespace-nowrap">{datumKurz(t.datum)}</td>
							<td class="whitespace-nowrap">{stundeText(t.stunde)}</td>
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
