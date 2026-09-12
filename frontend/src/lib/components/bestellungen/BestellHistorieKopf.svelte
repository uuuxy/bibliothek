<!-- @component BestellHistorieKopf — Überschrift, Kennzahlen und der Topf-Filter der
     Bestellhistorie.

     Herausgelöst am 12.09.2026 beim Einbau der Aufteilung nach Mittelherkunft (#596):
     Mit ihr und dem Filter überschritt BestellHistorie.svelte die 200-Zeilen-Regel aus
     ARCHITECTURE.md. Reine Verschiebung — dieselben Zeilen, dieselbe Reihenfolge. -->
<script>
	import { orderStore } from '../../stores/orderStore.svelte.js';
	import Select from '../ui/Select.svelte';
	import { Clock } from '@lucide/svelte';

	/**
	 * @type {{
	 *   offeneBestaetigungen: number,
	 *   gesamtsumme: number,
	 *   gesamtExemplare: number,
	 *   aufteilung: any[],
	 *   zeigeKennzahlen: boolean,
	 *   mittel: string,
	 *   mittelFilter: Array<{ value: string, label: string }>,
	 *   euro: (n: number) => string,
	 *   topfLabel: (wert: string) => string,
	 *   onFilterWechsel: () => void
	 * }}
	 */
	let {
		offeneBestaetigungen,
		gesamtsumme,
		gesamtExemplare,
		aufteilung,
		zeigeKennzahlen,
		mittel = $bindable(''),
		mittelFilter,
		euro,
		topfLabel,
		onFilterWechsel
	} = $props();
</script>

<div class="flex items-center justify-between border-b border-slate-200 pb-4">
	<div>
		<h2 class="text-base font-bold text-slate-800">Bestellhistorie</h2>
		<p class="text-sm text-slate-500 mt-0.5">
			Alle aufgegebenen Bestellungen — automatisch erfasst beim Bestellen
		</p>
		<!-- Nur wenn wirklich etwas aussteht. „Alles bestätigt" jeden Tag zu lesen, wäre
		     dieselbe Zeile ohne Nachricht — auffallen soll die Abweichung. Wer den Satz
		     sieht, weiß ohne Scrollen, dass in der Statusspalte etwas auf ihn wartet. -->
		{#if offeneBestaetigungen > 0}
			<p class="mt-2 flex items-center gap-1.5 text-sm font-medium text-amber-700">
				<Clock size={15} aria-hidden="true" />
				{offeneBestaetigungen === 1
					? '1 Bestellung wartet noch auf die Bestätigung des Händlers'
					: `${offeneBestaetigungen} Bestellungen warten noch auf die Bestätigung des Händlers`}
			</p>
		{/if}
	</div>
	<!-- Ohne Preiserfassung ist "Gesamtausgaben 0,00 €" keine Auskunft, sondern eine
	     falsche: Die Schule hat ausgegeben, nur steht es nirgends. Dann lieber die Zahl
	     nennen, die stimmt. -->
	{#if zeigeKennzahlen}
		<div class="text-right">
			{#if orderStore.preiseErfassen}
				<div class="text-xs text-slate-400 font-semibold">Gesamtausgaben</div>
				<div class="text-2xl font-black text-slate-800">{euro(gesamtsumme)}</div>
			{:else}
				<div class="text-xs text-slate-400 font-semibold">Bestellte Exemplare</div>
				<div class="text-2xl font-black text-slate-800">{gesamtExemplare}</div>
			{/if}
			<!-- Woraus die Zahl darüber besteht: Für das Schulamt zählt der
				     Landes-Anteil, für den Schulträger seiner (Migration 109). -->
			{#if aufteilung.length > 1}
				<div class="mt-1 space-y-0.5 text-xs text-on-surface-variant">
					{#each aufteilung as t (t.mittel)}
						<div>
							{topfLabel(t.mittel)}:
							<span class="font-semibold">
								{orderStore.preiseErfassen ? euro(t.gesamtbetrag) : t.gesamt_exemplare}
							</span>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</div>

<!-- Der Filter steht über der Liste und lädt neu: Gefiltert wird am Server, weil
	     die Liste gedeckelt ist. -->
<div class="flex items-center gap-3">
	<label class="text-sm font-medium text-on-surface" for="historie-mittel">Mittelherkunft</label>
	<div class="w-72">
		<Select
			id="historie-mittel"
			bind:value={mittel}
			options={mittelFilter}
			onchange={onFilterWechsel}
		/>
	</div>
</div>
