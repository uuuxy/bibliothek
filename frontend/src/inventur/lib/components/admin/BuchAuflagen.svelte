<!-- @component BuchAuflagen — der Abschnitt „Auflagen" der Titelmaske (docs/OFFEN.md 4.18,
     Stufe 2). Eine neue Auflage ist ein eigener Titel mit eigener ISBN; hier ordnet man die
     Titel zu, die dasselbe Buch sind, und löst sie wieder. Die Regeln stehen im Server
     (repository/auflagen.go): nur Lernmittel, zwei Gruppen werden eine, ein Buch mit einer
     Auflage ist keine Gruppe.

     Sichtbar bei jedem Lernmittel — und bei jedem Titel, der schon zu einem Buch gehört, auch
     wenn er inzwischen kein Lernmittel mehr ist: Sonst ließe er sich nicht mehr lösen.

     Aufbau wie der Abschnitt „Exemplare" darunter (Trennlinie, Überschrift, umrandete Zeilen),
     damit die beiden Abschnitte derselben Maske gleich lesen. Lösen fragt nicht nach: Es lässt
     sich mit „Andere Auflage zuordnen" jederzeit zurücknehmen und löscht nichts. -->
<script>
	import { onMount } from 'svelte';
	import { Plus, Unlink } from '@lucide/svelte';
	import { apiFetch, extractApiError } from '../../../../lib/apiFetch.js';
	import { showToast } from '$lib/store.svelte.js';
	import { bestandSatz } from '../../../../lib/utils/format.js';
	import Button from '../../../../lib/components/ui/Button.svelte';
	import AuflagenZeile from './AuflagenZeile.svelte';
	import AuflageZuordnenDialog from './AuflageZuordnenDialog.svelte';
	import { auflagenBeschriftung } from '../../../../lib/utils/auflagenText.js';

	/** @type {{ formular: any }} */
	let { formular } = $props();

	/** @type {any[]} */
	let auflagen = $state([]);
	let geladen = $state(false);
	let fehler = $state('');
	let dialogOffen = $state(false);
	let loest = $state('');

	onMount(() => {
		laden();
	});

	async function laden() {
		if (!formular.id) return;
		try {
			const res = await apiFetch(`/api/buecher/titel/${formular.id}/auflagen`);
			if (!res.ok) {
				fehler = await extractApiError(res);
				return;
			}
			auflagen = (await res.json()).auflagen ?? [];
			fehler = '';
		} catch {
			fehler = 'Netzwerkfehler — die Auflagen ließen sich nicht laden.';
		} finally {
			geladen = true;
		}
	}

	/** @param {any} a */
	async function loesen(a) {
		if (loest) return;
		loest = a.id;
		try {
			const res = await apiFetch(`/api/buecher/titel/${a.id}/auflagen`, { method: 'DELETE' });
			if (!res.ok) {
				showToast(await extractApiError(res), 'error');
				return;
			}
			// Die Antwort gilt dem gelösten Titel; die Liste DIESES Titels wird neu gelesen,
			// denn mit dem vorletzten fällt auch der letzte heraus.
			await laden();
			showToast(`Gelöst: ${auflagenBeschriftung(a)}`, 'success');
		} catch {
			showToast('Netzwerkfehler — die Auflage wurde nicht gelöst.', 'error');
		} finally {
			loest = '';
		}
	}

	const sichtbar = $derived(formular.istLernmittel || auflagen.length > 1);
	const zusammen = $derived(
		bestandSatz(
			auflagen.reduce((n, a) => n + a.gesamt, 0),
			auflagen.reduce((n, a) => n + a.verfuegbar, 0),
			auflagen.reduce((n, a) => n + a.im_zulauf, 0)
		)
	);
</script>

{#if sichtbar}
	<div class="mt-8 border-t border-outline-variant pt-6">
		<h3 class="text-lg font-semibold text-on-surface">Auflagen</h3>
		<p class="mb-4 mt-1 text-sm text-on-surface-variant">
			Andere Auflagen desselben Buchs — jede behält ihre ISBN und ihre Exemplare.
		</p>

		{#if !geladen}
			<p class="py-2 text-sm text-on-surface-variant">Lade Auflagen …</p>
		{:else if fehler}
			<p class="py-2 text-sm text-error" role="alert">{fehler}</p>
		{:else if auflagen.length < 2}
			<p class="py-2 text-sm text-on-surface-variant">Keine andere Auflage zugeordnet.</p>
		{:else}
			<ul class="space-y-2" aria-label="Auflagen dieses Buchs">
				{#each auflagen as a (a.id)}
					<li class="flex items-center gap-3 rounded-lg border border-outline-variant p-3">
						<AuflagenZeile
							auflage={a.auflage}
							erscheinungsjahr={a.erscheinungsjahr}
							titel={a.titel}
							isbn={a.isbn}
							verlag={a.verlag}
							gesamt={a.gesamt}
							verfuegbar={a.verfuegbar}
							imZulauf={a.im_zulauf}
							diese={a.id === formular.id}
						/>
						<button
							type="button"
							class="icon-btn text-on-surface-variant hover:text-error focus-visible:ring-2 focus-visible:ring-primary focus:outline-none"
							data-tip="Aus den Auflagen lösen"
							aria-label={`${auflagenBeschriftung(a)} aus den Auflagen lösen`}
							disabled={loest !== ''}
							onclick={() => loesen(a)}
						>
							<Unlink class="h-4 w-4" aria-hidden="true" />
						</button>
					</li>
				{/each}
			</ul>
			<p class="mt-3 text-sm text-on-surface-variant">Zusammen: {zusammen}</p>
		{/if}

		{#if geladen && !fehler}
			<Button variant="secondary" class="mt-4" onclick={() => (dialogOffen = true)}>
				<Plus class="h-4 w-4" aria-hidden="true" />
				Andere Auflage zuordnen
			</Button>
		{/if}
	</div>

	<AuflageZuordnenDialog
		bind:open={dialogOffen}
		titelId={formular.id}
		bekannte={auflagen.map((a) => a.id)}
		onZugeordnet={(liste) => (auflagen = liste)}
	/>
{/if}
