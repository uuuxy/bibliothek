<!--
  @component
  Papierkorb

  Der Papierkorb (weichgelöschte Leser): wiederherstellen, endgültig löschen.

  Zustand und Fehlerausgänge liegen in papierkorbListe.svelte.js: Ein Fehlschlag wird
  gezeigt, und an einer anonymisierten Zeile steht kein Wiederherstellen-Knopf.
-->
<script>
	import { onMount } from 'svelte';
	import Tabelle from '../ui/Tabelle.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import Icon from '../ui/Icon.svelte';
	import PapierkorbLoeschenDialog from './PapierkorbLoeschenDialog.svelte';
	import { erzeugePapierkorb, istAnonymisiert } from './papierkorbListe.svelte.js';
	import { istKollegium, leserArtText } from './leserArt.js';

	let { onRestoreSuccess = () => {}, darfEndgueltigLoeschen = false } = $props();

	// In eine Closure gewickelt: `onRestoreSuccess` ist eine Prop und darf nicht bei der
	// Erzeugung eingefroren werden (state_referenced_locally) — der Elternbildschirm darf
	// sie austauschen.
	const papierkorb = erzeugePapierkorb(() => onRestoreSuccess());
	/** @type {any} Schüler, für den die Endgültig-löschen-Rückfrage offen ist */
	let loeschKandidat = $state(null);

	/** Der Elternbildschirm lädt nach, wenn er selbst etwas gelöscht hat. */
	export async function ladeGeloeschte() {
		await papierkorb.laden();
	}

	async function loescheEndgueltig() {
		if (!loeschKandidat) return;
		const id = loeschKandidat.id;
		await papierkorb.endgueltigLoeschen(id);
		// Nach „Abbrechen" kann schon die Rückfrage zu einem anderen Leser offen stehen.
		if (loeschKandidat?.id === id) loeschKandidat = null;
	}

	onMount(() => {
		papierkorb.laden();
	});
</script>

