<!-- @component LmfPlanRahmen — Abschnitt „Zeitraum": der Anker des Plans und die freien
     Tage der Schule (LmfPlanFreieTage). Der Anker hängt an der Art (Peter, 06.09.2026:
     „es endet immer am gleichen Tag — Donnerstags vor den Ferien zur vierten Stunde"):
     Der Büchertausch vor den Sommerferien ENDET (letzter Tag, Ende am letzten Tag), die
     Reihenfolge fließt rückwärts davor, und der Server sagt, wo sie beginnt; die
     Bücherausgabe nach den Ferien BEGINNT (erster Tag, Beginn am ersten Tag). Beide Tage
     sind aus den Sommerferien Hessen vorbelegt; fehlt das Jahr in der Tabelle, steht es
     hier und der Tag wird von Hand eingetragen. Alles Weitere — Wochentage, Feiertage,
     Folgetage — rechnet der Server; hier steht nichts, was er auch weiß.

     M3-Aufbau eines Abschnitts (Typografie-Rollen): Title-Medium, darunter EIN Satz
     Supporting Text in On-Surface-Variant (hier der gerechnete: Ferien und Beginn),
     dann der Inhalt auf EINEM Raster mit vier Spalten: drei Felder und die freien
     Tage als Chip-Spalte — eine Zeile statt zwei (06.09.2026).

     Das Auswahlfeld hat keine eigene Beschriftung; damit es neben dem Datumsfeld auf
     derselben Zeile steht, bekommt es dieselben drei Subgrid-Zeilen wie Feld.svelte
     (Beschriftung, Feld, Hinweis) — sonst rutscht das Datumsfeld eine Zeile tiefer
     (flasch3, 23.08. und 05.09.2026). -->
<script>
	import Feld from '../ui/Feld.svelte';
	import Select from '../ui/Select.svelte';
	import LmfPlanFreieTage from './LmfPlanFreieTage.svelte';
	import { STUNDEN, datumKurz, wochentag } from '../../lmfplanDienst.js';

	/** @type {{ art: string, entwurf: import('../../lmfplanDienst.js').PlanEntwurf, ausfaelle: import('../../lmfplanDienst.js').Ausfall[], beginn: { datum: string, stunde: number } | null, sommerferien: import('../../lmfplanDienst.js').Sommerferien | null }} */
	let { art, entwurf = $bindable(), ausfaelle, beginn, sommerferien } = $props();

	const ende = $derived(art === 'rueckgabe');
	// Die Felder des Ankers — je Art ein anderes Paar im Entwurf.
	const tagFeld = $derived(ende ? 'letzter_tag' : 'erster_tag');
	const stundeFeld = $derived(ende ? 'letzte_stunde' : 'startstunde');

	// Die Ankerstunde kann nicht hinter dem Tagesende liegen — die Auswahl endet dort.
	const stunden = $derived(STUNDEN.filter((s) => s <= entwurf.stunden_je_tag));
	$effect(() => {
		if (entwurf[stundeFeld] > entwurf.stunden_je_tag) entwurf[stundeFeld] = entwurf.stunden_je_tag;
	});

	// Ein Textknoten je Satz — kein Umbruch in der Vorlage, der im Text als Leerraum landet.
	const beginnText = $derived(
		ende && beginn
			? `Der Plan beginnt ${wochentag(beginn.datum)}, ${datumKurz(beginn.datum)} in der ${beginn.stunde}. Stunde.`
			: ''
	);
	const ferienText = $derived.by(() => {
		if (!sommerferien) return '';
		if (!sommerferien.bekannt)
			// Zeigt auf die EINSTELLUNG, nicht auf ein Programm-Update (Peter, 06.09.2026:
			// „das muss doch dann irgendwo eingestellt werden"). Die Selbstprüfung sagt
			// seit 47c1d461 dasselbe; hier stand bis zum Rasterdurchgang noch die alte,
			// für den Betreiber nutzlose Antwort.
			return `Die Sommerferien ${sommerferien.jahr} sind noch nicht hinterlegt — unter Einstellungen → LUSD & Versetzung → Sommerferien eintragen, oder den ${ende ? 'letzten' : 'ersten'} Tag hier von Hand setzen.`;
		return `Sommerferien ${sommerferien.jahr}: ${datumKurz(sommerferien.von)} bis ${datumKurz(sommerferien.bis)}.`;
	});
</script>

<section aria-labelledby="lmf-zeitraum-titel">
	<h2 id="lmf-zeitraum-titel" class="text-title-medium font-medium text-on-surface">Zeitraum</h2>
	<!-- Der EINE Satz unter dem Titel ist der dynamische (Ferien, gerechneter Beginn);
	     die Regel selbst („endet am Donnerstag vor den Ferien") steht als Hinweis unter
	     dem Datumsfeld und im Handbuch — als eigener Satz kostete sie eine Zeile (Peter,
	     06.09.2026: „verschenken wir im oberen Bereich nicht viel Platz?"). -->
	{#if ferienText || beginnText}
		<p class="mt-1 max-w-3xl text-sm text-on-surface-variant" data-testid="lmf-zeitraum-hinweis">
			{[ferienText, beginnText].filter(Boolean).join(' ')}
		</p>
	{/if}
	<div class="mt-4 grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-2 xl:grid-cols-4">
		<Feld
			id="lmf-plan-anker-tag"
			label={ende ? 'Letzter Tag' : 'Erster Tag'}
			type="date"
			bind:value={entwurf[tagFeld]}
			hint={ende ? 'Donnerstag vor den Sommerferien' : 'Erster Schultag nach den Sommerferien'}
		/>
		<div class="row-span-3 grid grid-rows-subgrid gap-y-1.5">
			<label for="lmf-plan-anker-stunde" class="text-sm font-medium text-on-surface-variant"
				>{ende ? 'Ende am letzten Tag' : 'Beginn am ersten Tag'}</label
			>
			<Select
				id="lmf-plan-anker-stunde"
				bind:value={entwurf[stundeFeld]}
				options={stunden.map((s) => ({ value: s, label: `${s}. Stunde` }))}
			/>
		</div>
		<div class="row-span-3 grid grid-rows-subgrid gap-y-1.5">
			<label for="lmf-plan-stunden" class="text-sm font-medium text-on-surface-variant"
				>Stunden je Tag</label
			>
			<Select
				id="lmf-plan-stunden"
				bind:value={entwurf.stunden_je_tag}
				options={STUNDEN.map((s) => ({ value: s, label: `${s} Stunden` }))}
			/>
		</div>
		<LmfPlanFreieTage bind:tage={entwurf.freie_tage} {ausfaelle} />
	</div>
</section>
