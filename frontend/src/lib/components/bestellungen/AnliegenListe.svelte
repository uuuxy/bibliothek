<!-- @component AnliegenListe — LMF-Arbeitsliste für Wünsche und Meldungen der
     Lehrkräfte (Betreiber-Entscheidung 18.08.2026: EIN schlanker Mechanismus,
     kein Ticketsystem). Lehrkräfte legen Anliegen im Kollegiums-Portal an;
     hier wird in Ruhe abgearbeitet — „Abhaken" schickt der Lehrkraft eine
     Mail mit der optionalen Notiz. Älteste zuerst, wie die Klassensätze. -->
<script>
	import { onMount } from 'svelte';
	import { apiFetch } from '../../apiFetch.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import Button from '../ui/Button.svelte';
	import ArbeitsZeile from './ArbeitsZeile.svelte';
	import Feld from '../ui/Feld.svelte';
	import { Check } from '@lucide/svelte';

	/** @typedef {{ id: string, art: string, titel_text: string, isbn?: string, klasse: string, kommentar?: string, von?: string, erstellt_am: string }} Anliegen */

	/** @type {Anliegen[]} */
	let anliegen = $state([]);
	let loading = $state(true);
	/** @type {string | null} */
	let confirmingId = $state(null);
	/** @type {string | null} */
	let completingId = $state(null);
	let notiz = $state('');

	async function loadAnliegen() {
		loading = true;
		try {
			const res = await apiFetch('/api/anliegen/offen');
			anliegen = res.ok ? await res.json() : [];
		} catch {
			anliegen = [];
		} finally {
			loading = false;
		}
	}

	onMount(loadAnliegen);

	/** @param {string} id */
	function requestConfirm(id) {
		confirmingId = id;
		notiz = '';
	}

	/** @param {string} id */
	async function erledigen(id) {
		if (completingId) return;
		completingId = id;
		try {
			const res = await apiFetch(`/api/anliegen/${id}/erledigen`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ notiz })
			});
			if (res.status === 404) {
				// Bereits von einem anderen Arbeitsplatz abgehakt — keine zweite Mail.
				anliegen = anliegen.filter((a) => a.id !== id);
				confirmingId = null;
				uiStore.fetchOffeneAnliegen();
				toastStore.addToast('Anliegen war bereits erledigt.', 'success');
				return;
			}
			if (!res.ok) {
				const data = await res.json().catch(() => null);
				throw new Error(data?.error || 'Anliegen konnte nicht abgehakt werden.');
			}
			anliegen = anliegen.filter((a) => a.id !== id);
			confirmingId = null;
			uiStore.fetchOffeneAnliegen();
			// Der Server meldet, ob die Benachrichtigung wirklich raus ist — ein
			// Mail-Ausfall war vorher nur eine Server-Logzeile, und hier stand
			// trotzdem „bekommt eine Mail".
			const mail = (await res.json().catch(() => null))?.mail;
			if (mail === 'fehlgeschlagen') {
				toastStore.addToast(
					'Erledigt, aber die Mail an die Lehrkraft konnte nicht versendet werden — bitte selbst Bescheid geben.',
					'warning'
				);
			} else if (mail === 'keine_adresse') {
				toastStore.addToast('Erledigt — zum Konto ist keine Mail-Adresse hinterlegt.', 'warning');
			} else {
				toastStore.addToast('Erledigt — die Lehrkraft bekommt eine Mail.', 'success');
			}
		} catch (err) {
			toastStore.addToast(/** @type {any} */ (err).message || String(err), 'error');
		} finally {
			completingId = null;
		}
	}
</script>

{#snippet anliegenRow(a)}
	<ArbeitsZeile
		klasse={a.klasse || '–'}
		titel={a.titel_text}
		neben={[
			a.von,
			new Date(a.erstellt_am).toLocaleDateString('de-DE'),
			a.isbn ? `ISBN ${a.isbn}` : ''
		]
			.filter(Boolean)
			.join(' · ')}
		notiz={a.kommentar ?? ''}
		art={a.art === 'wunsch'
			? { text: 'Wunsch', ton: 'neutral' }
			: { text: 'Meldung', ton: 'fehler' }}
	>
		{#snippet aktion()}
			{#if confirmingId !== a.id}
				<Button variant="primary" onclick={() => requestConfirm(a.id)}>
					<Check class="w-4 h-4" aria-hidden="true" />
					Abhaken
				</Button>
			{/if}
		{/snippet}
	</ArbeitsZeile>
	{#if confirmingId === a.id}
		<!-- Die Notiz landet in der Mail an die Lehrkraft — ein Einzeiler wie
		     „bestellt, kommt Anfang September" erspart die Rückfrage im Flur. -->
		<li class="flex items-center gap-2 pb-3 pl-16">
			<Feld
				bind:value={notiz}
				maxlength={500}
				placeholder="Notiz für die Mail an die Lehrkraft (optional)"
				aria-label="Notiz für die Mail an die Lehrkraft"
				feld="flex-1"
			/>
			<Button
				variant="secondary"
				size="sm"
				onclick={() => (confirmingId = null)}
				disabled={completingId === a.id}
			>
				Abbrechen
			</Button>
			<Button
				variant="primary"
				size="sm"
				onclick={() => erledigen(a.id)}
				disabled={completingId === a.id}
			>
				{#if completingId === a.id}
					<span class="w-3 h-3 border-2 border-white/60 border-t-white rounded-full animate-spin"
					></span>
				{:else}
					Erledigt & Mail senden
				{/if}
			</Button>
		</li>
	{/if}
{/snippet}

<div class="space-y-6">
	<div>
		<h2 class="text-base font-bold text-on-surface">Wünsche & Meldungen</h2>
		<p class="text-sm text-on-surface-variant mt-0.5">
			Anliegen der Lehrkräfte aus dem Kollegiums-Portal, älteste zuerst. „Abhaken" schließt das
			Anliegen ab und schickt der Lehrkraft eine Mail — mit deiner Notiz, wenn du eine schreibst.
		</p>
	</div>

	{#if loading}
		<div class="py-16 text-center text-on-surface-variant text-base animate-pulse">
			Lade Anliegen…
		</div>
	{:else if anliegen.length === 0}
		<div class="py-16 text-center text-on-surface-variant text-base">Keine offenen Anliegen.</div>
	{:else}
		<ul class="divide-y divide-outline-variant">
			{#each anliegen as a (a.id)}
				{@render anliegenRow(a)}
			{/each}
		</ul>
	{/if}
</div>