<div class="w-full border-l-4 border-l-error">
	<div class="px-6 py-4 border-b border-outline-variant flex items-center justify-between">
		<h3 class="text-title-medium text-error flex items-center gap-2">
			<Icon name="delete" />
			Gelöschte Leser im Papierkorb
		</h3>
	</div>

	{#if papierkorb.laedt}
		<div class="py-16 flex justify-center items-center">
			<Ladekreis size="lg" verzoegert />
		</div>
	{:else if papierkorb.ladefehler}
		<!-- Ein gescheiterter Abruf ist KEIN leerer Papierkorb: „leer" wäre hier eine
		     falsche Auskunft über gelöschte Schülerdaten. -->
		<div class="py-16 flex flex-col items-center justify-center space-y-2 px-6 text-center">
			<Icon name="remove_moderator" groesse={40} class="text-error" />
			<span role="alert" class="text-body-medium-emphasized text-error"
				>{papierkorb.ladefehler}</span
			>
			<button
				onclick={() => papierkorb.laden()}
				class="text-body-medium-emphasized text-primary underline cursor-pointer"
			>
				Erneut versuchen
			</button>
		</div>
	{:else if papierkorb.liste.length === 0}
		<div class="py-16 text-center text-base text-on-surface-variant">Der Papierkorb ist leer</div>
	{:else}
		<!-- Der Server zeigt die jüngsten Löschungen bis zu seiner Grenze (Kopf X-Grenze); eine Suche
		     gibt es hier nicht, der Satz sagt deshalb nur, dass Ältere fehlen, wie im Logbuch. -->
		{#if papierkorb.gekappt}
			<p class="px-6 pt-4 text-sm text-on-surface-variant">
				Die {papierkorb.liste.length} zuletzt gelöschten Leser. Ältere Löschungen sind hier nicht sichtbar.
			</p>
		{/if}
		<div class="overflow-x-auto w-full text-left">
			<Tabelle beschriftung="Papierkorb">
				<thead class="text-title-small">
					<tr>
						<th>Name</th>
						<th class="w-24">Klasse</th>
						<th class="w-44">Gelöscht am</th>
						<th class="w-44 text-right">Aktion</th>
					</tr>
				</thead>
				<tbody>
					{#each papierkorb.liste as s, _i (_i)}
						<tr>
							<td class="text-body-medium-emphasized">
								{s.vorname}
								{s.nachname}
								<!-- Wie in der Leserdatei: Ein Kollege hat oft keinen Ausweis und nie eine
								     Klasse; statt der Klasse steht seine Art wie in der Buchakte. -->
								<div
									class="text-sm text-on-surface-variant font-normal mt-0.5 {s.barcode_id
										? 'font-mono'
										: 'italic'}"
								>
									{s.barcode_id || 'ohne Ausweis'}
								</div>
							</td>
							<td class="font-medium">
								{#if istKollegium(s)}
									{leserArtText(s.art)}
								{:else}
									{s.klasse ? `Kl. ${s.klasse}` : 'Klasse unbekannt'}
								{/if}
							</td>
							<td>
								{new Date(s.deleted_at).toLocaleString('de-DE', {
									day: '2-digit',
									month: '2-digit',
									year: 'numeric',
									hour: '2-digit',
									minute: '2-digit'
								})}
							</td>
							<td class="text-right">
								<div class="inline-flex items-center gap-4">
									{#if istAnonymisiert(s)}
										<!-- Kein Wiederherstellen-Knopf: Der Server antwortet hier mit 409,
										     und ein Knopf, der nur scheitern kann, ist keine Aktion,
										     sondern eine Falle. -->
										<span
											class="inline-flex items-center gap-1.5 rounded-lg bg-surface-container px-2 py-1 text-label-medium text-on-surface-variant"
											data-tip="Der nächtliche DSGVO-Lauf hat Name, Adresse und Geburtsdatum getilgt. Diese Zeile lässt sich nicht mehr wiederherstellen und wird nach 180 Tagen im Papierkorb endgültig gelöscht."
										>
											<Icon name="remove_moderator" />
											anonymisiert (DSGVO)
											<span class="sr-only"
												>Der nächtliche DSGVO-Lauf hat Name, Adresse und Geburtsdatum getilgt. Diese
												Zeile lässt sich nicht mehr wiederherstellen und wird nach 180 Tagen im
												Papierkorb endgültig gelöscht.</span
											>
										</span>
									{:else}
										<!-- Gesperrt mit Ladekreis, solange die Anfrage läuft, wie „Verlängern" in der
										     Leserakte: Ein zweiter Klick schickte sonst eine zweite. -->
										<button
											onclick={() => papierkorb.wiederherstellen(s.id)}
											disabled={papierkorb.stelltHer(s.id)}
											data-tip="Wiederherstellen"
											aria-label="Wiederherstellen"
											class="icon-btn bg-success-container text-on-success-container disabled:cursor-default"
										>
											{#if papierkorb.stelltHer(s.id)}
												<Ladekreis size="sm" farbe="aktuell" />
											{:else}
												<Icon name="undo" />
											{/if}
										</button>
									{/if}
									{#if darfEndgueltigLoeschen}
										<button
											onclick={() => (loeschKandidat = s)}
											data-tip="Endgültig löschen"
											aria-label="Endgültig löschen"
											class="icon-btn bg-error-container text-on-error-container"
										>
											<Icon name="delete" />
										</button>
									{/if}
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</Tabelle>
		</div>
	{/if}
</div>

<PapierkorbLoeschenDialog
	open={loeschKandidat !== null}
	name={loeschKandidat ? `${loeschKandidat.vorname} ${loeschKandidat.nachname}` : ''}
	laeuft={papierkorb.loeschtGerade}
	onConfirm={loescheEndgueltig}
	onClose={() => (loeschKandidat = null)}
/>
