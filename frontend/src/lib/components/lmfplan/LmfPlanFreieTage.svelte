<!-- @component LmfPlanFreieTage — die freien Tage des Plans: Datum und Grund für das,
     was nur die Schule weiß (Brückentag, pädagogischer Tag); darunter als Input-Chips
     (M3: das Entfernen-Symbol „is required and must be used to remove the chip") und
     die Zeile „Übersprungen", die jeden ausgefallenen Werktag des Plan-Zeitraums mit
     Grund nennt — Feiertage eingeschlossen, damit ein fehlender Donnerstag in der
     Tabelle erklärt ist (Peter, 05.09.2026). Ohne eigene Überschrift: Der Abschnitt
     „Zeitraum" (LmfPlanRahmen) trägt sie, das Raster ist dasselbe. -->
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

<div class="mt-4 grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-3">
	<Feld id="lmf-freier-tag-datum" label="Freier Tag" type="date" bind:value={datum} />
	<Feld
		id="lmf-freier-tag-grund"
		label="Grund"
		bind:value={grund}
		placeholder="z. B. Pädagogischer Tag"
		onkeydown={(/** @type {KeyboardEvent} */ e) => {
			if (e.key === 'Enter') {
				e.preventDefault();
				hinzufuegen();
			}
		}}
	/>
	<!-- Der Knopf steht in der Feldzeile des Subgrids, nicht in der Beschriftungszeile. -->
	<div class="row-span-3 grid grid-rows-subgrid gap-y-1.5">
		<span aria-hidden="true"></span>
		<Button variant="secondary" onclick={hinzufuegen} disabled={!datum} class="justify-self-start">
			Tag freihalten
		</Button>
	</div>
</div>
{#if tage.length > 0}
	<div class="mt-4 flex flex-wrap gap-2" data-testid="lmf-freie-tage">
		{#each tage as t (t.datum)}
			<LmfKlasseChip
				name={text(t)}
				onentfernen={() => (tage = tage.filter((x) => x.datum !== t.datum))}
			/>
		{/each}
	</div>
{/if}
{#if ausfaelle.length > 0}
	<p class="mt-4 text-sm text-on-surface-variant" data-testid="lmf-ausfaelle">
		Übersprungen:
		{#each ausfaelle as a, i (a.datum)}{i > 0 ? ' · ' : ''}{wochentag(a.datum)}
			{datumKurz(a.datum)} ({a.grund}){/each}
	</p>
{/if}
