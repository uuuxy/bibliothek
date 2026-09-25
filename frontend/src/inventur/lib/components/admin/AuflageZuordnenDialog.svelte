<!-- @component AuflageZuordnenDialog — eine andere Auflage desselben Buchs zuordnen
     (docs/OFFEN.md 4.18, Stufe 2). Aufbau wie SchuelerZusammenfuehrenDialog: suchen, wählen,
     bestätigen. Material 3, Dialogs: „Dialogs should contain a maximum of two actions" — eine
     bestätigende und eine abbrechende; „Disable confirming actions until a choice is made.
     Dismissive actions are never disabled." Die Regeln stehen im Server
     (repository/auflagen.go); der Dialog bietet nur an, was dort durchgeht. -->
<script>
	import { AlertCircle } from '@lucide/svelte';
	import { apiClient, apiFetch, extractApiError } from '../../../../lib/apiFetch.js';
	import { showToast } from '$lib/store.svelte.js';
	import Modal from '../../../../lib/Modal.svelte';
	import Button from '../../../../lib/components/ui/Button.svelte';
	import Suchfeld from '../../../../lib/components/ui/Suchfeld.svelte';
	import AuflagenZeile from './AuflagenZeile.svelte';
	import { erzeugeAuflagenSuche } from './auflagenSuche.svelte.js';
	import { auflagenBeschriftung } from './auflagenText.js';

	/** @type {{ open: boolean, titelId: string, bekannte: string[], onZugeordnet: (auflagen: any[]) => void }} */
	let { open = $bindable(false), titelId, bekannte, onZugeordnet } = $props();

	/** @type {any | null} */
	let gewaehlt = $state(null);
	// Wie viele Auflagen das Buch des gewählten Titels schon hat: Hat er selbst welche,
	// kommen sie alle mit — das soll vor dem Klick dastehen, nicht danach. -1 heißt: Die
	// Nachfrage ist gescheitert; dann sagt der Dialog das, statt „keine" zu behaupten.
	let mitgebracht = $state(0);
	let laeuft = $state(false);
	let fehler = $state('');
	const s = erzeugeAuflagenSuche(() => bekannte);

	// Zurückgesetzt wird beim Öffnen wie beim Schließen — sonst stünde beim nächsten Titel
	// der vorige Treffer da (dieselbe Regel wie im Zusammenführen-Dialog).
	$effect(() => {
		if (open) leeren();
		return leeren;
	});

	function leeren() {
		s.zuruecksetzen();
		gewaehlt = null;
		mitgebracht = 0;
		fehler = '';
	}

	/** @param {any} t */
	async function waehle(t) {
		gewaehlt = t;
		mitgebracht = 0;
		try {
			const res = await apiFetch(`/api/buecher/titel/${t.id}/auflagen`);
			const anzahl = res.ok ? ((await res.json()).auflagen ?? []).length : -1;
			if (gewaehlt === t) mitgebracht = anzahl;
		} catch {
			if (gewaehlt === t) mitgebracht = -1;
		}
	}

	async function zuordnen() {
		if (!gewaehlt || laeuft) return;
		laeuft = true;
		fehler = '';
		try {
			const res = await apiClient.post(`/api/buecher/titel/${titelId}/auflagen`, {
				titel_id: gewaehlt.id
			});
			if (!res.ok) {
				fehler = await extractApiError(res);
				return;
			}
			const antwort = await res.json();
			onZugeordnet(antwort.auflagen ?? []);
			showToast(`Zugeordnet: ${auflagenBeschriftung(gewaehlt)}`, 'success');
			open = false;
		} catch {
			fehler = 'Netzwerkfehler — die Zuordnung hat den Server nicht erreicht.';
		} finally {
			laeuft = false;
		}
	}
</script>

