<!--
  @component
  DeletedStudentList

  Der Papierkorb (weichgelöschte Schüler): wiederherstellen, endgültig löschen.

  Zustand und Fehlerausgänge liegen in papierkorbListe.svelte.js — dort steht auch,
  warum: Bis zum Rasterdurchgang am 06.09.2026 verschluckte diese Ansicht jeden
  Fehlschlag, und der Wiederherstellen-Knopf stand auch an Zeilen, an denen er nur
  scheitern kann.
-->
<script>
	import { onMount } from 'svelte';
	import Tabelle from '../ui/Tabelle.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import { Trash2, Undo2, ShieldOff } from '@lucide/svelte';
	import PapierkorbLoeschenDialog from './PapierkorbLoeschenDialog.svelte';
	import { erzeugePapierkorb, istAnonymisiert } from './papierkorbListe.svelte.js';

	let { onRestoreSuccess = () => {}, darfEndgueltigLoeschen = false } = $props();

	// In eine Closure gewickelt: `onRestoreSuccess` ist eine Prop und darf nicht bei der
	// Erzeugung eingefroren werden (state_referenced_locally) — der Elternbildschirm darf
	// sie austauschen.
	const papierkorb = erzeugePapierkorb(() => onRestoreSuccess());
	/** @type {any} Schüler, für den die Endgültig-löschen-Rückfrage offen ist */
	let loeschKandidat = $state(null);

	/** Der Elternbildschirm lädt nach, wenn er selbst etwas gelöscht hat. */
	export async function loadDeletedStudents() {
		await papierkorb.laden();
	}

	async function purgeStudent() {
		if (!loeschKandidat) return;
		const id = loeschKandidat.id;
		await papierkorb.endgueltigLoeschen(id);
		loeschKandidat = null;
	}

	onMount(() => {
		papierkorb.laden();
	});
</script>

<div class="w-full border-l-4 border-l-rose-400">
	<div class="px-6 py-4 border-b border-slate-200 flex items-center justify-between">
		<h3 class="text-base font-bold text-rose-800 flex items-center gap-2">
			<Trash2 class="h-5 w-5" aria-hidden="true" />
			Gelöschte Schüler (Papierkorb)
		</h3>
	</div>

	{#if papierkorb.laedt}
		<div class="py-16 flex justify-center items-center">
			<Ladekreis size="lg" />
		</div>
	{:else if papierkorb.ladefehler}
		<!-- Ein gescheiterter Abruf ist KEIN leerer Papierkorb: „leer" wäre hier eine
		     falsche Auskunft über gelöschte Schülerdaten. -->
		<div class="py-16 flex flex-col items-center justify-center space-y-2 px-6 text-center">
			<ShieldOff class="h-10 w-10 text-error" aria-hidden="true" />
			<span class="text-sm font-semibold text-error">{papierkorb.ladefehler}</span>
			<button
				onclick={() => papierkorb.laden()}
				class="text-sm font-semibold text-primary underline cursor-pointer"
			>
				Erneut versuchen
			</button>
		</div>
	{:else if papierkorb.liste.length === 0}
		<div class="py-16 flex flex-col items-center justify-center text-slate-400 space-y-2">
			<Trash2 class="h-10 w-10 text-slate-300" aria-hidden="true" />
			<span class="text-xs font-semibold">Der Papierkorb ist leer.</span>
		</div>
	{:else}
		<div class="overflow-x-auto w-full text-left">
			<Tabelle>
				<thead class="font-semibold">
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
							<td class="font-semibold">
								{s.vorname}
								{s.nachname}
								<div class="text-sm font-mono text-slate-400 font-normal mt-0.5">
									{s.barcode_id}
								</div>
							</td>
							<td class="font-medium">
								Kl. {s.klasse || 'N/A'}
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
								<div class="inline-flex items-center gap-2">
									{#if istAnonymisiert(s)}
										<!-- Kein Wiederherstellen-Knopf: Der Server antwortet hier mit 409,
										     und ein Knopf, der nur scheitern kann, ist keine Aktion,
										     sondern eine Falle. -->
										<span
											class="inline-flex items-center gap-1.5 rounded-lg bg-surface-container px-2 py-1 text-xs font-semibold text-on-surface-variant"
											title="Nach 180 Tagen im Papierkorb tilgt der nächtliche DSGVO-Lauf Name, Adresse und Geburtsdatum. Diese Zeile lässt sich nicht mehr wiederherstellen."
										>
											<ShieldOff class="h-4 w-4" aria-hidden="true" />
											anonymisiert (DSGVO)
										</span>
									{:else}
										<button
											onclick={() => papierkorb.wiederherstellen(s.id)}
											title="Wiederherstellen"
											aria-label="Wiederherstellen"
											class="inline-flex items-center justify-center w-8 h-8 rounded-lg bg-emerald-100 text-emerald-700 hover:bg-emerald-200 transition-colors shadow-sm cursor-pointer"
										>
											<Undo2 class="h-4.5 w-4.5" aria-hidden="true" />
										</button>
									{/if}
									{#if darfEndgueltigLoeschen}
										<button
											onclick={() => (loeschKandidat = s)}
											title="Endgültig löschen"
											aria-label="Endgültig löschen"
											class="inline-flex items-center justify-center w-8 h-8 rounded-lg bg-error-container text-on-error-container hover:opacity-80 transition-opacity shadow-sm cursor-pointer"
										>
											<Trash2 class="h-4.5 w-4.5" aria-hidden="true" />
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
	onConfirm={purgeStudent}
	onClose={() => (loeschKandidat = null)}
/>
