<!-- @component AusweisGueltigkeit — das Ablaufjahr des Ausweises, vorgeschlagen und
     überschreibbar.

     Der Vorschlag kommt vom Server (schueler.ausweis_gueltig_bis) und folgt dem
     Bildungsgang: Hauptschulzweig bis Jahrgang 9, Real- und Gymnasialzweig bis 10,
     Oberstufe bis 13 (Regel in internal/ausweis).

     Warum trotzdem ein Eingabefeld: Ein Ausweis wird auch für Fälle gedruckt, die keine
     Regel kennt — Wiederholer, Zweigwechsler, ein Ersatzausweis kurz vor dem Wechsel.
     Wer dafür in die Einstellungen oder gar in den Ausweis-Designer müsste, ändert das
     Layout für ALLE Arbeitsplätze, nur um eine Karte anders zu datieren.

     Die Änderung gilt deshalb ausdrücklich nur für diesen einen Druck: Sie wird nirgends
     gespeichert und ist nach dem Verlassen des Profils wieder weg. -->
<script>
	import { RotateCcw, CalendarClock, AlertTriangle } from '@lucide/svelte';

	/**
	 * @typedef {Object} Props
	 * @property {number|null} vorschlag  aus der Klasse gerechnet; null = nicht ableitbar
	 * @property {number|null} wert       aktuell gewähltes Jahr
	 * @property {string} klasse          nur für die Begründung, wenn nichts ableitbar ist
	 * @property {(jahr: number|null) => void} onWert
	 */
	/** @type {Props} */
	let { vorschlag, wert, klasse = '', onWert } = $props();

	const abweichend = $derived(vorschlag !== null && wert !== vorschlag);
	// Ohne Vorschlag UND ohne Eingabe stünde "31.07.–" auf der Karte. Das ist kein
	// Schönheitsfehler: Eine Karte ohne Ablaufdatum wird an der Ausleihe abgewiesen.
	const fehlt = $derived(wert === null || Number.isNaN(wert));

	/** @param {Event} e */
	function uebernehmen(e) {
		const roh = /** @type {HTMLInputElement} */ (e.currentTarget).value.trim();
		if (roh === '') {
			onWert(null);
			return;
		}
		const jahr = Number.parseInt(roh, 10);
		onWert(Number.isNaN(jahr) ? null : jahr);
	}
</script>

<div class="flex flex-col gap-1.5">
	<div
		class="inline-flex h-9 items-center gap-2 rounded-lg border bg-white px-3
			{fehlt ? 'border-amber-400' : 'border-slate-200'}"
	>
		<CalendarClock class="h-4 w-4 shrink-0 {fehlt ? 'text-amber-600' : 'text-slate-400'}" />
		<span class="text-sm whitespace-nowrap text-slate-500">Gültig bis 31.07.</span>
		<input
			type="number"
			inputmode="numeric"
			min="2000"
			max="2099"
			value={wert ?? ''}
			oninput={uebernehmen}
			aria-label="Ablaufjahr des Ausweises"
			class="w-16 border-0 bg-transparent p-0 text-sm font-semibold text-slate-800
				focus:outline-none focus:ring-0"
		/>
		{#if abweichend}
			<button
				type="button"
				onclick={() => onWert(vorschlag)}
				aria-label="Auf den vorgeschlagenen Wert {vorschlag} zurücksetzen"
				data-tip="Zurück auf {vorschlag} (aus der Klasse gerechnet)"
				class="-mr-1 rounded p-1 text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
			>
				<RotateCcw class="h-3.5 w-3.5" />
			</button>
		{/if}
	</div>

	{#if vorschlag === null}
		<p class="flex items-start gap-1.5 text-xs leading-relaxed text-amber-700">
			<AlertTriangle class="mt-0.5 h-3.5 w-3.5 shrink-0" />
			<span>
				Aus der Klasse {klasse ? `„${klasse}“` : ''} lässt sich kein Ablaufjahr ableiten — bitte
				eintragen. Geraten wird hier nichts: Ein falsches Datum fällt erst auf, wenn die Karte an
				der Ausleihe abgewiesen wird.
			</span>
		</p>
	{:else if abweichend}
		<p class="text-xs text-slate-400">
			Abweichend vom Vorschlag ({vorschlag}). Gilt nur für diesen Druck.
		</p>
	{/if}
</div>
