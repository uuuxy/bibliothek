<!-- @component KlassensatzReservierungen — Admin-Arbeitsliste für Klassensatz-Reservierungen.
     Lehrkräfte legen die Anfrage über KollegiumPortal an; hier wird sie geprüft und über
     PUT /api/reservierungen/klassensatz/{id}/erledigen abgeschlossen. Reservieren SPERRT
     keinen Bestand (Warteschlangen-Modell, 16.08.2026): Die Liste ist die Reihenfolge,
     „verfügbar" zeigt je Zeile, ob der Satz aktuell vollständig aus dem Regal zu holen
     ist. Flaches Edge-to-Edge-Listen-Design, kein Modal.

     Seit 27.08.2026 kann die Theke beim Abschließen eine Notiz für die Bereit-Mail
     schreiben — dasselbe Muster wie AnliegenListe. Vorher war die Mail ein fester Text,
     und „24 von 30, Rest bei der 8a" ging nur mündlich im Flur. -->
<script>
	import { onMount } from 'svelte';
	import { apiFetch } from '../../apiFetch.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import ArbeitsZeile from './ArbeitsZeile.svelte';
	import KlassensatzErledigte from './KlassensatzErledigte.svelte';
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import { Check } from '@lucide/svelte';

	/** @typedef {{ id: string, titel_name: string, klasse: string, anzahl: number, verfuegbar: number, notiz?: string, angefordert_von?: string, erledigt: boolean, erledigt_notiz?: string, erledigt_am?: string, erstellt_am: string }} KlassensatzReservierung */

	/** @type {KlassensatzReservierung[]} */
	let reservierungen = $state([]);
	let loading = $state(true);
	/** @type {string | null} */
	let confirmingId = $state(null);
	/** @type {string | null} */
	let completingId = $state(null);
	let notiz = $state('');

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
		notiz = '';
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
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ notiz })
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
			// Kein Reload: Die Zeile wandert lokal samt Notiz in „Zuletzt bereitgestellt".
			// Vorher wurde sie entfernt — die Zusage war bis zum nächsten Laden unauffindbar.
			const heute = new Date().toLocaleDateString('de-DE', { dateStyle: 'medium' });
			reservierungen = reservierungen.map((r) =>
				r.id === id ? { ...r, erledigt: true, erledigt_notiz: notiz.trim(), erledigt_am: heute } : r
			);
			confirmingId = null;
			// Der Server meldet, ob die Bereit-Mail wirklich raus ist — ein Mail-Ausfall
			// war vorher nur eine Server-Logzeile, die Lehrkraft wartete vergeblich.
			const mail = (await res.json().catch(() => null))?.mail;
			if (mail === 'fehlgeschlagen') {
				toastStore.addToast(
					'Abgeschlossen, aber die Bereit-Mail konnte nicht versendet werden — bitte die Lehrkraft selbst benachrichtigen.',
					'warning'
				);
			} else if (mail === 'keine_adresse') {
				toastStore.addToast(
					'Abgeschlossen — zum Konto ist keine Mail-Adresse hinterlegt, bitte die Lehrkraft selbst benachrichtigen.',
					'warning'
				);
			} else {
				toastStore.addToast('Reservierung abgeschlossen.', 'success');
			}
			uiStore.fetchPendingReservierungen();
		} catch (err) {
			toastStore.addToast(/** @type {any} */ (err).message || String(err), 'error');
		} finally {
			completingId = null;
		}
	}
</script>

{#snippet reservierungRow(r)}
	<ArbeitsZeile
		klasse={r.klasse}
		titel={r.titel_name}
		neben={[r.angefordert_von, r.erstellt_am].filter(Boolean).join(' · ')}
		anzahl={r.anzahl}
		notiz={r.notiz ?? ''}
		chip={r.verfuegbar == null
			? undefined
			: r.verfuegbar >= r.anzahl
				? { text: `${r.verfuegbar} verfügbar`, ton: 'erfolg' }
				: {
						text: `nur ${r.verfuegbar} verfügbar`,
						ton: 'fehler',
						tip: 'Regal-Blick: verliehene oder ausgesonderte Exemplare fehlen gerade'
					}}
	>
		{#snippet aktion()}
			{#if confirmingId !== r.id}
				<Button variant="primary" onclick={() => requestConfirm(r.id)}>
					<Check class="w-4 h-4" aria-hidden="true" />
					Abschließen
				</Button>
			{/if}
		{/snippet}
	</ArbeitsZeile>
	{#if confirmingId === r.id}
		<!-- Die Notiz landet in der Bereit-Mail — „24 von 30, Rest bei der 8a" oder
		     „steht hinter der Theke, bitte bis Freitag" erspart die Rückfrage im Flur. -->
		<li class="flex items-center gap-2 pb-3 pl-16">
			<Feld
				bind:value={notiz}
				maxlength={500}
				placeholder="Notiz für die Bereit-Mail an die Lehrkraft (optional)"
				aria-label="Notiz für die Bereit-Mail an die Lehrkraft"
				feld="flex-1"
			/>
			<Button
				variant="secondary"
				size="sm"
				onclick={cancelConfirm}
				disabled={completingId === r.id}
			>
				Abbrechen
			</Button>
			<Button
				variant="primary"
				size="sm"
				onclick={() => completeReservierung(r.id)}
				disabled={completingId === r.id}
			>
				{#if completingId === r.id}
					<span class="w-3 h-3 border-2 border-white/60 border-t-white rounded-full animate-spin"
					></span>
				{:else}
					Abschließen & Mail senden
				{/if}
			</Button>
		</li>
	{/if}
{/snippet}

<div class="space-y-6">
	<div>
		<h2 class="text-base font-bold text-slate-800">Klassensatz-Reservierungen</h2>
		<p class="text-sm text-slate-500 mt-0.5">
			Von Lehrkräften angefragte Klassensätze in Warteschlangen-Reihenfolge (älteste zuerst).
			Reservieren sperrt keinen Bestand — „Abschließen" schließt den Vorgang nach der Übergabe ab
			und schickt der Lehrkraft die Bereit-Mail — mit deiner Notiz, wenn du eine schreibst.
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

	<KlassensatzErledigte erledigte={reservierungen.filter((r) => r.erledigt)} />
</div>
