<script>
	import { apiFetch } from '../../../../lib/apiFetch.js';
	import { loeschenBestaetigen } from '../../../../lib/stores/bestaetigung.svelte.js';
	import { showToast } from '$lib/store.svelte.js';
	import { ladeBestandNach } from '../../buch_speichern.js';
	import { onMount } from 'svelte';
	import { Trash2 } from '@lucide/svelte';
	import StatusChip from '../../../../lib/components/ui/StatusChip.svelte';
	import BuchEingabefelderInventar from './BuchEingabefelderInventar.svelte';

	let { formular = $bindable() } = $props();

	// Der Abschnitt „Exemplare": die Zahl im Feld „Aktueller Bestand" und darunter die Liste,
	// die denselben Bestand zeigt. Das Feld legt beim Speichern Exemplare an oder sondert aus;
	// ein neuer Titel hat das Feld, aber noch keine Liste. Ausgesonderte und bestellte
	// Exemplare zählt der Bestand nicht; sie führt die Buchakte.
	/** @type {any[]} */
	let exemplare = $state([]);
	let ausgesondert = $state(0);
	let bestellt = $state(0);
	let loading = $state(true);
	let error = $state('');

	const nichtImBestand = $derived(
		[ausgesondert && `${ausgesondert} ausgesondert`, bestellt && `${bestellt} bestellt`]
			.filter(Boolean)
			.join(', ')
	);

	onMount(() => {
		loadExemplare();
	});

	async function loadExemplare() {
		if (!formular.id) return;
		loading = true;
		error = '';
		try {
			const res = await apiFetch(`/api/buecher/titel/${formular.id}/exemplare`, {
				credentials: 'include'
			});
			if (!res.ok) {
				const err = await res.json().catch(() => ({}));
				throw new Error(err.error || 'Fehler beim Laden der Exemplare');
			}
			// Die Antwort ist das nackte Array (RespondJSON), wie Buchakte und Druck-Center
			// sie lesen. Ob ein Exemplar zum Bestand zählt, sagt der Server (im_bestand).
			/** @type {any[]} */
			const alle = (await res.json()) ?? [];
			exemplare = alle.filter((e) => e.im_bestand !== false);
			ausgesondert = alle.filter((e) => e.im_bestand === false && e.ist_ausgesondert).length;
			bestellt = alle.length - exemplare.length - ausgesondert;
		} catch (err) {
			error = err instanceof Error ? err.message : String(err);
		} finally {
			loading = false;
		}
	}

	/** @param {any} ex */
	async function deleteCopy(ex) {
		if (!(await loeschenBestaetigen(`Exemplar ${ex.barcode_id} löschen?`))) return;
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
				showToast('Exemplar erfolgreich gelöscht', 'success');
			} else {
				const err = await res.json().catch(() => ({}));
				showToast(err.error || 'Fehler beim Löschen des Exemplars.', 'error');
			}
		} catch {
			showToast('Netzwerkfehler beim Löschen.', 'error');
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
			{#if loading}
				<div class="text-sm text-on-surface-variant py-4 flex items-center justify-center">
					Lade Exemplare...
				</div>
			{:else if error}
				<div class="text-sm text-error py-4">{error}</div>
			{:else if exemplare.length === 0}
				<div class="text-sm text-on-surface-variant py-4 italic text-center">
					Kein Exemplar im Bestand.
				</div>
			{:else}
				<!-- Ohne eigene Höhe: Die Liste zeigt alle Exemplare, gescrollt wird die Seite. -->
				<div class="space-y-2">
					{#each exemplare as ex, _i (_i)}
						<!-- Dieselben Exemplare zeigt die Buchakte (BookExemplarCard): umrandete Fläche in
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
								<StatusChip
									ton={!ex.ist_ausleihbar ? 'fehler' : !ex.ist_verfuegbar ? 'warten' : 'erfolg'}
									text={!ex.ist_ausleihbar
										? 'Gesperrt'
										: !ex.ist_verfuegbar
											? 'Ausgeliehen'
											: 'Verfügbar'}
								/>
								{#if ex.zustand_notiz}
									<span
										class="text-label-small text-on-surface-variant truncate max-w-37.5"
										title={ex.zustand_notiz}>{ex.zustand_notiz}</span
									>
								{/if}
							</div>
							<button
								title="Exemplar löschen"
								aria-label="Exemplar löschen"
								class="icon-btn text-on-surface-variant hover:text-error focus-visible:ring-2 focus-visible:ring-primary focus:outline-none"
								onclick={() => deleteCopy(ex)}
							>
								<Trash2 class="w-4 h-4" aria-hidden="true" />
							</button>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</div>
