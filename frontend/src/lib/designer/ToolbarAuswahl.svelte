<!-- @component ToolbarAuswahl — die drei Einstellungen der Ausweis-Werkstatt:
     welcher Barcode, welcher Kartenhintergrund, welche Vergrößerung.
     Eigene Datei, damit die Werkzeugleiste selbst nur noch die drei Reihen
     zusammensetzt.

     Bis zum 06.08.2026 stand hier zusätzlich eine Klassenauswahl. Sie gehörte nie
     hierher: Dieser Bildschirm gestaltet das Ausweis-Design, und das gilt zentral für
     ALLE Arbeitsplätze. WER gedruckt wird, entscheidet seitdem die Schülerdatei
     (markieren → „Ausweise drucken"), wo auch das Ablaufjahr je Schüler sichtbar ist. -->
<script>
	import Select from '../components/ui/Select.svelte';
	import { AUSWEIS_VORLAGEN } from './ausweisVorlagen.js';

	/**
	 * @type {{
	 *   barcodeType: 'code39'|'qr', onBarcodeType: (t: 'code39'|'qr') => void,
	 *   currentTheme: string, themes: Array<{ value: string, name: string }>,
	 *   setTheme: (v: string) => void,
	 *   onVorlage: (kennung: string) => void,
	 *   zoom: number, onZoom: (v: number) => void
	 * }}
	 */
	let { barcodeType, onBarcodeType, currentTheme, themes, setTheme, onVorlage, zoom, onZoom } =
		$props();

	const BARCODE_TYPEN = [
		{ value: 'code39', label: 'Code39 (1D)' },
		{ value: 'qr', label: 'QR-Code (2D)' }
	];

	// Das Vorlagen-Dropdown ist eine AKTION, kein Zustand: Nach der Wahl springt es auf
	// den Platzhalter zurück, statt „aktiv" zu bleiben — die Vorlage ist danach ja nur
	// noch der Ausgangspunkt für freies Editieren.
	let vorlageWahl = $state('');

	/** @param {string} kennung */
	function vorlageGewaehlt(kennung) {
		onVorlage(kennung);
		vorlageWahl = '';
	}
</script>

<div
	class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-3 bg-slate-50 border border-slate-100 rounded-2xl p-4"
>
	<div class="space-y-1">
		<!-- text-on-surface-variant statt des slate-500 der Nachbarzellen: Die Farb-Ratsche
	     lässt keine NEUEN Paletten-Fundstellen zu; Neues gehört auf die M3-Rollen. -->
		<span class="text-xs font-medium text-on-surface-variant">Design-Vorlage</span>
		<Select
			bind:value={vorlageWahl}
			options={AUSWEIS_VORLAGEN}
			placeholder="Vorlage wählen …"
			onchange={vorlageGewaehlt}
			aria-label="Design-Vorlage"
		/>
	</div>

	<div class="space-y-1">
		<span class="text-xs font-medium text-slate-500">Barcode-Typ</span>
		<Select
			value={barcodeType}
			options={BARCODE_TYPEN}
			onchange={(/** @type {'code39'|'qr'} */ wert) => onBarcodeType(wert)}
			aria-label="Barcode-Typ"
		/>
	</div>

	<div class="space-y-1">
		<span class="text-xs font-medium text-slate-500">Karten-Hintergrund</span>
		<Select
			value={currentTheme}
			options={themes.map((/** @type {any} */ t) => ({ value: t.value, label: t.name }))}
			onchange={(/** @type {string} */ wert) => setTheme(wert)}
			aria-label="Karten-Hintergrund"
		/>
	</div>

	<div class="space-y-1">
		<span class="text-xs font-medium text-slate-500">Zoom</span>
		<div class="flex items-center gap-2">
			<input
				type="range"
				min="80"
				max="300"
				step="5"
				value={zoom}
				oninput={(e) => onZoom(parseInt(/** @type {HTMLInputElement} */ (e.currentTarget).value))}
				class="accent-blue-600 h-1 bg-slate-200 rounded-lg cursor-pointer flex-1"
			/>
			<span class="text-xs font-bold text-blue-600 w-10 text-right">{zoom}%</span>
		</div>
	</div>
</div>
