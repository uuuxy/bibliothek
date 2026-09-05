<!-- @component LmfPlanRahmen — Abschnitt „Zeitraum": was der Planer vorgibt (erster
     Tag, Startstunde am ersten Tag — der Donnerstag im Juni begann in der 3. Stunde —,
     Stunden je Tag) und die freien Tage der Schule (LmfPlanFreieTage). Alles Weitere —
     Wochentage, Feiertage, Folgetage — rechnet der Server; hier steht nichts, was er
     auch weiß.

     M3-Aufbau eines Abschnitts (Typografie-Rollen): Title-Medium, darunter EIN Satz
     Supporting Text in On-Surface-Variant, dann der Inhalt auf einem Raster. Beide
     Zeilen des Rasters (Rahmen, freier Tag) teilen dieselben drei Spalten — dieselbe
     Feldbreite, dieselbe Grundlinie.

     Das Auswahlfeld hat keine eigene Beschriftung; damit es neben dem Datumsfeld auf
     derselben Zeile steht, bekommt es dieselben drei Subgrid-Zeilen wie Feld.svelte
     (Beschriftung, Feld, Hinweis) — sonst rutscht das Datumsfeld eine Zeile tiefer
     (flasch3, 23.08. und 05.09.2026). -->
<script>
	import Feld from '../ui/Feld.svelte';
	import Select from '../ui/Select.svelte';
	import LmfPlanFreieTage from './LmfPlanFreieTage.svelte';
	import { STUNDEN } from '../../lmfplanDienst.js';

	/** @type {{ ersterTag: string, startstunde: number, stundenJeTag: number, tage: import('../../lmfplanDienst.js').FreierTag[], ausfaelle: import('../../lmfplanDienst.js').Ausfall[] }} */
	let {
		ersterTag = $bindable(),
		startstunde = $bindable(),
		stundenJeTag = $bindable(),
		tage = $bindable(),
		ausfaelle
	} = $props();

	// Die Startstunde kann nicht hinter dem Tagesende liegen — die Auswahl endet dort.
	const startstunden = $derived(STUNDEN.filter((s) => s <= stundenJeTag));
	$effect(() => {
		if (startstunde > stundenJeTag) startstunde = stundenJeTag;
	});
</script>

<section aria-labelledby="lmf-zeitraum-titel">
	<h2 id="lmf-zeitraum-titel" class="text-title-medium font-medium text-on-surface">Zeitraum</h2>
	<p class="mt-1 max-w-3xl text-sm text-on-surface-variant">
		Der Plan läuft ab dem ersten Tag über die Schultage. Wochenenden und gesetzliche Feiertage
		überspringt er von selbst; bewegliche Ferientage, pädagogische Tage und Brückentage der Schule
		trägst du als freie Tage ein.
	</p>
	<div class="mt-4 grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-3">
		<Feld id="lmf-plan-erster-tag" label="Erster Tag" type="date" bind:value={ersterTag} />
		<div class="row-span-3 grid grid-rows-subgrid gap-y-1.5">
			<label for="lmf-plan-startstunde" class="text-sm font-medium text-on-surface-variant"
				>Beginn am ersten Tag</label
			>
			<Select
				id="lmf-plan-startstunde"
				bind:value={startstunde}
				options={startstunden.map((s) => ({ value: s, label: `${s}. Stunde` }))}
			/>
		</div>
		<div class="row-span-3 grid grid-rows-subgrid gap-y-1.5">
			<label for="lmf-plan-stunden" class="text-sm font-medium text-on-surface-variant"
				>Stunden je Tag</label
			>
			<Select
				id="lmf-plan-stunden"
				bind:value={stundenJeTag}
				options={STUNDEN.map((s) => ({ value: s, label: `${s} Stunden` }))}
			/>
		</div>
	</div>
	<LmfPlanFreieTage bind:tage {ausfaelle} />
</section>
