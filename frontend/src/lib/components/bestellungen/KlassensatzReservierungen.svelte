<!-- @component KlassensatzReservierungen — Admin-Arbeitsliste für Klassensatz-Reservierungen.
     Lehrkräfte legen die Anfrage über LehrerPortal an; hier wird sie geprüft und über
     PUT /api/reservierungen/klassensatz/{id}/erledigen abgeschlossen (gibt den geblockten
     Bestand wieder frei). Flaches Edge-to-Edge-Listen-Design, kein Modal. -->
<script>
	import { onMount } from 'svelte';
	import { apiFetch } from '../../apiFetch.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import { uiStore } from '../../stores/uiStore.svelte.js';

	/** @typedef {{ id: string, titel_name: string, klasse: string, anzahl: number, notiz?: string, angefordert_von?: string, erledigt: boolean, erstellt_am: string }} KlassensatzReservierung */

	/** @type {KlassensatzReservierung[]} */
	let reservierungen = $state([]);
	let loading = $state(true);
	/** @type {string | null} */
	let confirmingId = $state(null);
	/** @type {string | null} */
	let completingId = $state(null);

	// GET liefert die gesamte Historie (erledigt + offen); hier interessieren nur die offenen.
	const offeneReservierungen = $derived(reservierungen.filter((r) => !r.erledigt));

	async function loadReservierungen() {
		loading = true;
		try {
			const res = await apiFetch('/api/reservierungen/klassensatz');
			reservierungen = res.ok ? await res.json() : [];
		} catch {
			reservierungen = [];
		} finally {
			loading = false;
		}
	}

	onMount(loadReservierungen);

	/** @param {string} id */
	function requestConfirm(id) {
		confirmingId = id;
	}

	function cancelConfirm() {
		confirmingId = null;
	}

	/** @param {string} id */
	async function completeReservierung(id) {
		if (completingId) return;
		completingId = id;
		try {
			const res = await apiFetch(`/api/reservierungen/klassensatz/${id}/erledigen`, {
				method: 'PUT'
			});
			if (res.status === 404) {
				// Bereits anderweitig erledigt (z. B. zweiter Admin) — lokal genauso entfernen
				reservierungen = reservierungen.filter((r) => r.id !== id);
				confirmingId = null;
				toastStore.addToast('Reservierung war bereits abgeschlossen.', 'success');
				return;
			}
			if (!res.ok) {
				const data = await res.json().catch(() => null);
				throw new Error(data?.error || 'Reservierung konnte nicht abgeschlossen werden.');
			}
			// Kein Reload: die erledigte Reservierung wird direkt aus dem reaktiven Array gefiltert.
			reservierungen = reservierungen.filter((r) => r.id !== id);
			confirmingId = null;
			toastStore.addToast('Reservierung abgeschlossen — Bestand wieder freigegeben.', 'success');
			uiStore.fetchPendingReservierungen();
		} catch (err) {
			toastStore.addToast(/** @type {any} */ (err).message || String(err), 'error');
		} finally {
			completingId = null;
		}
	}
</script>

{#snippet reservierungRow(r)}
	<li class="flex items-center justify-between gap-4 py-4">
		<div class="min-w-0 flex-1">
			<p class="text-sm font-bold text-slate-800 truncate">{r.titel_name}</p>
			<p class="text-xs text-slate-450 mt-0.5">
				Klasse <span class="font-semibold text-slate-600">{r.klasse}</span> · {r.anzahl} Exemplare
				{#if r.angefordert_von}· angefragt von {r.angefordert_von}{/if}
			</p>
			{#if r.notiz}
				<p class="text-xs text-slate-400 italic mt-1 truncate">„{r.notiz}"</p>
			{/if}
		</div>
		<div class="text-xs text-slate-400 shrink-0 w-20 text-right">{r.erstellt_am}</div>
		<div class="shrink-0">
			{#if confirmingId === r.id}
				<div class="flex items-center gap-2">
					<button
						onclick={cancelConfirm}
						disabled={completingId === r.id}
						class="px-3 py-2 rounded-full bg-slate-100 hover:bg-slate-200 disabled:opacity-50 text-slate-650 text-xs font-bold transition-colors cursor-pointer"
					>
						Abbrechen
					</button>
					<button
						onclick={() => completeReservierung(r.id)}
						disabled={completingId === r.id}
						class="px-3 py-2 rounded-full bg-rose-600 hover:bg-rose-700 disabled:opacity-50 disabled:cursor-not-allowed text-white text-xs font-bold transition-all cursor-pointer flex items-center gap-2"
					>
						{#if completingId === r.id}
							<span
								class="w-3 h-3 border-2 border-white/60 border-t-white rounded-full animate-spin"
							></span>
						{:else}
							Wirklich abschließen?
						{/if}
					</button>
				</div>
			{:else}
				<button
					onclick={() => requestConfirm(r.id)}
					class="px-3 py-2 rounded-full bg-slate-900 hover:bg-slate-800 text-white text-xs font-bold transition-colors cursor-pointer flex items-center gap-1.5"
				>
					<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"
						><path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M5 13l4 4L19 7"
						/></svg
					>
					Abschließen
				</button>
			{/if}
		</div>
	</li>
{/snippet}

<div class="space-y-6">
	<div>
		<h2 class="text-base font-bold text-slate-800">Klassensatz-Reservierungen</h2>
		<p class="text-sm text-slate-500 mt-0.5">
			Von Lehrkräften angefragte Klassensätze — „Abschließen" gibt die geblockten Bestände wieder
			frei.
		</p>
	</div>

	{#if loading}
		<div class="py-16 text-center text-slate-400 text-base animate-pulse">Lade Reservierungen…</div>
	{:else if offeneReservierungen.length === 0}
		<div class="py-16 text-center text-slate-400 text-base">
			Keine offenen Klassensatz-Reservierungen.
		</div>
	{:else}
		<ul class="divide-y divide-slate-100">
			{#each offeneReservierungen as r (r.id)}
				{@render reservierungRow(r)}
			{/each}
		</ul>
	{/if}
</div>
