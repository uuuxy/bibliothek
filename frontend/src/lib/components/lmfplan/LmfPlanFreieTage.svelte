<!-- @component LmfPlanFreieTage — die Tage, an denen der Plan nicht läuft. Wochenenden
     und die gesetzlichen Feiertage Hessens (Fronleichnam!) überspringt der Server von
     selbst; hier trägt die Bibliothek ein, was nur die Schule weiß: bewegliche
     Ferientage, pädagogische Tage, den Brückentag. Darunter steht, was der Server im
     Plan-Zeitraum tatsächlich übersprungen hat, mit Grund — damit ein fehlender
     Donnerstag in der Tabelle erklärt ist (Peter, 05.09.2026: „manchmal gibt es ja auch
     noch gesetzliche Feiertage"). -->
<script>
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import LmfKlasseChip from './LmfKlasseChip.svelte';
	import { datumKurz, wochentag } from '../../lmfplanDienst.js';

	/** @type {{ tage: import('../../lmfplanDienst.js').FreierTag[], ausfaelle: import('../../lmfplanDienst.js').Ausfall[] }} */
	let { tage = $bindable(), ausfaelle } = $props();

	let datum = $state('');
	let grund = $state('');

	function hinzufuegen() {
		if (!datum) return;
		tage = [...tage.filter((t) => t.datum !== datum), { datum, grund: grund.trim() }].sort((a, b) =>
			a.datum.localeCompare(b.datum)
		);
		datum = '';
		grund = '';
	}

	/** @param {import('../../lmfplanDienst.js').FreierTag} t */
	function text(t) {
		return t.grund ? `${datumKurz(t.datum)} ${t.grund}` : datumKurz(t.datum);
	}
</script>

<section aria-labelledby="lmf-freie-tage-titel" class="space-y-3">
	<h2 id="lmf-freie-tage-titel" class="text-title-medium font-medium text-on-surface">
		Freie Tage
		<span class="text-sm font-normal text-on-surface-variant">
			— Wochenenden und gesetzliche Feiertage überspringt der Plan von selbst; hier stehen
			bewegliche Ferientage, pädagogische Tage, Brückentage.
		</span>
	</h2>
	{#if tage.length > 0}
		<div class="flex flex-wrap gap-2" data-testid="lmf-freie-tage">
			{#each tage as t (t.datum)}
				<LmfKlasseChip
					name={text(t)}
					onentfernen={() => (tage = tage.filter((x) => x.datum !== t.datum))}
				/>
			{/each}
		</div>
	{/if}
	<div class="flex max-w-2xl flex-wrap items-end gap-2">
		<Feld
			id="lmf-freier-tag-datum"
			label="Freier Tag"
			type="date"
			bind:value={datum}
			class="w-44"
		/>
		<Feld
			id="lmf-freier-tag-grund"
			label="Grund"
			bind:value={grund}
			placeholder="z. B. Pädagogischer Tag"
			class="flex-1"
			onkeydown={(/** @type {KeyboardEvent} */ e) => {
				if (e.key === 'Enter') {
					e.preventDefault();
					hinzufuegen();
				}
			}}
		/>
		<Button variant="secondary" onclick={hinzufuegen} disabled={!datum}>Tag freihalten</Button>
	</div>
	{#if ausfaelle.length > 0}
		<p class="text-sm text-on-surface-variant" data-testid="lmf-ausfaelle">
			Übersprungen:
			{#each ausfaelle as a, i (a.datum)}{i > 0 ? ' · ' : ''}{wochentag(a.datum)}
				{datumKurz(a.datum)} ({a.grund}){/each}
		</p>
	{/if}
</section>