<Modal {open} onclose={() => (open = false)} size="2xl" beschriftetDurch="auflage-zuordnen-titel">
	{#snippet header()}
		<h3 id="auflage-zuordnen-titel" class="text-lg font-bold text-on-surface">
			Andere Auflage zuordnen
		</h3>
	{/snippet}
	<div class="space-y-5 px-6 py-6 text-on-surface">
		<p class="text-sm text-on-surface-variant">
			Gesucht wird unter den Lernmitteln, auch unter Titeln ohne Exemplar. Jede Auflage bleibt ein
			eigener Titel mit ihrer ISBN und ihren Exemplaren.
		</p>

		{#if !gewaehlt}
			<div class="space-y-2">
				<Suchfeld
					bind:wert={s.suche}
					platzhalter="Titel, ISBN oder Verlag …"
					etikett="Andere Auflage suchen"
					oninput={s.tippen}
					autofokus
				/>
				{#if s.fehler}
					<p class="text-xs font-bold text-error">{s.fehler}</p>
				{:else if s.treffer.length > 0}
					<ul
						class="max-h-72 divide-y divide-outline-variant/40 overflow-y-auto rounded-xl border border-outline-variant"
					>
						{#each s.treffer as t (t.id)}
							<li>
								<button
									type="button"
									class="w-full cursor-pointer px-3 py-2 text-left transition-colors hover:bg-surface-container"
									onclick={() => waehle(t)}
								>
									<AuflagenZeile
										auflage={t.auflage}
										erscheinungsjahr={t.erscheinungsjahr}
										titel={t.title}
										isbn={t.isbn}
										verlag={t.verlag}
										gesamt={t.gesamt}
										verfuegbar={t.verfuegbar}
										imZulauf={t.imZulauf}
									/>
								</button>
							</li>
						{/each}
					</ul>
				{:else if s.suche.trim().length >= 2}
					<p class="text-xs text-on-surface-variant">Kein Lernmittel gefunden.</p>
				{/if}
			</div>
		{:else}
			<div class="flex items-center gap-3 rounded-lg border border-outline-variant p-3">
				<AuflagenZeile
					auflage={gewaehlt.auflage}
					erscheinungsjahr={gewaehlt.erscheinungsjahr}
					titel={gewaehlt.title}
					isbn={gewaehlt.isbn}
					verlag={gewaehlt.verlag}
					gesamt={gewaehlt.gesamt}
					verfuegbar={gewaehlt.verfuegbar}
					imZulauf={gewaehlt.imZulauf}
				/>
				<Button variant="ghost" size="sm" onclick={() => (gewaehlt = null)}>Andere wählen</Button>
			</div>
			{#if mitgebracht > 1}
				<p class="text-sm text-on-surface-variant">
					Dieser Titel ist schon mit {mitgebracht - 1}
					{mitgebracht === 2 ? 'anderen Auflage' : 'anderen Auflagen'} zusammengefasst; alle kommen mit.
				</p>
			{:else if mitgebracht < 0}
				<p class="text-sm text-on-surface-variant">
					Ob dieser Titel schon mit anderen Auflagen zusammengefasst ist, ließ sich nicht prüfen.
					Hat er welche, kommen sie mit und stehen danach in der Liste.
				</p>
			{/if}
		{/if}

		{#if fehler}
			<div
				role="alert"
				class="flex items-start gap-2 rounded-xl bg-error-container p-3 text-on-error-container"
			>
				<AlertCircle class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
				<p class="text-xs font-bold leading-tight">{fehler}</p>
			</div>
		{/if}
	</div>

	<div
		class="flex justify-end gap-3 border-t border-outline-variant bg-surface-container-low px-6 py-4"
	>
		<Button variant="secondary" onclick={() => (open = false)}>Abbrechen</Button>
		<Button onclick={zuordnen} disabled={!gewaehlt || laeuft}>
			{laeuft ? 'Wird zugeordnet …' : 'Zuordnen'}
		</Button>
	</div>
</Modal>
