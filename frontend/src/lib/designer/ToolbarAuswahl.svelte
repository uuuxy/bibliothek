<!-- @component ToolbarAuswahl — die vier Einstellungen der Ausweis-Werkstatt:
     welche Klasse, welcher Barcode, welcher Kartenhintergrund, welche Vergrößerung.
     Eigene Datei, damit die Werkzeugleiste selbst nur noch die drei Reihen
     zusammensetzt. -->
<script>
	import Select from '../components/ui/Select.svelte';

	/**
	 * @type {{
	 *   classesList: string[], selectedKlasse: string, onKlasse: (k: string) => void,
	 *   loadingStudents: boolean,
	 *   barcodeType: 'code39'|'qr', onBarcodeType: (t: 'code39'|'qr') => void,
	 *   currentTheme: string, themes: Array<{ value: string, name: string }>,
	 *   setTheme: (v: string) => void,
	 *   zoom: number, onZoom: (v: number) => void
	 * }}
	 */
	let {
		classesList,
		selectedKlasse,
		onKlasse,
		loadingStudents,
		barcodeType,
		onBarcodeType,
		currentTheme,
		themes,
		setTheme,
		zoom,
		onZoom
	} = $props();

	const BARCODE_TYPEN = [
		{ value: 'code39', label: 'Code39 (1D)' },
		{ value: 'qr', label: 'QR-Code (2D)' }
	];
</script>

<div
	class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-3 bg-slate-50 border border-slate-100 rounded-2xl p-4"
>
	<div class="space-y-1">
		<span class="text-xs font-medium text-slate-500">Klasse</span>
		{#if classesList.length > 0}
			<Select
				value={selectedKlasse}
				options={classesList.map((/** @type {string} */ kl) => ({
					value: kl,
					label: `Klasse ${kl}`
				}))}
				onchange={(/** @type {string} */ wert) => onKlasse(wert)}
				aria-label="Klasse für die Ausweise"
			/>
		{:else}
			<div class="text-xs text-slate-400 font-medium py-2">
				{loadingStudents ? 'Lade…' : 'Keine Klassen'}
			</div>
		{/if}
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
