<!-- @component FehlbestandBericht — welche Exemplare eine Inventur als Verlust gebucht hat.

     Eine Zahl allein reicht nicht: Mit der Liste geht jemand ins Regal und sieht nach, ob
     ein Buch nur falsch einsortiert war, und der Schule lässt sich sagen, was fehlt. Später
     ist sie nicht mehr zu errechnen, weil die ausgesonderten Exemplare aus dem Umfang der
     Inventur fallen.

     Das Absuchen hat zwei Ausgänge, und beide stehen hier: Ein wiedergefundenes Buch kommt
     über „Gefunden" zurück in Umlauf, ein endgültig fehlendes über den Lösch-Knopf aus dem
     Katalog.

     Sortiert kommt die Liste nach Signatur und Titel vom Server, in der Reihenfolge des
     Regals. Der Bericht bleibt stehen, bis er ausdrücklich geschlossen wird, auch über das
     Zurücksetzen der Inventur hinweg; sonst wäre er weg, sobald er entsteht. -->
<script>
	import { Printer, X, PackageSearch, Trash } from '@lucide/svelte';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import { baueFehlbestandDruckHtml } from '../../utils/fehlbestandDruck.js';
	import { druckeDokument, FENSTER_BLOCKIERT } from '../../utils/listenDruck.js';
	import Tabelle from '../ui/Tabelle.svelte';
	import Button from '../ui/Button.svelte';
	import Kaestchen from '../ui/Kaestchen.svelte';
	import StatusChip from '../ui/StatusChip.svelte';
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

	// Auf das Blatt kommt, was noch zu suchen ist; Geklärtes steht dort nur als Zahl.
	function drucken() {
		const html = baueFehlbestandDruckHtml(offene, { label, gebucht: eintraege.length });
		if (!druckeDokument(html)) toastStore.addToast(FENSTER_BLOCKIERT, 'warning');
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

<!-- Eine Karte mit Rahmen und ohne Füllung (M3 Cards, outlined): Der Bericht ist ein Gegenstand
     für sich, mit eigenen Aktionen. Die Tabelle steht ohne zweiten Rahmen darin. -->
<section
	class="w-full space-y-5 rounded-xl border border-outline-variant p-5 print:border-0 print:p-0"
>
	<div class="flex flex-wrap items-start justify-between gap-4">
		<div class="flex items-start gap-3">
			<PackageSearch class="mt-0.5 h-5 w-5 shrink-0 text-warning" aria-hidden="true" />
			<div>
				<h2 class="text-base font-bold text-on-surface">
					Fehlbestand{label ? ` — ${label}` : ''}
				</h2>
				<p class="mt-0.5 text-sm text-on-surface-variant">
					{eintraege.length}
					{eintraege.length === 1 ? 'Exemplar wurde' : 'Exemplare wurden'} als Verlust gebucht. Die Liste
					ist nach Signatur sortiert — in der Reihenfolge lässt sich das Regal absuchen.
					{#if offene.length !== eintraege.length}
						<span class="font-medium text-success">
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
					<Trash class="h-4 w-4" aria-hidden="true" />
					{offene.length} endgültig löschen
				</Button>
			{/if}
			<Button variant="secondary" size="sm" onclick={drucken} disabled={offene.length === 0}>
				<Printer class="h-4 w-4" aria-hidden="true" />
				Liste drucken
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
		<p class="py-6 text-center text-sm text-on-surface-variant">
			Kein Fehlbestand — jedes erwartete Exemplar wurde erfasst.
		</p>
	{:else}
		<div class="overflow-x-auto">
			<Tabelle beschriftung="Fehlbestand">
				<thead>
					<tr>
						<th>Signatur</th>
						<th>Titel</th>
						<th>Barcode</th>
						<!-- Zum Abhaken beim Regal-Absuchen: mit Tastatur/Maus am Bildschirm,
						     oder auf dem Ausdruck mit dem Stift — beides bleibt möglich. -->
						<th class="w-24 text-center no-print">Gefunden</th>
					</tr>
				</thead>
				<tbody>
					{#each eintraege as e (e.barcode_id)}
						{@const istGefunden = Boolean(e.gefunden_am)}
						<tr>
							<td class="whitespace-nowrap">{e.signatur || '—'}</td>
							<!-- w-full: Die Zelle nimmt den Rest der Breite, und ein langer Titel läuft um.
							     Gekürzt stünde der Rest nur in einer Sprechblase, die ohne Maus nicht erscheint. -->
							<td class="w-full">
								<span
									class="block font-semibold {istGefunden
										? 'text-on-surface-variant line-through'
										: 'text-on-surface'}">{e.titel}</span
								>
								{#if e.autor}
									<span class="block text-sm text-on-surface-variant">{e.autor}</span>
								{/if}
							</td>
							<td class="font-mono whitespace-nowrap">{e.barcode_id}</td>
							<td class="text-center no-print">
								{#if istGefunden}
									<StatusChip ton="erfolg" text="Gefunden" />
								{:else if e.exemplar_id}
									<Kaestchen
										checked={false}
										disabled={gefundenLaeuft === e.exemplar_id}
										onclick={() => gefunden(e.exemplar_id)}
										aria-label="{e.titel} als gefunden markieren und zurück in Umlauf bringen"
									/>
								{:else}
									<span class="text-sm text-on-surface-variant" title="Bereits endgültig gelöscht"
										>—</span
									>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</Tabelle>
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
