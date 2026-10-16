<script>
	import { onMount } from 'svelte';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import { slide } from 'svelte/transition';
	import { useInventur } from './useInventur.svelte.js';
	import InventurStartDialog from './components/InventurStartDialog.svelte';
	import Button from './components/ui/Button.svelte';
	import Suchpille from './components/ui/Suchpille.svelte';
	import FehlbestandBericht from './components/inventur/FehlbestandBericht.svelte';
	import ScanRueckmeldung from './components/inventur/ScanRueckmeldung.svelte';
	import Seitenrahmen from './components/layout/Seitenrahmen.svelte';
	import { formatDatum } from './utils/format.js';
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

			{#if ablauf.offeneLaeufe.length > 0}
				<div class="w-full max-w-4xl mx-auto text-left space-y-2 pt-4">
					<h4 class="text-sm font-semibold text-on-surface-variant">Laufende Inventuren</h4>
					{#each ablauf.offeneLaeufe as session (session.session_id)}
						<div
							class="flex items-center justify-between gap-3 p-3 bg-warning-container text-on-warning-container rounded-lg"
						>
							<div class="min-w-0">
								<div class="font-semibold truncate">{session.label}</div>
								<div class="text-xs">
									{session.erfasst} / {session.erwartet} erfasst · seit {session.gestartet_am?.slice(
										0,
										16
									)}
								</div>
							</div>
							<div class="flex items-center gap-2 shrink-0">
								<Button size="sm" onclick={() => ablauf.setzeLaufFort(session)}>Fortsetzen</Button>
								<Button variant="secondary" size="sm" onclick={() => ablauf.verwerfeLauf(session)}>
									Verwerfen
								</Button>
							</div>
						</div>
					{/each}
				</div>
			{/if}

			<!-- Frühere Inventuren.
			     Der Fehlbestandsbericht entstand bisher nur aus der Antwort des Abschlusses
			     und lebte im Arbeitsspeicher DIESES Browsers: Neu laden — weg. Der Kollege am
			     zweiten Arbeitsplatz, der mit der Liste ins Regal geht, sah ihn nie. Die Daten
			     liegen dauerhaft auf dem Server; hier ist der Weg zurück zu ihnen. Eine verworfene
			     Inventur hat nichts gebucht und keinen Bericht; ihr Bereich ist nicht geprüft. -->
			{#if ablauf.abgeschlosseneInventuren.length > 0}
				<div class="w-full max-w-4xl mx-auto text-left space-y-2 pt-4">
					<h4 class="text-sm font-semibold text-on-surface-variant">Frühere Inventuren</h4>
					{#each ablauf.abgeschlosseneInventuren as inventur (inventur.session_id)}
						<div
							class="flex items-center justify-between gap-3 px-3 py-2.5 border-b border-outline-variant"
						>
							<div class="min-w-0">
								<div class="font-semibold text-on-surface truncate">{inventur.label}</div>
								<div class="text-xs text-on-surface-variant">
									{formatDatum(inventur.abgeschlossen_am)} · {inventur.erfasst} erfasst ·
									{#if inventur.verworfen}
										verworfen
									{:else if inventur.verluste > 0}
										<span class="text-error font-semibold">{inventur.verluste} fehlend</span>
									{:else}
										vollständig
									{/if}
								</div>
							</div>
							{#if !inventur.verworfen}
								<Button
									variant="secondary"
									size="sm"
									class="shrink-0"
									disabled={ablauf.ladeFruehereLaeuft}
									onclick={() => ablauf.zeigeFrueherenFehlbestand(inventur)}
								>
									Fehlbestand
								</Button>
							{/if}
						</div>
					{/each}
				</div>
			{/if}
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
					<div class="text-3xl font-bold text-primary">{ablauf.fortschrittProzent()}%</div>
				</div>
				<!-- Spur und Anzeige wie der lineare Fortschritt in M3 (material-web.dev, Tokens
				     track-color: surface-container-highest, active-indicator-color: primary). -->
				<div class="w-full bg-surface-container-highest rounded-full h-3 overflow-hidden">
					<div
						class="bg-primary h-3 rounded-full transition-all duration-500 ease-out"
						style="width: {ablauf.fortschrittProzent()}%"
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
	state={ablauf}
	onClose={() => {
		ablauf.startOffen = false;
		ablauf.leereFehler();
	}}
	onStart={ablauf.starteInventur}
/>
