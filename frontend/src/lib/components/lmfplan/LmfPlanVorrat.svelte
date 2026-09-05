<!-- @component LmfPlanVorrat — „Nicht im Plan": die Klassen, die keine Zeile haben.
     Ein Klick holt eine Klasse ans Ende des Plans; das Textfeld nimmt Klassen auf, die
     das Vokabular noch nicht kennt („7G1" vor dem August-Import). Was hier liegt, wird
     beim Speichern als ausgelassen gemerkt: Es gilt nicht als „ohne Termin", und der
     Plan des nächsten Jahres lässt es wieder aus — so bleibt die Oberstufe draußen,
     die sich an dieser Schule selbst organisiert (Peter, 05.09.2026). -->
<script>
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import LmfKlasseChip from './LmfKlasseChip.svelte';

	/** @type {{ klassen: string[], onhinein: (klasse: string) => void }} */
	let { klassen, onhinein } = $props();

	let neue = $state('');

	function hinzufuegen() {
		const k = neue.trim();
		if (!k) return;
		onhinein(k);
		neue = '';
	}
</script>

<section aria-labelledby="lmf-vorrat-titel" class="space-y-3">
	<h2 id="lmf-vorrat-titel" class="text-title-medium font-medium text-on-surface">
		Nicht im Plan
		<span class="text-sm font-normal text-on-surface-variant">
			— {klassen.length === 0 ? 'jede Klasse hat eine Zeile' : `${klassen.length} Klassen`}
		</span>
	</h2>
	{#if klassen.length > 0}
		<div class="flex flex-wrap gap-2" data-testid="lmf-vorrat">
			{#each klassen as k (k)}
				<LmfKlasseChip name={k} onklick={() => onhinein(k)} />
			{/each}
		</div>
	{/if}
	<div class="flex max-w-md items-end gap-2">
		<Feld
			id="lmf-plan-weitere-klasse"
			label="Weitere Klasse"
			bind:value={neue}
			placeholder="z. B. 07G1"
			class="flex-1"
			onkeydown={(/** @type {KeyboardEvent} */ e) => {
				if (e.key === 'Enter') {
					e.preventDefault();
					hinzufuegen();
				}
			}}
		/>
		<Button variant="secondary" onclick={hinzufuegen} disabled={!neue.trim()}>In den Plan</Button>
	</div>
</section>
