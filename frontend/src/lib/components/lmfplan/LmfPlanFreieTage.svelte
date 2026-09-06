<!-- @component LmfPlanFreieTage — die freien Tage des Plans, das, was nur die Schule
     weiß (Brückentag, pädagogischer Tag): ein Chip-Set — Input-Chips je Tag (M3: das
     Entfernen-Symbol „is required and must be used to remove the chip") und dahinter
     der Assist-Chip „Tag freihalten", der das kleine Dialogfenster mit Datum und Grund
     öffnet (LmfPlanEingabeDialog). Bis 06.09.2026 stand hier ein leeres Dauerformular
     mit drei Spalten für eine Eingabe, die ein- bis zweimal im Jahr vorkommt. Darunter
     die Zeile „Übersprungen", die jeden ausgefallenen Werktag des Plan-Zeitraums mit
     Grund nennt — Feiertage eingeschlossen, damit ein fehlender Donnerstag in der
     Tabelle erklärt ist (Peter, 05.09.2026). Ohne eigene Überschrift: Der Abschnitt
     „Zeitraum" (LmfPlanRahmen) trägt sie. -->
<script>
	import Feld from '../ui/Feld.svelte';
	import LmfKlasseChip from './LmfKlasseChip.svelte';
	import LmfPlanEingabeDialog from './LmfPlanEingabeDialog.svelte';
	import { datumKurz, wochentag } from '../../lmfplanDienst.js';

	/** @type {{ tage: import('../../lmfplanDienst.js').FreierTag[], ausfaelle: import('../../lmfplanDienst.js').Ausfall[] }} */
	let { tage = $bindable(), ausfaelle } = $props();

	let offen = $state(false);
	let datum = $state('');
	let grund = $state('');

	function oeffnen() {
		datum = '';
		grund = '';
		offen = true;
	}

	function hinzufuegen() {
		if (!datum) return;
		tage = [...tage.filter((t) => t.datum !== datum), { datum, grund: grund.trim() }].sort((a, b) =>
			a.datum.localeCompare(b.datum)
		);
		offen = false;
	}

	/** @param {import('../../lmfplanDienst.js').FreierTag} t */
	function text(t) {
		return t.grund ? `${datumKurz(t.datum)} ${t.grund}` : datumKurz(t.datum);
	}
</script>

<div class="mt-4 flex flex-wrap items-center gap-2">
	<span class="text-sm text-on-surface-variant">Freie Tage:</span>
	{#if tage.length > 0}
		<span class="contents" data-testid="lmf-freie-tage">
			{#each tage as t (t.datum)}
				<LmfKlasseChip
					name={text(t)}
					onentfernen={() => (tage = tage.filter((x) => x.datum !== t.datum))}
				/>
			{/each}
		</span>
	{/if}
	<LmfKlasseChip name="Tag" verb="freihalten" onklick={oeffnen} />
</div>
{#if ausfaelle.length > 0}
	<p class="mt-4 text-sm text-on-surface-variant" data-testid="lmf-ausfaelle">
		Übersprungen:
		{#each ausfaelle as a, i (a.datum)}{i > 0 ? ' · ' : ''}{wochentag(a.datum)}
			{datumKurz(a.datum)} ({a.grund}){/each}
	</p>
{/if}

<LmfPlanEingabeDialog
	open={offen}
	titel="Tag freihalten"
	aktion="Freihalten"
	gueltig={Boolean(datum)}
	onclose={() => (offen = false)}
	onbestaetigen={hinzufuegen}
>
	<Feld id="lmf-freier-tag-datum" label="Freier Tag" type="date" bind:value={datum} />
	<Feld
		id="lmf-freier-tag-grund"
		label="Grund"
		bind:value={grund}
		placeholder="z. B. Pädagogischer Tag"
	/>
</LmfPlanEingabeDialog>
