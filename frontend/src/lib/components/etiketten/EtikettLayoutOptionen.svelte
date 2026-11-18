<!-- @component EtikettLayoutOptionen — Schritt 3 des Druck-Centers: Wie das Blatt
     bedruckt wird (Format, Startfeld auf angebrochenen Bögen, Barcode-Art). Steht
     getrennt, weil es die einzige Gruppe ist, die nichts über den INHALT der Etiketten
     sagt. -->
<script>
	import { etikettenStore } from '../../stores/etiketten.svelte.js';
	import Select from '../ui/Select.svelte';
	import Feld from '../ui/Feld.svelte';
	import { BUCH_ETIKETT_FORMATE, etikettFormat } from './etikettformate.js';
	const BARCODE_AUSGABE = [
		// Der Wert heißt 'code39', gedruckt wird Code 128 (internal/pkg/strichcode). Der gespeicherte
		// Wert bleibt, weil er in den Ausweis-Layouts steht; die Beschriftung sagt, was
		// tatsächlich herauskommt.
		{ value: 'code39', label: 'Strichcode', zusatz: 'Code 128' },
		{ value: 'qr', label: 'QR-Code' }
	];
	// Das große Lernmittel-Etikett hat ein festes Raster und immer den Strichcode: Startfeld und
	// Barcode-Art entfallen, statt wirkungslos stehen zu bleiben.
	const grossesEtikett = $derived(etikettFormat(etikettenStore.formatId).ohneStartfeld === true);
</script>

<div class="py-5 space-y-4 border-b border-outline-variant">
	<h3 class="text-title-medium text-on-surface-variant">3. Druckeinstellungen</h3>

	<div class="space-y-3.5">
		<div class="space-y-1.5">
			<span class="text-xs font-medium text-on-surface-variant block">Etikettenformat</span>
			<Select
				bind:value={etikettenStore.formatId}
				options={BUCH_ETIKETT_FORMATE}
				aria-label="Etikettenformat"
			/>
			{#if grossesEtikett}
				<p class="text-xs text-on-surface-variant mt-1">
					Zum Ausschneiden, nicht für Bücher der Schülerbücherei
				</p>
			{/if}
		</div>

		{#if !grossesEtikett}
			<div class="space-y-1.5">
				<span class="text-xs font-medium text-on-surface-variant block"
					>Startposition auf dem A4-Bogen</span
				>
				<div class="flex items-center gap-2">
					<Feld
						type="number"
						min="1"
						max={etikettenStore.felderJeBogen}
						bind:value={etikettenStore.startFeld}
						aria-label="Startposition auf dem A4-Bogen"
						feld="w-24"
					/>
					<span class="text-label-small text-on-surface-variant"
						>max. {etikettenStore.felderJeBogen}</span
					>
				</div>
				<p class="text-xs text-on-surface-variant mt-1">
					Für angebrochene Bögen: Gibt an, auf welchem Feld der Druck starten soll
				</p>
			</div>

			<div class="space-y-1.5">
				<span class="text-xs font-medium text-on-surface-variant block">Barcode-Ausgabe</span>
				<Select
					bind:value={etikettenStore.barcodeArt}
					options={BARCODE_AUSGABE}
					aria-label="Barcode-Ausgabe"
				/>
			</div>
		{/if}
	</div>
</div>
