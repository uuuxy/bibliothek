<!-- @component NachbuchMeldungen — was beim Nachbuchen anders lief als beim Scan.

     Schritt C des Offline-Baus (OFFEN.md 2.2, Commit 16). Der Server hält jede Abweichung
     fest (Migration 117); bis zum 16.09.2026 hatte diese Ablage im Browser keinen
     Aufrufer — er schrieb Meldungen, die niemand zu sehen bekam.

     Eine Meldung ist KEIN Fehler des Systems, sondern eine Arbeit für einen Menschen: Das
     Buch lag bei jemand anderem, die Rückgabe kam nach einer neueren Ausleihe, der Ausweis
     liess sich nicht auflösen. Deshalb steht bei jeder Zeile, was der Server getan hat und
     warum — und der Knopf heisst „Erledigt", nicht „Löschen": Quittiert wird, was jemand
     angesehen hat.

     Liegt über dem Band (Ebene „darueber"), weil die Liste aus ihm heraus öffnet. -->
<script>
	import Modal from '../Modal.svelte';
	import Button from './ui/Button.svelte';
	import Tabelle from './ui/Tabelle.svelte';
	import Kaestchen from './ui/Kaestchen.svelte';
	import Ladekreis from './ui/Ladekreis.svelte';
	import { nachbuchMeldungen as m } from '../stores/nachbuchMeldungen.svelte.js';

	/** Die sieben Ergebniswörter der Tür in der Sprache der Theke. */
	const WORT = {
		umgebucht: 'lag bei jemand anderem — dort zurückgenommen, neu ausgeliehen',
		nur_reaktiviert: 'Buch war abgeschrieben und ist wieder im Umlauf',
		nicht_gebucht: 'nicht gebucht',
		veraltet: 'nicht gebucht — es gab schon eine neuere Buchung',
		bereits_ausgeliehen: 'war schon ausgeliehen',
		zurueckgegeben: 'zurückgegeben',
		bereits_gebucht: 'war schon gebucht'
	};

	/** @param {string} iso */
	const zeit = (iso) =>
		iso ? new Date(iso).toLocaleString('de-DE', { dateStyle: 'short', timeStyle: 'short' }) : '';

	/** @param {any} z */
	const person = (z) =>
		[z.ausleiher, z.ausleiher_klasse].filter(Boolean).join(', ') || z.ausweis_text || '';
</script>

<Modal
	open={m.geoeffnet}
	onclose={() => m.schliesse()}
	size="3xl"
	ebene="darueber"
	beschriftetDurch="nachbuch-titel"
>
	{#snippet header()}
		<h3 id="nachbuch-titel" class="text-lg font-bold text-on-surface">
			Meldungen aus dem Nachbuchen
		</h3>
	{/snippet}
	<div class="p-4">
		<p class="mb-3 text-sm text-on-surface-variant">
			Hier steht, was beim Nachbuchen der ohne Netz gescannten Vorgänge anders lief als beim Scan.
			„Erledigt" heisst: angesehen und in Ordnung gebracht.
		</p>

		<div class="mb-3">
			<Kaestchen
				bind:checked={m.auchQuittierte}
				label="Auch erledigte zeigen"
				onchange={() => m.lade()}
			/>
		</div>

		{#if m.fehler}
			<p class="mb-3 text-sm font-semibold text-error">{m.fehler}</p>
		{/if}

		{#if m.laeuft && m.liste.length === 0}
			<Ladekreis />
		{:else if m.liste.length === 0}
			<p class="py-6 text-center text-sm text-on-surface-variant">
				{m.auchQuittierte ? 'Keine Meldungen.' : 'Nichts offen.'}
			</p>
		{:else}
			<Tabelle beschriftung="Meldungen aus dem Nachbuchen">
				<thead>
					<tr>
						<th>Gescannt</th>
						<th>Buch</th>
						<th>Person</th>
						<th>Was geschah</th>
						<th></th>
					</tr>
				</thead>
				<tbody>
					{#each m.liste as z (z.id)}
						<tr>
							<td class="whitespace-nowrap tabular-nums">{zeit(z.gescannt_am)}</td>
							<td>
								<span class="font-mono">{z.barcode}</span>
								{#if z.titel}<span class="block text-on-surface-variant">{z.titel}</span>{/if}
							</td>
							<td>
								{person(z)}
								{#if z.vorbesitzer}
									<span class="block text-on-surface-variant">vorher: {z.vorbesitzer}</span>
								{/if}
							</td>
							<td>
								{WORT[z.ergebnis] ?? z.ergebnis}
								{#if z.grund}<span class="block text-on-surface-variant">{z.grund}</span>{/if}
							</td>
							<td class="text-right">
								{#if z.quittiert_am}
									<span class="text-on-surface-variant">erledigt {zeit(z.quittiert_am)}</span>
								{:else}
									<Button variant="secondary" size="sm" onclick={() => m.quittiere(z.id)}>
										Erledigt
									</Button>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</Tabelle>
		{/if}
	</div>
</Modal>
