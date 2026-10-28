<script>
	import { onMount } from 'svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import { slide } from 'svelte/transition';
	import { useInventur } from './useInventur.svelte.js';
	import InventurStartDialog from './InventurStartDialog.svelte';
	import Button from '../ui/Button.svelte';
	import Suchpille from '../ui/Suchpille.svelte';
	import FehlbestandBericht from './FehlbestandBericht.svelte';
	import InventurLaeufe from './InventurLaeufe.svelte';
	import ScanRueckmeldung from './ScanRueckmeldung.svelte';
	import Seitenrahmen from '../layout/Seitenrahmen.svelte';
	import { ClipboardCheck, Plus, ScanBarcode } from '@lucide/svelte';

	const ablauf = useInventur();

	// $state, nicht `let`: Das Feld wird per bind:this gefüllt und unten in einem $effect
	// gelesen. Es entsteht erst, wenn die Inventur läuft; ohne $state liefe der Effekt nur
	// einmal davor, das Feld bliebe ohne Fokus, und jeder Scan ginge ins Leere.
	let barcodeInputEl = $state();

	$effect(() => {
		if (ablauf.status === 'active' && barcodeInputEl && !ablauf.scanLaeuft) {
			barcodeInputEl.focus();
		}
	});

	onMount(async () => {
		await ablauf.ladeSignaturen();
		await ablauf.ladeFaecher();
		await ablauf.ladeOffeneLaeufe();
		await ablauf.ladeFruehereInventuren();
	});

	function focusInput() {
		if (barcodeInputEl) barcodeInputEl.focus();
	}

	// Anteil der erfassten an den erwarteten Exemplaren, für die Zahl und den Balken.
	const prozent = $derived(
		ablauf.stand.erwartet === 0
			? 0
			: Math.min(100, Math.round((ablauf.stand.erfasst / ablauf.stand.erwartet) * 100))
	);
</script>

<Seitenrahmen>
	<!-- Steht GANZ OBEN und ueberlebt das Zuruecksetzen: Der Bericht ist das Ergebnis der
	     Arbeit, nicht eine Randnotiz. Vorher endete eine Inventur mit einer Zahl im Toast,
	     die drei Sekunden spaeter weg war. -->
	{#if ablauf.fehlbestand.length > 0}
		<FehlbestandBericht
			eintraege={ablauf.fehlbestand}
			etikett={ablauf.fehlbestandBezeichnung}
			onSchliessen={ablauf.fehlbestandSchliessen}
			onGefunden={ablauf.fehlbestandGefunden}
			onEndgueltigLoeschen={ablauf.fehlbestandEndgueltigLoeschen}
		/>
	{/if}

	{#if ablauf.status === 'idle'}
		<div class="p-12 text-center flex flex-col items-center justify-center space-y-6">
			<!-- Neutral wie der Leerzustand der Abgängerliste (AbgaengerTabelle.svelte): Die
			     Hauptfarbe bleibt dem einen gefüllten Knopf darunter. -->
			<div
				class="w-20 h-20 rounded-full bg-surface-container-low border border-outline-variant text-on-surface-variant flex items-center justify-center"
			>
				<ClipboardCheck class="w-10 h-10" aria-hidden="true" />
			</div>
			<div>
				<h3 class="text-xl font-bold text-on-surface">Keine Inventur aktiv</h3>
				<p class="text-on-surface-variant mt-2 max-w-md mx-auto">
					Starte einen neuen Inventur-Lauf. Du kannst entweder die gesamte Bibliothek prüfen oder
					gezielt nach einer bestimmten Signatur / Kategorie scannen.
				</p>
			</div>
			<Button size="lg" onclick={() => (ablauf.startOffen = true)} class="px-6">
				<Plus class="w-5 h-5" aria-hidden="true" />
				<span>Neue Bestandsprüfung starten</span>
			</Button>

			{#if ablauf.fehler}
				<div
					class="w-full max-w-lg mx-auto p-3 bg-warning-container rounded-lg text-sm text-on-warning-container"
				>
					{ablauf.fehler}
				</div>
			{/if}

			<InventurLaeufe {ablauf} />
		</div>
	{:else}
		<div class="space-y-6">
			<!-- Progress & Stats -->
			<div class="p-6">
				<div class="flex justify-between items-end mb-4">
					<div>
						<span class="text-sm font-semibold text-on-surface-variant">Aktueller Fortschritt</span>
						<div class="text-2xl font-bold text-on-surface mt-1">
							{ablauf.stand.erfasst} / {ablauf.stand.erwartet}
							<span class="text-base font-medium text-on-surface-variant">erfasst</span>
						</div>
					</div>
					<div class="text-3xl font-bold text-primary">{prozent}%</div>
				</div>
				<!-- Spur und Anzeige wie der lineare Fortschritt in M3 (material-web.dev, Tokens
				     track-color: surface-container-highest, active-indicator-color: primary). -->
				<div class="w-full bg-surface-container-highest rounded-full h-3 overflow-hidden">
					<div
						class="bg-primary h-3 rounded-full transition-all duration-500 ease-out"
						style="width: {prozent}%"
					></div>
				</div>
			</div>

			<!-- Dieselbe 48-px-Suchpille wie die Ausleihe-Omnibox: Werkzeug der Seite, kein
			     Datenfeld in einer Leiste. Das Feld bleibt während einer Anfrage bedienbar: Der
			     nächste Scan wird eingereiht (useInventur), gesperrt ginge er verloren. -->
			<form
				onsubmit={(e) => {
					e.preventDefault();
					ablauf.verarbeiteScan(ablauf.barcodeEingabe, focusInput);
				}}
			>
				<Suchpille
					id="inventur-scan"
					bind:element={barcodeInputEl}
					bind:wert={ablauf.barcodeEingabe}
					etikett="Barcode scannen"
					platzhalter="Barcode scannen..."
				>
					{#snippet nachlaufend()}
						{#if ablauf.scanLaeuft}
							<Ladekreis size="md" />
						{:else}
							<ScanBarcode class="h-5 w-5 shrink-0 text-on-surface-variant" aria-hidden="true" />
						{/if}
					{/snippet}
				</Suchpille>
			</form>
			<!-- Die Rückmeldung zum letzten Scan (ScanRueckmeldung.svelte: Erfolg, Warnung,
			     Fehler auf eigenen Farbrollen). Das Einblenden bleibt hier, am {#if}. -->
			{#if ablauf.letzterScan}
				<div transition:slide>
					<ScanRueckmeldung scan={ablauf.letzterScan} />
				</div>
			{/if}

			<div class="pt-8 border-t border-outline-variant flex justify-end">
				<Button
					variant="danger"
					size="lg"
					onclick={() => ablauf.abschliessenNachRueckfrage(focusInput)}
					class="px-6"
				>
					Inventur abschließen
				</Button>
			</div>
		</div>
	{/if}
</Seitenrahmen>

<InventurStartDialog
	open={ablauf.startOffen}
	{ablauf}
	onClose={() => {
		ablauf.startOffen = false;
		ablauf.leereFehler();
	}}
	onStart={ablauf.starteInventur}
/>
