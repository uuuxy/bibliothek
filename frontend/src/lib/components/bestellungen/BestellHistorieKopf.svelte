<!-- @component BestellHistorieKopf — Überschrift, Kennzahlen und der Topf-Filter der
     Bestellhistorie.

     Eigene Datei: Mit der Aufteilung nach Mittelherkunft und dem Filter läge
     BestellHistorie.svelte über der 200-Zeilen-Regel aus docs/ARCHITEKTUR.md (5.3). -->
<script>
	import { bestellStore } from '../../stores/bestellStore.svelte.js';
	import FilterChips from '../ui/FilterChips.svelte';
	import { Clock } from '@lucide/svelte';

	/**
	 * @type {{
	 *   offeneBestaetigungen: number,
	 *   standBekannt: boolean,
	 *   gesamtsumme: number,
	 *   gesamtExemplare: number,
	 *   aufteilung: any[],
	 *   zeigeKennzahlen: boolean,
	 *   mittel: string,
	 *   mittelFilter: Array<{ value: string, label: string }>,
	 *   euro: (n: number) => string,
	 *   topfName: (wert: string) => string,
	 *   onFilterWechsel: () => void
	 * }}
	 */
	let {
		offeneBestaetigungen,
		standBekannt,
		gesamtsumme,
		gesamtExemplare,
		aufteilung,
		zeigeKennzahlen,
		mittel = $bindable(''),
		mittelFilter,
		euro,
		topfName,
		onFilterWechsel
	} = $props();

	// Alle Chips aus heißt kein Filter, wie im Bestandsbuch.
	const chips = $derived(
		mittelFilter.filter((o) => o.value !== '').map((o) => ({ wert: o.value, text: o.label }))
	);

	/** @param {string | null} wert */
	function waehle(wert) {
		mittel = wert ?? '';
		onFilterWechsel();
	}
</script>

<div class="flex items-center justify-between border-b border-outline-variant pb-4">
	<div>
		<h2 class="sr-only">Bestellhistorie</h2>
		<p class="text-sm text-on-surface-variant">
			Alle aufgegebenen Bestellungen — automatisch erfasst beim Bestellen
		</p>
		<!-- Nur wenn wirklich etwas aussteht. „Alles bestätigt" jeden Tag zu lesen, wäre
		     dieselbe Zeile ohne Nachricht — auffallen soll die Abweichung. Wer den Satz
		     sieht, weiß ohne Scrollen, dass in der Statusspalte etwas auf ihn wartet.
		     Die Zahl sinkt ohne Zutun, wenn der Händler bestätigt, und wird deshalb vorgelesen. In den
		     Zeilen geht das nicht: Eine Zeile ist ein Knopf, und ein Knopf nennt nur seine Beschriftung.
		     Solange keine Antwort da ist, sagt der Kopf nichts: „Keine" wäre eine Behauptung. -->
		<div aria-live="polite">
			{#if offeneBestaetigungen > 0}
				<p class="mt-2 flex items-center gap-1.5 text-sm font-medium text-warning">
					<Clock size={15} aria-hidden="true" />
					{offeneBestaetigungen === 1
						? '1 Bestellung wartet noch auf die Bestätigung des Händlers'
						: `${offeneBestaetigungen} Bestellungen warten noch auf die Bestätigung des Händlers`}
				</p>
			{:else if standBekannt}
				<span class="sr-only">Keine Bestellung wartet auf die Bestätigung des Händlers</span>
			{/if}
		</div>
	</div>
	<!-- Ohne Preiserfassung ist "Gesamtausgaben 0,00 €" keine Auskunft, sondern eine
	     falsche: Die Schule hat ausgegeben, nur steht es nirgends. Dann lieber die Zahl
	     nennen, die stimmt. -->
	{#if zeigeKennzahlen}
		<div class="text-right">
			{#if bestellStore.preiseErfassen}
				<div class="text-xs text-on-surface-variant font-semibold">Gesamtausgaben</div>
				<div class="text-2xl font-black text-on-surface">{euro(gesamtsumme)}</div>
			{:else}
				<div class="text-xs text-on-surface-variant font-semibold">Bestellte Exemplare</div>
				<div class="text-2xl font-black text-on-surface">{gesamtExemplare}</div>
			{/if}
			<!-- Woraus die Zahl darüber besteht: Für das Schulamt zählt der
				     Landes-Anteil, für den Schulträger seiner (Migration 109). -->
			{#if aufteilung.length > 1}
				<div class="mt-1 space-y-0.5 text-xs text-on-surface-variant">
					{#each aufteilung as t (t.mittel)}
						<div>
							{topfName(t.mittel)}:
							<span class="font-semibold">
								{bestellStore.preiseErfassen ? euro(t.gesamtbetrag) : t.gesamt_exemplare}
							</span>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</div>

<!-- Der Filter steht über der Liste und lädt neu: Gefiltert wird am Server, weil
	     die Liste gedeckelt ist. M3, Chips: „Filter chips use tags or descriptive words to filter
     content". -->
<FilterChips
	etikett="Nach Mittelherkunft filtern"
	optionen={chips}
	wert={mittel || null}
	onwahl={waehle}
/>
