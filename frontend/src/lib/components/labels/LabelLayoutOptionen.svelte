<!-- @component LabelLayoutOptionen — Schritt 3 des Druck-Centers: Wie das Blatt
     bedruckt wird (Format, Startfeld auf angebrochenen Bögen, Barcode-Art,
     Hilfsrahmen). Steht getrennt, weil es die einzige Gruppe ist, die nichts über
     den INHALT der Etiketten sagt. -->
<script>
	import { labelStore } from '../../stores/labels.svelte.js';
	import Select from '../ui/Select.svelte';
	import Feld from '../ui/Feld.svelte';
	import { ETIKETT_FORMATE } from '../../etikettformate.js';
	const BARCODE_AUSGABE = [
		{ value: 'code39', label: 'Code39 (1D Standard)' },
		{ value: 'qr', label: 'QR-Code (2D)' }
	];
</script>

<div class="py-5 space-y-4 border-b border-slate-200">
	<h3 class="text-sm font-semibold text-slate-500">3. Layout-Optionen</h3>

	<div class="space-y-3.5">
		<div class="space-y-1.5">
			<span class="text-xs font-medium text-slate-500 block">Etikettenformat</span>
			<Select
				bind:value={labelStore.formatId}
				options={ETIKETT_FORMATE}
				aria-label="Etikettenformat"
			/>
		</div>

		<div class="space-y-1.5">
			<span class="text-xs font-medium text-slate-500 block">Startposition auf dem A4-Bogen</span>
			<div class="flex items-center gap-2">
				<Feld
					type="number"
					min="1"
					max={labelStore.maxPositions}
					bind:value={labelStore.startPosition}
					aria-label="Startposition auf dem A4-Bogen"
					feld="w-24"
				/>
				<span class="text-label-small text-slate-400">max. {labelStore.maxPositions}</span>
			</div>
			<p class="text-xs text-slate-400 mt-1">
				Für angebrochene Bögen: Gibt an, auf welchem Feld der Druck starten soll.
			</p>
		</div>

		<div class="space-y-1.5">
			<span class="text-xs font-medium text-slate-500 block">Barcode-Ausgabe</span>
			<Select
				bind:value={labelStore.barcodeType}
				options={BARCODE_AUSGABE}
				aria-label="Barcode-Ausgabe"
			/>
		</div>

		<label class="flex items-center space-x-3 text-xs text-slate-705 cursor-pointer select-none">
			<input
				type="checkbox"
				bind:checked={labelStore.labelBorder}
				class="accent-blue-600 w-4 h-4 rounded border-slate-200 bg-white"
			/>
			<span>Hilfsrahmen auf dem Etikett zeichnen</span>
		</label>
	</div>
</div>
