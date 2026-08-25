<script>
	import { onMount } from 'svelte';
	import { apiFetch, apiClient } from '../apiFetch.js';
	import { toastStore } from '../stores/toastStore.svelte.js';
	import Button from './ui/Button.svelte';
	import Feld from './ui/Feld.svelte';
	import { Plus, Wrench, MonitorSmartphone } from '@lucide/svelte';

	/**
	 * Geräte-Verwaltung (Bereich im Medienkatalog, Betreiber-Entscheidung 16.08.2026:
	 * kein eigener Menüpunkt — ausgebucht wird ohnehin über die Omnibox mit G--Barcode).
	 * Anlegen, Liste mit aktuellem Ausleiher, Defekt-Schalter, Stammdaten-Pflege.
	 */

	/** @type {any[]} */
	let geraete = $state([]);
	let laedt = $state(true);
	let formOffen = $state(false);
	/** @type {string | null} */
	let bearbeiteId = $state(null);
	let form = $state({
		modellname: '',
		barcode_id: '',
		seriennummer: '',
		zubehoer: '',
		zustand_notiz: ''
	});
	let speichert = $state(false);

	async function lade() {
		laedt = true;
		try {
			const res = await apiFetch('/api/geraete');
			const data = res.ok ? await res.json() : null;
			geraete = Array.isArray(data?.data) ? data.data : [];
		} catch {
			geraete = [];
		} finally {
			laedt = false;
		}
	}
	onMount(lade);

	function oeffneAnlegen() {
		bearbeiteId = null;
		form = { modellname: '', barcode_id: 'G-', seriennummer: '', zubehoer: '', zustand_notiz: '' };
		formOffen = true;
	}

	/** @param {any} g */
	function oeffneBearbeiten(g) {
		bearbeiteId = g.id;
		form = {
			modellname: g.modellname,
			barcode_id: g.barcode_id,
			seriennummer: g.seriennummer ?? '',
			zubehoer: g.zubehoer ?? '',
			zustand_notiz: g.zustand_notiz ?? ''
		};
		formOffen = true;
	}

	function schliesseForm() {
		formOffen = false;
		bearbeiteId = null;
	}

	async function speichere() {
		speichert = true;
		try {
			const res = bearbeiteId
				? await apiClient.put(`/api/geraete/${bearbeiteId}`, form)
				: await apiClient.post('/api/geraete', form);
			if (res.ok) {
				toastStore.addToast(bearbeiteId ? 'Gerät gespeichert.' : 'Gerät angelegt.', 'success');
				schliesseForm();
				lade();
			} else {
				const err = await res.json().catch(() => ({}));
				toastStore.addToast(err.error || 'Speichern fehlgeschlagen.', 'error');
			}
		} catch {
			toastStore.addToast('Netzwerkfehler.', 'error');
		} finally {
			speichert = false;
		}
	}

	/** @param {any} g */
	async function schalteDefekt(g) {
		try {
			const res = await apiClient.put(`/api/geraete/${g.id}`, {
				modellname: g.modellname,
				zubehoer: g.zubehoer ?? '',
				zustand_notiz: g.zustand_notiz ?? '',
				ist_ausleihbar: !g.ist_ausleihbar
			});
			if (res.ok) {
				lade();
			} else {
				const err = await res.json().catch(() => ({}));
				toastStore.addToast(err.error || 'Statuswechsel fehlgeschlagen.', 'error');
			}
		} catch {
			toastStore.addToast('Netzwerkfehler.', 'error');
		}
	}
</script>

<div class="space-y-6 pt-2">
	<div class="flex items-center justify-between border-b border-outline-variant pb-3">
		<div>
			<h2 class="text-base font-bold text-on-surface">Geräte ({geraete.length})</h2>
			<p class="mt-0.5 text-sm text-on-surface-variant">
				Ausgebucht wird am Kiosk über den G--Barcode; das Zubehör wird dort als Checkliste
				bestätigt.
			</p>
		</div>
		<Button variant="primary" onclick={oeffneAnlegen}>
			<Plus class="h-4 w-4" aria-hidden="true" />
			Gerät anlegen
		</Button>
	</div>

	{#if formOffen}
		<div class="rounded-2xl border border-outline-variant bg-surface-container-low p-4">
			<div class="grid gap-3 sm:grid-cols-2">
				<Feld label="Modellname *" bind:value={form.modellname} placeholder="z. B. iPad 9. Gen" />
				<Feld label="Barcode (G-…) *" bind:value={form.barcode_id} disabled={bearbeiteId != null} />
				<Feld label="Seriennummer" bind:value={form.seriennummer} disabled={bearbeiteId != null} />
				<Feld
					label="Zubehör (Checkliste, mit Komma getrennt)"
					bind:value={form.zubehoer}
					placeholder="Ladekabel, Stift, Hülle"
				/>
				<Feld label="Zustandsnotiz" bind:value={form.zustand_notiz} class="sm:col-span-2" />
			</div>
			<div class="mt-4 flex justify-end gap-2">
				<Button variant="secondary" onclick={schliesseForm} disabled={speichert}>Abbrechen</Button>
				<Button variant="primary" onclick={speichere} disabled={speichert}>
					{bearbeiteId ? 'Speichern' : 'Anlegen'}
				</Button>
			</div>
		</div>
	{/if}

	{#if laedt}
		<p class="py-12 text-center text-sm text-on-surface-variant">Lade Geräte …</p>
	{:else if geraete.length === 0}
		<div class="flex flex-col items-center gap-3 py-16 text-on-surface-variant">
			<MonitorSmartphone class="h-12 w-12" aria-hidden="true" />
			<p class="text-sm font-semibold">Noch keine Geräte erfasst.</p>
		</div>
	{:else}
		<ul class="divide-y divide-outline-variant">
			{#each geraete as g (g.id)}
				<li class="flex items-center justify-between gap-4 py-4">
					<div class="min-w-0 flex-1">
						<p class="truncate text-sm font-bold text-on-surface">{g.modellname}</p>
						<p class="mt-0.5 text-xs text-on-surface-variant">
							{g.barcode_id}{#if g.seriennummer}&nbsp;· SN {g.seriennummer}{/if}
							{#if g.zubehoer}&nbsp;· Zubehör: {g.zubehoer}{/if}
						</p>
						{#if g.zustand_notiz}
							<p class="mt-0.5 text-xs italic text-on-surface-variant">{g.zustand_notiz}</p>
						{/if}
					</div>
					<span
						class="shrink-0 rounded-full px-2 py-0.5 text-label-small font-semibold {g.ausgeliehen_an
							? 'bg-secondary-container text-on-secondary-container'
							: g.ist_ausleihbar
								? 'bg-primary-container text-on-primary-container'
								: 'bg-error-container text-on-error-container'}"
					>
						{g.ausgeliehen_an
							? `verliehen an ${g.ausgeliehen_an}`
							: g.ist_ausleihbar
								? 'im Schrank'
								: 'defekt/gesperrt'}
					</span>
					<div class="flex shrink-0 items-center gap-2">
						<Button variant="secondary" size="sm" onclick={() => oeffneBearbeiten(g)}>
							Bearbeiten
						</Button>
						<Button
							variant={g.ist_ausleihbar ? 'danger' : 'secondary'}
							size="sm"
							onclick={() => schalteDefekt(g)}
						>
							<Wrench class="h-3.5 w-3.5" aria-hidden="true" />
							{g.ist_ausleihbar ? 'Defekt melden' : 'Wieder freigeben'}
						</Button>
					</div>
				</li>
			{/each}
		</ul>
	{/if}
</div>
