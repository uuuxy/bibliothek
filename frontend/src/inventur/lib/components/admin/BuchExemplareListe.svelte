<script>
	import { apiFetch, leseApiFehler } from '../../../../lib/apiFetch.js';
	import { loeschenBestaetigen } from '../../../../lib/stores/bestaetigung.svelte.js';
	import { melde } from '$lib/store.svelte.js';
	import { ladeBestandNach } from '../../buch_speichern.js';
	import { onMount } from 'svelte';
	import { SvelteSet } from 'svelte/reactivity';
	import { Trash } from '@lucide/svelte';
	import Ladekreis from '../../../../lib/components/ui/Ladekreis.svelte';
	import StatusChip from '../../../../lib/components/ui/StatusChip.svelte';
	import { exemplarStatus } from '../../../../lib/components/buchakte/exemplarStatus.js';
	import BuchEingabefelderInventar from './BuchEingabefelderInventar.svelte';
	import { fehlertext } from '../../../../lib/utils/fehlertext.js';

	let { formular = $bindable() } = $props();

	// Der Abschnitt „Exemplare": die Zahl im Feld „Aktueller Bestand" und darunter die Liste,
	// die denselben Bestand zeigt. Das Feld legt beim Speichern Exemplare an oder sondert aus;
	// ein neuer Titel hat das Feld, aber noch keine Liste. Ausgesonderte und bestellte
	// Exemplare zählt der Bestand nicht; sie führt die Buchakte.
	/** @type {any[]} */
	let exemplare = $state([]);
	let ausgesondert = $state(0);
	let bestellt = $state(0);
	let laedt = $state(true);
	let fehler = $state('');
	// Exemplare, deren Löschen läuft. Ein zweiter Auftrag vor der Antwort schickte ein zweites
	// DELETE, und nach dem Erfolg des ersten stand „Exemplar nicht gefunden oder schon ausgebucht".
	const loescht = new SvelteSet();

	const nichtImBestand = $derived(
		[ausgesondert && `${ausgesondert} ausgesondert`, bestellt && `${bestellt} bestellt`]
			.filter(Boolean)
			.join(', ')
	);

	onMount(() => {
		ladeExemplare();
	});

	async function ladeExemplare() {
		if (!formular.id) return;
		laedt = true;
		fehler = '';
		try {
			const res = await apiFetch(`/api/buecher/titel/${formular.id}/exemplare`, {
				credentials: 'include'
			});
			if (!res.ok) throw new Error(await leseApiFehler(res, 'Fehler beim Laden der Exemplare'));
			// Die Antwort ist das nackte Array (AntworteJSON), wie Buchakte und Druck-Center
			// sie lesen. Ob ein Exemplar zum Bestand zählt, sagt der Server (im_bestand).
			/** @type {any[]} */
			const alle = (await res.json()) ?? [];
			exemplare = alle.filter((e) => e.im_bestand !== false);
			ausgesondert = alle.filter((e) => e.im_bestand === false && e.ist_ausgesondert).length;
			bestellt = alle.length - exemplare.length - ausgesondert;
		} catch (err) {
			fehler = fehlertext(err);
		} finally {
			laedt = false;
		}
	}

	/** @param {any} ex */
	async function loescheExemplar(ex) {
		if (loescht.has(ex.id)) return;
		if (!(await loeschenBestaetigen(`Exemplar ${ex.barcode_id} löschen?`))) return;
		loescht.add(ex.id);
		try {
			const res = await apiFetch(`/api/buecher/exemplare/${ex.id}`, {
				method: 'DELETE',
				credentials: 'include'
			});
			if (res.ok) {
				// „Löschen" sondert aus: Das Exemplar verlässt den Bestand und die Liste.
				exemplare = exemplare.filter((e) => e.id !== ex.id);
				ausgesondert += 1;
				// Die Zahl im Feld kommt neu vom Server, die Maske zählt nicht selbst.
				await ladeBestandNach(formular);
				melde('Exemplar erfolgreich gelöscht', 'success');
			} else {
				melde(await leseApiFehler(res, 'Fehler beim Löschen des Exemplars.'), 'error');
			}
		} catch {
			melde('Netzwerkfehler beim Löschen.', 'error');
		} finally {
			loescht.delete(ex.id);
		}
	}
</script>

<div class="mt-8 border-t border-outline-variant pt-6">
	<h3 class="text-lg font-semibold text-on-surface">
		{formular.id ? `Exemplare (${exemplare.length})` : 'Exemplare'}
	</h3>
	{#if nichtImBestand}
		<p class="text-sm text-on-surface-variant">
			Nicht im Bestand: {nichtImBestand}. Sie stehen in der Buchakte.
		</p>
	{/if}

	<div class="mt-4">
		<BuchEingabefelderInventar bind:formular />
	</div>

	{#if formular.id}
		<div class="mt-4">
			{#if laedt}
				<div class="text-sm text-on-surface-variant py-4 flex items-center justify-center">
					Lade Exemplare...
				</div>
			{:else if fehler}
				<div class="text-sm text-error py-4">{fehler}</div>
			{:else if exemplare.length === 0}
				<div class="text-sm text-on-surface-variant py-4 italic text-center">
					Kein Exemplar im Bestand.
				</div>
			{:else}
				<!-- Ohne eigene Höhe: Die Liste zeigt alle Exemplare, gescrollt wird die Seite. -->
				<div class="space-y-2">
					{#each exemplare as ex, _i (_i)}
						<!-- Dieselben Exemplare zeigt die Buchakte (ExemplarKarte): umrandete Fläche in
					     outline-variant, Barcode als getönte Chip-Form, Zustand über StatusChip. Zwei
					     Ansichten desselben Exemplars sollen nicht zwei Farbsprachen sprechen. -->
						<div
							class="flex items-center justify-between p-3 rounded-lg border border-outline-variant"
						>
							<div class="flex items-center gap-3">
								<span
									class="rounded-md px-2 py-0.5 font-mono text-xs font-bold whitespace-nowrap bg-primary-container text-on-primary-container"
								>
									{ex.barcode_id}
								</span>
								<StatusChip {...exemplarStatus(ex)} />
								{#if ex.zustand_notiz}
									<span
										class="text-label-small text-on-surface-variant truncate max-w-37.5"
										title={ex.zustand_notiz}>{ex.zustand_notiz}</span
									>
								{/if}
							</div>
							<button
								data-tip="Exemplar löschen"
								aria-label="Exemplar löschen"
								class="icon-btn text-on-surface-variant hover:text-error focus-visible:ring-2 focus-visible:ring-primary focus:outline-none disabled:cursor-not-allowed disabled:text-on-surface/38"
								onclick={() => loescheExemplar(ex)}
								disabled={loescht.has(ex.id)}
							>
								{#if loescht.has(ex.id)}
									<Ladekreis size="sm" farbe="aktuell" />
								{:else}
									<Trash class="w-4 h-4" aria-hidden="true" />
								{/if}
							</button>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</div>
