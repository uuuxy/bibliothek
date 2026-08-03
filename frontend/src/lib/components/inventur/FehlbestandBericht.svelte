<!-- @component FehlbestandBericht — welche Exemplare eine Inventur als Verlust gebucht hat.

     Der Abschluss meldete bisher nur eine Zahl: „47 Bücher wurden als verloren markiert."
     Damit kann niemand ins Regal gehen und nachsehen, ob eines davon nur falsch
     einsortiert war, und der Schule sagen, was fehlt, auch nicht. Rekonstruieren liess
     sich die Liste danach nicht — durch die Aussonderung fallen die Exemplare aus der
     Scope-Bedingung, nach der gerechnet wird.

     Sortiert kommt sie nach Signatur und Titel vom Server: Das ist die Reihenfolge, in
     der man mit dem Zettel durchs Regal läuft. Nach Barcode sortiert müsste man kreuz und
     quer gehen.

     Der Bericht bleibt stehen, bis er ausdrücklich geschlossen wird — auch über das
     Zurücksetzen der Inventur hinweg. Sonst wäre er im selben Moment wieder weg, in dem
     er entsteht. -->
<script>
	import { Printer, X, PackageSearch } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';

	/** @type {{ eintraege: any[], label?: string, onSchliessen: () => void }} */
	let { eintraege, label = '', onSchliessen } = $props();

	function drucken() {
		window.print();
	}
</script>

<section
	class="w-full space-y-5 rounded-xl border border-amber-200 bg-amber-50/50 p-5 print:border-0 print:bg-white print:p-0"
>
	<div class="flex flex-wrap items-start justify-between gap-4">
		<div class="flex items-start gap-3">
			<PackageSearch class="mt-0.5 h-5 w-5 shrink-0 text-amber-700" aria-hidden="true" />
			<div>
				<h2 class="text-base font-bold text-slate-800">
					Fehlbestand{label ? ` — ${label}` : ''}
				</h2>
				<p class="mt-0.5 text-sm text-slate-600">
					{eintraege.length}
					{eintraege.length === 1 ? 'Exemplar wurde' : 'Exemplare wurden'} als Verlust gebucht. Die
					Liste ist nach Signatur sortiert — in der Reihenfolge lässt sich das Regal absuchen.
				</p>
			</div>
		</div>
		<div class="flex items-center gap-2 no-print">
			<Button variant="secondary" size="sm" onclick={drucken} data-tip="Liste zum Nachsuchen ausdrucken">
				<Printer class="h-4 w-4" aria-hidden="true" />
				Drucken
			</Button>
			<!-- Eindeutiger Name: „Schließen" allein gibt es auf dieser Ansicht mehrfach —
			     für den Screenreader wie für die Bedienung ist dann nicht klar, was zugeht. -->
			<Button
				variant="ghost"
				size="sm"
				onclick={onSchliessen}
				aria-label="Fehlbestandsbericht schließen"
				data-tip="Bericht schließen"
			>
				<X class="h-4 w-4" aria-hidden="true" />
				Schließen
			</Button>
		</div>
	</div>

	{#if eintraege.length === 0}
		<p class="py-6 text-center text-sm text-slate-500">
			Kein Fehlbestand — jedes erwartete Exemplar wurde erfasst.
		</p>
	{:else}
		<div class="overflow-x-auto rounded-xl border border-slate-200 bg-white">
			<table class="w-full border-collapse text-sm">
				<thead>
					<tr class="border-b border-slate-200 bg-slate-50/60 text-xs font-semibold text-slate-400">
						<th class="px-3 py-2 text-left font-semibold">Signatur</th>
						<th class="px-3 py-2 text-left font-semibold">Titel</th>
						<th class="px-3 py-2 text-left font-semibold">Barcode</th>
						<!-- Eine Spalte zum Abhaken. Der Zettel wandert mit ins Regal, und was
						     sich anfindet, wird dort angestrichen. -->
						<th class="w-16 px-3 py-2 text-center font-semibold">Gefunden</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100">
					{#each eintraege as e (e.barcode_id)}
						<tr>
							<td class="px-3 py-2 whitespace-nowrap text-slate-600">{e.signatur || '—'}</td>
							<td class="max-w-0 px-3 py-2">
								<span class="block truncate font-semibold text-slate-800">{e.titel}</span>
								{#if e.autor}
									<span class="block truncate text-xs text-slate-400">{e.autor}</span>
								{/if}
							</td>
							<td class="px-3 py-2 font-mono text-xs whitespace-nowrap text-slate-500"
								>{e.barcode_id}</td
							>
							<td class="px-3 py-2 text-center text-slate-300">☐</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</section>
