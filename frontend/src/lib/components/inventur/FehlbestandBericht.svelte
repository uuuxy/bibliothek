<!-- @component FehlbestandBericht — welche Exemplare eine Inventur als Verlust gebucht hat.

     Der Abschluss meldete bisher nur eine Zahl: „47 Bücher wurden als verloren markiert."
     Damit kann niemand ins Regal gehen und nachsehen, ob eines davon nur falsch
     einsortiert war, und der Schule sagen, was fehlt, auch nicht. Rekonstruieren liess
     sich die Liste danach nicht — durch die Aussonderung fallen die Exemplare aus der
     Scope-Bedingung, nach der gerechnet wird.

     Zwei Handlungen, nicht nur Anzeige (05.08.2026, Peter: „ich kann nicht weiter damit
     machen!"): Ein wiedergefundenes Buch kommt über "Gefunden" zurück in Umlauf, ein
     endgültig fehlendes über den Lösch-Knopf ganz aus dem Katalog — beide Ausgänge, die
     das Regal-Absuchen tatsächlich hat.

     Sortiert kommt sie nach Signatur und Titel vom Server: Das ist die Reihenfolge, in
     der man mit dem Zettel durchs Regal läuft. Nach Barcode sortiert müsste man kreuz und
     quer gehen.

     Der Bericht bleibt stehen, bis er ausdrücklich geschlossen wird — auch über das
     Zurücksetzen der Inventur hinweg. Sonst wäre er im selben Moment wieder weg, in dem
     er entsteht. -->
<script>
	import { Printer, X, PackageSearch, Trash2 } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import VerlustLoeschenDialog from './VerlustLoeschenDialog.svelte';

	/**
	 * @type {{
	 *   eintraege: any[], label?: string, onSchliessen: () => void,
	 *   onGefunden: (exemplarId: string) => Promise<void>,
	 *   onEndgueltigLoeschen: (exemplarIds: string[]) => Promise<void>
	 * }}
	 */
	let { eintraege, label = '', onSchliessen, onGefunden, onEndgueltigLoeschen } = $props();

	let loeschDialogOffen = $state(false);
	let loeschtGerade = $state(false);
	/** Exemplar-ID, deren "Gefunden"-Klick gerade unterwegs ist — verhindert Doppelklicks. */
	let gefundenLaeuft = $state('');

	// Offen = noch nicht gefunden UND noch da (exemplar_id vorhanden — sonst wurde es
	// bereits auf einem anderen Weg endgültig gelöscht und es gibt nichts mehr zu tun).
	let offene = $derived(eintraege.filter((e) => !e.gefunden_am && e.exemplar_id));

	function drucken() {
		window.print();
	}

	/** @param {string} exemplarId */
	async function gefunden(exemplarId) {
		gefundenLaeuft = exemplarId;
		try {
			await onGefunden(exemplarId);
		} finally {
			gefundenLaeuft = '';
		}
	}

	async function endgueltigLoeschen() {
		loeschtGerade = true;
		try {
			await onEndgueltigLoeschen(offene.map((e) => e.exemplar_id));
			loeschDialogOffen = false;
		} finally {
			loeschtGerade = false;
		}
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
					{eintraege.length === 1 ? 'Exemplar wurde' : 'Exemplare wurden'} als Verlust gebucht. Die Liste
					ist nach Signatur sortiert — in der Reihenfolge lässt sich das Regal absuchen.
					{#if offene.length !== eintraege.length}
						<span class="font-medium text-emerald-700">
							{eintraege.length - offene.length} bereits geklärt.
						</span>
					{/if}
				</p>
			</div>
		</div>
		<div class="flex items-center gap-2 no-print">
			{#if offene.length > 0}
				<Button
					variant="danger"
					size="sm"
					onclick={() => (loeschDialogOffen = true)}
					data-tip="Weiterhin fehlende Exemplare unwiderruflich aus dem Katalog entfernen"
				>
					<Trash2 class="h-4 w-4" aria-hidden="true" />
					{offene.length} endgültig löschen
				</Button>
			{/if}
			<Button
				variant="secondary"
				size="sm"
				onclick={drucken}
				data-tip="Liste zum Nachsuchen ausdrucken"
			>
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
						<!-- Zum Abhaken beim Regal-Absuchen: mit Tastatur/Maus am Bildschirm,
						     oder auf dem Ausdruck mit dem Stift — beides bleibt möglich. -->
						<th class="w-24 px-3 py-2 text-center font-semibold no-print">Gefunden</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100">
					{#each eintraege as e (e.barcode_id)}
						{@const istGefunden = Boolean(e.gefunden_am)}
						<tr class={istGefunden ? 'bg-emerald-50/40' : ''}>
							<td class="px-3 py-2 whitespace-nowrap text-slate-600">{e.signatur || '—'}</td>
							<td class="max-w-0 px-3 py-2">
								<span
									class="block truncate font-semibold text-slate-800 {istGefunden
										? 'line-through decoration-slate-300'
										: ''}">{e.titel}</span
								>
								{#if e.autor}
									<span class="block truncate text-sm text-slate-400">{e.autor}</span>
								{/if}
							</td>
							<td class="px-3 py-2 font-mono text-sm whitespace-nowrap text-slate-500"
								>{e.barcode_id}</td
							>
							<td class="px-3 py-2 text-center no-print">
								{#if istGefunden}
									<span class="text-sm font-medium text-emerald-700">Gefunden</span>
								{:else if e.exemplar_id}
									<input
										type="checkbox"
										class="h-4 w-4 cursor-pointer rounded border-slate-300 text-emerald-600 focus:ring-emerald-500/20"
										checked={false}
										disabled={gefundenLaeuft === e.exemplar_id}
										onclick={() => gefunden(e.exemplar_id)}
										aria-label="{e.titel} als gefunden markieren und zurück in Umlauf bringen"
									/>
								{:else}
									<span class="text-sm text-slate-400" title="Bereits endgültig gelöscht">—</span>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</section>

<VerlustLoeschenDialog
	open={loeschDialogOffen}
	anzahl={offene.length}
	laeuft={loeschtGerade}
	onConfirm={endgueltigLoeschen}
	onClose={() => (loeschDialogOffen = false)}
/>
