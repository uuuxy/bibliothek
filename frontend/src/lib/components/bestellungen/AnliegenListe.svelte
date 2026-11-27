<!-- @component AnliegenListe — Arbeitsliste der Bibliothek für die Meldungen der Lehrkräfte
     aus dem Portal, ohne Ticketsystem: „Abhaken" schickt der Lehrkraft eine Mail mit der
     Notiz. Wünsche legt das Portal nicht mehr an; vorhandene stehen unter den Meldungen. -->
<script>
	import { onMount } from 'svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import { apiFetch, leseApiFehler, FRIST_MAILVERSAND_MS } from '../../apiFetch.js';
	import { snackbars } from '../../stores/snackbars.svelte.js';
	import { fehlertext } from '../../utils/fehlertext.js';
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import Button from '../ui/Button.svelte';
	import ArbeitsZeile from './ArbeitsZeile.svelte';
	import Feld from '../ui/Feld.svelte';
	import Icon from '../ui/Icon.svelte';
	import SuchZustand from '../ui/SuchZustand.svelte';
	import { formatDatum } from '../../utils/format.js';
	import { hatRecht } from '../../menue.js';
	import { authStore } from '../../stores/authStore.svelte.js';

	// Die Tür verlangt create_orders; ohne das Recht scheiterte der Klick mit 403. Wie in
	// BestellDetail steht der Knopf dann nicht da.
	const darfBestellen = $derived(hatRecht(authStore.currentUser, 'create_orders'));

	/** @typedef {{ id: string, art: string, titel_text: string, klasse: string, kommentar?: string, von?: string, erstellt_am: string }} Anliegen */

	/** @type {Anliegen[]} */
	let anliegen = $state([]);
	let laedt = $state(true);
	/** Leer heißt leer — ein Ladefehler heißt Ladefehler. */
	let ladefehler = $state('');
	/** @type {string | null} */
	let confirmingId = $state(null);
	/** @type {string | null} */
	let completingId = $state(null);
	let notiz = $state('');

	// Ein Problem soll am selben Tag erledigt werden, ein Wunsch wartet auf die Bestellung.
	// Die Meldungen tragen keine Zwischenüberschrift: Sie stünde direkt unter der der Seite.
	const abschnitte = $derived(
		[
			{ titel: '', liste: anliegen.filter((a) => a.art === 'meldung') },
			{ titel: 'Wünsche', liste: anliegen.filter((a) => a.art !== 'meldung') }
		].filter((g) => g.liste.length > 0)
	);

	async function ladeAnliegen() {
		laedt = true;
		try {
			const res = await apiFetch('/api/anliegen/offen');
			if (res.ok) {
				anliegen = (await res.json()) || [];
				ladefehler = '';
			} else {
				anliegen = [];
				ladefehler = await leseApiFehler(res);
			}
		} catch {
			anliegen = [];
			ladefehler = 'Die Meldungen konnten nicht geladen werden: Server nicht erreichbar';
		} finally {
			laedt = false;
		}
	}

	onMount(ladeAnliegen);

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
				timeoutMs: FRIST_MAILVERSAND_MS,
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ notiz })
			});
			if (res.status === 404) {
				// Bereits von einem anderen Arbeitsplatz abgehakt — keine zweite Mail.
				anliegen = anliegen.filter((a) => a.id !== id);
				if (confirmingId === id) confirmingId = null;
				uiStore.fetchOffeneAnliegen();
				snackbars.addToast('Meldung war bereits erledigt', 'success');
				return;
			}
			if (!res.ok)
				throw new Error(await leseApiFehler(res, 'Meldung konnte nicht abgehakt werden'));
			anliegen = anliegen.filter((a) => a.id !== id);
			if (confirmingId === id) confirmingId = null;
			uiStore.fetchOffeneAnliegen();
			// Der Server meldet, ob die Benachrichtigung wirklich raus ist — ein
			// Mail-Ausfall war vorher nur eine Server-Logzeile, und hier stand
			// trotzdem „bekommt eine Mail".
			const mail = (await res.json().catch(() => null))?.mail;
			if (mail === 'fehlgeschlagen') {
				snackbars.addToast(
					'Erledigt, aber die Mail an die Lehrkraft konnte nicht versendet werden. Bitte selbst Bescheid geben.',
					'warning'
				);
			} else if (mail === 'keine_adresse') {
				snackbars.addToast('Erledigt, zum Konto ist keine Mail-Adresse hinterlegt', 'warning');
			} else {
				snackbars.addToast('Erledigt, die Lehrkraft bekommt eine Mail', 'success');
			}
		} catch (err) {
			snackbars.addToast(fehlertext(err), 'error');
		} finally {
			completingId = null;
		}
	}
</script>

{#snippet anliegenRow(a)}
	<ArbeitsZeile
		klasse={a.klasse || '–'}
		titel={a.titel_text}
		neben={[a.von, formatDatum(a.erstellt_am)].filter(Boolean).join(' · ')}
		notiz={a.kommentar ?? ''}
		art={a.art === 'wunsch'
			? { text: 'Wunsch', ton: 'neutral' }
			: { text: 'Meldung', ton: 'fehler' }}
	>
		{#snippet aktion()}
			{#if confirmingId !== a.id}
				{#if darfBestellen}
					<Button variant="primary" onclick={() => requestConfirm(a.id)}>
						<Icon name="check" />
						Abhaken
					</Button>
				{/if}
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
				placeholder="Notiz für die Mail an die Lehrkraft"
				aria-label="Notiz für die Mail an die Lehrkraft"
				feld="flex-1"
			/>
			<Button variant="secondary" size="sm" onclick={() => (confirmingId = null)}>Abbrechen</Button>
			<Button
				variant="primary"
				size="sm"
				onclick={() => erledigen(a.id)}
				disabled={completingId !== null}
			>
				{#if completingId === a.id}
					<Ladekreis size="sm" farbe="aktuell" />
				{:else}
					Erledigt & Mail senden
				{/if}
			</Button>
		</li>
	{/if}
{/snippet}

<div class="space-y-6">
	<div>
		<h2 class="sr-only">Meldungen</h2>
		<p class="text-sm text-on-surface-variant">
			Meldungen der Lehrkräfte aus dem Kollegiums-Portal, je die ältesten zuerst. „Abhaken" schließt
			die Meldung ab und schickt der Lehrkraft eine Mail, mit deiner Notiz, wenn du eine schreibst.
		</p>
	</div>

	{#if laedt}
		<div class="flex justify-center py-16"><Ladekreis size="lg" verzoegert /></div>
	{:else if ladefehler}
		<LadeFehler onerneut={ladeAnliegen} titel="Meldungen nicht geladen" text={ladefehler} />
	{:else if anliegen.length === 0}
		<SuchZustand symbol="inbox" titel="Keine offenen Meldungen" />
	{:else}
		<!-- M3 Lists: Ein Abstand trennt die Gruppen, die Linien nur die Zeilen einer Gruppe. -->
		{#each abschnitte as g (g.titel)}
			<section class="space-y-1">
				{#if g.titel}
					<h3 class="text-title-medium text-on-surface">{g.titel}</h3>
				{/if}
				<ul class="divide-y divide-outline-variant">
					{#each g.liste as a (a.id)}
						{@render anliegenRow(a)}
					{/each}
				</ul>
			</section>
		{/each}
	{/if}
</div>
