<!-- @component WerkzeugleisteAuswahl — die drei Einstellungen der Ausweis-Werkstatt:
     welcher Barcode, welcher Kartenhintergrund, welche Vergrößerung.
     Eigene Datei, damit die Werkzeugleiste selbst nur die drei Reihen zusammensetzt.

     Keine Klassenauswahl: Dieser Bildschirm gestaltet das Ausweis-Design, und das gilt
     zentral für alle Arbeitsplätze. Wer gedruckt wird, entscheidet die Schülerdatei
     (markieren → „Ausweise drucken"), wo auch das Ablaufjahr je Schüler sichtbar ist. -->
<script>
	import Select from '../components/ui/Select.svelte';
	import VorlagenGalerie from './VorlagenGalerie.svelte';

	/**
	 * @type {{
	 *   barcodeType: 'code39'|'qr', onBarcodeType: (t: 'code39'|'qr') => void,
	 *   currentTheme: string, themes: Array<{ value: string, name: string }>,
	 *   setTheme: (v: string) => void,
	 *   onVorlage: (kennung: string) => void,
	 *   zoom: number, onZoom: (v: number) => void,
	 *   gesperrt?: boolean
	 * }}
	 */
	let {
		barcodeType,
		onBarcodeType,
		currentTheme,
		themes,
		setTheme,
		onVorlage,
		zoom,
		onZoom,
		gesperrt = false
	} = $props();

	const BARCODE_TYPEN = [
		// Wert historisch 'code39', gedruckt wird Code 128 — siehe EtikettLayoutOptionen.
		{ value: 'code39', label: 'Strichcode (Code 128)' },
		{ value: 'qr', label: 'QR-Code (2D)' }
	];
</script>

<div
	class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-3 bg-surface border border-outline-variant rounded-xl p-4"
>
	<div class="space-y-1">
		<span class="text-sm font-medium text-on-surface-variant">Design-Vorlage</span>
		<VorlagenGalerie {onVorlage} {gesperrt} />
	</div>

	<div class="space-y-1">
		<span class="text-sm font-medium text-on-surface-variant">Barcode-Typ</span>
		<Select
			value={barcodeType}
			options={BARCODE_TYPEN}
			onchange={(/** @type {'code39'|'qr'} */ wert) => onBarcodeType(wert)}
			disabled={gesperrt}
			aria-label="Barcode-Typ"
		/>
	</div>

	<div class="space-y-1">
		<span class="text-sm font-medium text-on-surface-variant">Karten-Hintergrund</span>
		<Select
			value={currentTheme}
			options={themes.map((/** @type {any} */ t) => ({ value: t.value, label: t.name }))}
			onchange={(/** @type {string} */ wert) => setTheme(wert)}
			disabled={gesperrt}
			aria-label="Karten-Hintergrund"
		/>
	</div>

	<div class="space-y-1">
		<span class="text-sm font-medium text-on-surface-variant">Zoom</span>
		<div class="flex items-center gap-2">
			<input
				type="range"
				min="80"
				max="300"
				step="5"
				value={zoom}
				oninput={(e) => onZoom(parseInt(/** @type {HTMLInputElement} */ (e.currentTarget).value))}
				class="accent-primary h-1 bg-secondary-container rounded-lg cursor-pointer flex-1"
			/>
			<span class="text-sm font-bold text-primary w-10 text-right">{zoom}%</span>
		</div>
	</div>
</div>
