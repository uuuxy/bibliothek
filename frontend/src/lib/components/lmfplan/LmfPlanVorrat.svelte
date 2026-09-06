<!-- @component LmfPlanVorrat — „Noch nicht im Plan": die Klassen ohne Zeile, seit dem
     06.09.2026 als Chip-Zeile ÜBER der Tabelle statt als eigener Abschnitt darunter
     (Peter: „es steht oben was und unten was"). Ein Klick auf den Assist-Chip plant die
     Klasse an ihren Platz — hinter die letzte Klasse desselben Jahrgangs und Zweigs
     (lmfplanZeilen.einordnen), nicht ans Ende; ziehen auf eine Zeile setzt sie davor.
     Zwei Gruppen: Offen sichtbar ist nur, was ohne Regel fehlt (die neue Klasse nach
     dem LUSD-Import). Was die Regel oder der gespeicherte Plan bewusst auslässt (die
     Oberstufe, die sich an dieser Schule selbst organisiert), steht eingeklappt hinter
     „11 Klassen bleiben draußen" — sonst böte die Seite jedes Jahr elf Chips an, die
     niemand will. „Andere Klasse eintragen" öffnet das kleine Dialogfenster für eine
     Klasse, die das Vokabular noch nicht kennt („07G1" vor dem August-Import). Was hier
     liegt, wird beim Speichern als ausgelassen gemerkt und gilt nicht als „ohne Termin". -->
<script>
	import { ChevronDown, ChevronRight } from '@lucide/svelte';
	import Feld from '../ui/Feld.svelte';
	import LmfKlasseChip from './LmfKlasseChip.svelte';
	import LmfPlanEingabeDialog from './LmfPlanEingabeDialog.svelte';

	/** @type {{ klassen: string[], draussen: (klasse: string) => boolean, marker: ReturnType<typeof import('../../lmfplanDienst.js').klassenMarker>, onhinein: (klasse: string) => void }} */
	let { klassen, draussen, marker, onhinein } = $props();

	const offene = $derived(klassen.filter((k) => !draussen(k)));
	const bewusst = $derived(klassen.filter((k) => draussen(k)));
	let zeigeBewusst = $state(false);
	let dialogOffen = $state(false);
	let neue = $state('');

	function eintragen() {
		const k = neue.trim();
		if (!k) return;
		onhinein(k);
		neue = '';
		dialogOffen = false;
	}
</script>

<!-- EINE Zeile: fehlende Klassen, „Andere Klasse eintragen", „… bleiben draußen".
     Vorher zwei Zeilen übereinander (06.09.2026, Peter: „verschenken wir im oberen
     Bereich nicht viel Platz?"). -->
<div class="mt-4 flex flex-wrap items-center gap-2" data-testid="lmf-vorrat">
	{#if offene.length > 0}
		<span class="text-sm text-on-surface-variant">Noch nicht im Plan:</span>
		{#each offene as k (k)}
			<LmfKlasseChip
				name={k}
				hinweis={marker.ohneSchueler(k) ? 'ohne Schüler' : ''}
				ziehbar
				onklick={() => onhinein(k)}
			/>
		{/each}
	{/if}
	<LmfKlasseChip
		name="Andere Klasse"
		verb="eintragen"
		onklick={() => {
			neue = '';
			dialogOffen = true;
		}}
	/>
	{#if bewusst.length > 0}
		<button
			type="button"
			class="inline-flex h-9 cursor-pointer items-center gap-1 rounded-full px-3 text-sm font-medium text-on-surface-variant hover:bg-surface-container"
			aria-expanded={zeigeBewusst}
			onclick={() => (zeigeBewusst = !zeigeBewusst)}
		>
			{#if zeigeBewusst}
				<ChevronDown class="h-4 w-4" aria-hidden="true" />
			{:else}
				<ChevronRight class="h-4 w-4" aria-hidden="true" />
			{/if}
			{bewusst.length === 1 ? 'Eine Klasse bleibt' : `${bewusst.length} Klassen bleiben`} draußen
		</button>
	{/if}
</div>
{#if bewusst.length > 0 && zeigeBewusst}
	<div class="mt-2 flex flex-wrap gap-2" data-testid="lmf-vorrat-draussen">
		{#each bewusst as k (k)}
			<LmfKlasseChip
				name={k}
				hinweis={marker.ohneSchueler(k) ? 'ohne Schüler' : ''}
				ziehbar
				onklick={() => onhinein(k)}
			/>
		{/each}
	</div>
{/if}

<LmfPlanEingabeDialog
	open={dialogOffen}
	titel="Andere Klasse eintragen"
	aktion="In den Plan"
	gueltig={neue.trim() !== ''}
	onclose={() => (dialogOffen = false)}
	onbestaetigen={eintragen}
>
	<Feld
		id="lmf-plan-weitere-klasse"
		label="Klasse"
		bind:value={neue}
		placeholder="z. B. 07G1"
		hint="Eine Klasse, die es im Programm noch nicht gibt — sie kommt mit dem LUSD-Import."
	/>
</LmfPlanEingabeDialog>
