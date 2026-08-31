<!-- @component Die Bestellhistorie als Tabelle — eine Zeile je Bestellung.

     Eine Kopfzeile für alle Bestellungen statt vier Beschriftungen je Zeile: Der Inhalt
     ist ein gleichförmiger Datensatz mit festen Spalten, also eine Tabelle. Als Karten
     brauchte dieselbe Information rund das Dreifache an Höhe.

     Die Zeile klappte bis zum 08.08.2026 auf und zeigte dann dieselben Angaben noch
     einmal untereinander — ohne Cover, ohne Exemplarnummern, also ohne Zugewinn. Sie
     FÜHRT jetzt in die Detailansicht: role="button" samt Tastaturbedienung bleibt, das
     Chevron zeigt nach rechts statt nach unten.

     Eigene Datei, seit die Historie den Detail-Zweig traegt: Beides zusammen lag ueber
     der 200-Zeilen-Marke, die in diesem Projekt gilt. -->
<script>
	import { orderStore } from '../../stores/orderStore.svelte.js';
	import { CheckCircle2, Clock, ChevronRight } from '@lucide/svelte';
	import StatusChip from '../ui/StatusChip.svelte';

	/**
	 * @type {{
	 *   bestellungen: any[],
	 *   euro: (n: number) => string,
	 *   datum: (iso: string) => string,
	 *   kurzdatum: (iso: string) => string,
	 *   onOeffnen: (id: string) => void
	 * }}
	 */
	let { bestellungen, euro, datum, kurzdatum, onOeffnen } = $props();
</script>

<div class="overflow-x-auto rounded-xl border border-slate-200 bg-white shadow-xs">
	<table class="w-full border-collapse text-sm">
		<thead>
			<tr class="border-b border-slate-200 bg-slate-50/60 text-xs font-semibold text-slate-400">
				<th class="px-3 py-2 text-left font-semibold">Datum</th>
				<th class="px-3 py-2 text-left font-semibold">Lieferant</th>
				<!-- Eigene Spalte, weil der Status vorher IN der Lieferantenzelle stand: Die
			     trägt max-w-0 + truncate, und das Chip wurde auf wenige Pixel zerquetscht —
			     die Angabe war da, aber nicht lesbar. -->
				<th class="px-3 py-2 text-left font-semibold">Bestätigung</th>
				<th class="px-3 py-2 text-right font-semibold">Exemplare</th>
				{#if orderStore.preiseErfassen}<th class="px-3 py-2 text-right font-semibold">Betrag</th
					>{/if}
				<th class="w-8 px-3 py-2"><span class="sr-only">Bestellung öffnen</span></th>
			</tr>
		</thead>
		<tbody class="divide-y divide-slate-100">
			{#each bestellungen as b (b.id)}
				<tr
					role="button"
					tabindex="0"
					aria-label="Bestellung vom {datum(b.bestelldatum)} bei {b.lieferant_name} öffnen"
					onclick={() => onOeffnen(b.id)}
					onkeydown={(e) => {
						if (e.key === 'Enter' || e.key === ' ') {
							e.preventDefault();
							onOeffnen(b.id);
						}
					}}
					class="cursor-pointer transition-colors hover:bg-slate-50/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-blue-500"
				>
					<td class="px-3 py-2 font-semibold whitespace-nowrap text-slate-800 tabular-nums">
						{datum(b.bestelldatum)}
					</td>
					<td class="max-w-0 px-3 py-2">
						<span class="block truncate font-semibold text-slate-800">{b.lieferant_name}</span>
						<span class="block truncate text-sm text-slate-400">
							{b.kundennummer ? 'Kd.-Nr. ' + b.kundennummer : b.lieferant_email}
						</span>
					</td>
					<!-- Nur Lieferanten mit dem externen Schritt tragen hier etwas. Ein „—" in
				     jeder anderen Zeile wäre Rauschen: Auffallen soll die Abweichung. -->
					<td class="px-3 py-2 whitespace-nowrap">
						{#if b.mit_bestaetigung && b.bestaetigt_am}
							<StatusChip
								ton="erfolg"
								icon={CheckCircle2}
								text="Bestätigt"
								detail={kurzdatum(b.bestaetigt_am)}
								tip={b.bestaetigt_durch === 'lieferant'
									? 'Der Lieferant hat die Bestellung über den Link bestätigt'
									: 'Bestätigung wurde in der Bibliothek von Hand nachgetragen'}
							/>
						{:else if b.mit_bestaetigung}
							<StatusChip
								ton="warten"
								icon={Clock}
								text="Wartet auf Händler"
								tip="Der Lieferant hat die Bestellung noch nicht über den Link bestätigt"
							/>
						{/if}
					</td>
					<td class="px-3 py-2 text-right whitespace-nowrap text-slate-700 tabular-nums">
						{b.anzahl_exemplare}
					</td>
					{#if orderStore.preiseErfassen}
						<td
							class="px-3 py-2 text-right font-bold whitespace-nowrap text-slate-900 tabular-nums"
						>
							{euro(b.gesamtbetrag)}
						</td>
					{/if}
					<!-- Chevron nach rechts: Die Zeile führt weiter, sie klappt nicht mehr auf. -->
					<td class="px-3 py-2 text-right">
						<ChevronRight class="inline-block h-4 w-4 text-slate-400" aria-hidden="true" />
					</td>
				</tr>
			{/each}
		</tbody>
	</table>
</div>
