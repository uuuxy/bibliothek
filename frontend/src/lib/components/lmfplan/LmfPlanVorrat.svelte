<!-- @component LmfPlanVorrat — „Nicht im Plan": die Klassen, die keine Zeile haben.
     Ein Klick auf den Assist-Chip plant eine Klasse ans Ende; das Textfeld nimmt
     Klassen auf, die das Vokabular noch nicht kennt („7G1" vor dem August-Import). Was
     hier liegt, wird beim Speichern als ausgelassen gemerkt: Es gilt nicht als „ohne
     Termin", und der Plan des nächsten Jahres lässt es wieder aus — so bleibt die
     Oberstufe draußen, die sich an dieser Schule selbst organisiert (Peter, 05.09.2026).
     Abschnittsaufbau wie „Zeitraum": Titel, ein Satz Supporting Text, Inhalt. -->
<script>
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import LmfKlasseChip from './LmfKlasseChip.svelte';

	/** @type {{ klassen: string[], marker: ReturnType<typeof import('../../lmfplanDienst.js').klassenMarker>, onhinein: (klasse: string) => void }} */
	let { klassen, marker, onhinein } = $props();

	let neue = $state('');

	function hinzufuegen() {
		const k = neue.trim();
		if (!k) return;
		onhinein(k);
		neue = '';
	}
</script>

<section aria-labelledby="lmf-vorrat-titel">
	<h2 id="lmf-vorrat-titel" class="text-title-medium font-medium text-on-surface">Nicht im Plan</h2>
	<p class="mt-1 max-w-3xl text-sm text-on-surface-variant">
		{#if klassen.length === 0}
			Jede Klasse hat eine Zeile.
		{:else}
			{klassen.length} Klassen ohne Zeile; was hier bleibt, gilt als bewusst ausgelassen.
		{/if}
	</p>
	{#if klassen.length > 0}
		<div class="mt-4 flex flex-wrap gap-2" data-testid="lmf-vorrat">
			{#each klassen as k (k)}
				<LmfKlasseChip
					name={k}
					hinweis={marker.ohneSchueler(k) ? 'ohne Schüler' : ''}
					onklick={() => onhinein(k)}
				/>
			{/each}
		</div>
	{/if}
	<div class="mt-4 grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-3">
		<Feld
			id="lmf-plan-weitere-klasse"
			label="Weitere Klasse"
			bind:value={neue}
			placeholder="z. B. 07G1"
			onkeydown={(/** @type {KeyboardEvent} */ e) => {
				if (e.key === 'Enter') {
					e.preventDefault();
					hinzufuegen();
				}
			}}
		/>
		<div class="row-span-3 grid grid-rows-subgrid gap-y-1.5">
			<span aria-hidden="true"></span>
			<Button
				variant="secondary"
				onclick={hinzufuegen}
				disabled={!neue.trim()}
				class="justify-self-start"
			>
				In den Plan
			</Button>
		</div>
	</div>
</section>
