<!-- @component LabelBarcodeSchritt — Schritt 2 des Druck-Centers: WELCHE Barcodes
     aufs Blatt kommen. Zwei Wege, die einander ausschließen — vorhandene Exemplare
     abhaken oder eine Reihe neuer Nummern erzeugen.

     Der Platzhalter im else-Zweig ist Absicht: Ohne ihn spränge die Schrittfolge von
     1 auf 3 und sähe aus wie ein übersprungener Schritt. -->
<script>
	import { labelStore } from '../../stores/labels.svelte.js';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import Feld from '../ui/Feld.svelte';
	import Kaestchen from '../ui/Kaestchen.svelte';
	import Segmente from '../ui/Segmente.svelte';
	import Suchfeld from '../ui/Suchfeld.svelte';

	const ARTEN = [
		{ wert: 'existing', text: 'Vorhandene Exemplare' },
		{ wert: 'new', text: 'Neue Barcodes' }
	];

	// Der Kasten zeigt fünf Zeilen. Was ganz hineinpasst, braucht kein Feld zum Suchen.
	const ZEILEN_IM_KASTEN = 5;

	// Die Zahl nennt alle gewählten Exemplare, auch die, die das Nummernfeld gerade ausblendet.
	const ueberschrift = $derived(
		labelStore.loadingCopies || labelStore.auswahl.gesamt === 0
			? 'Exemplare auswählen'
			: `Exemplare auswählen: ${labelStore.auswahl.gewaehlt} von ${labelStore.auswahl.gesamt}`
	);
	const gesucht = $derived(labelStore.exemplarSuche.trim());
	const sichtbar = $derived(labelStore.sichtbareExemplare.length);
	const alleText = $derived.by(() => {
		if (!gesucht) return `Alle ${labelStore.auswahl.gesamt} Exemplare`;
		return sichtbar === 1 ? 'Der eine Treffer' : `Alle ${sichtbar} Treffer`;
	});
	const ausgesondertText = $derived(
		labelStore.ausgesondertAnzahl === 1
			? '1 ausgesondertes Exemplar steht nicht in der Liste.'
			: `${labelStore.ausgesondertAnzahl} ausgesonderte Exemplare stehen nicht in der Liste.`
	);
</script>

{#if labelStore.selectedTitle}
	<div class="space-y-4 border-b border-outline-variant py-5">
		<h3 class="text-base font-semibold text-on-surface-variant">2. Barcodes generieren</h3>

		<Segmente
			etikett="Woher die Barcodes kommen"
			optionen={ARTEN}
			wert={labelStore.generationMode}
			onwahl={(art) => (labelStore.generationMode = art)}
			klasse="w-full"
		/>

		{#if labelStore.generationMode === 'existing'}
			<div class="space-y-2">
				<span class="block text-xs font-medium text-on-surface-variant">{ueberschrift}</span>
				{#if labelStore.loadingCopies}
					<div class="flex items-center justify-center py-4">
						<Ladekreis size="md" />
					</div>
				{:else if labelStore.exemplareNichtGeladen}
					<LadeFehler
						titel="Exemplare nicht geladen"
						text="Die Exemplare dieses Titels konnten nicht abgerufen werden. Welche es schon gibt, ist deshalb nicht zu sehen."
						onerneut={labelStore.ladeExemplare}
					/>
				{:else if labelStore.existingCopies.length === 0}
					<p class="text-xs text-on-surface-variant">
						Zu diesem Titel gibt es kein Exemplar, das ein Etikett bekommen kann.
					</p>
				{:else}
					<!-- Ein Kästchen für alle (M3: „A parent checkbox allows for easy selection or
					     deselection of all items“). Es wirkt auf das, was zu sehen ist. -->
					{#if labelStore.existingCopies.length > 1 && sichtbar > 0}
						<Kaestchen
							label={alleText}
							checked={labelStore.auswahlSichtbar === 'alle'}
							indeterminate={labelStore.auswahlSichtbar === 'teil'}
							onchange={() => labelStore.setzeSichtbare(labelStore.auswahlSichtbar !== 'alle')}
						/>
					{/if}
					{#if labelStore.existingCopies.length > ZEILEN_IM_KASTEN}
						<!-- Als Formular, damit die Eingabetaste zählt: Ein Handscanner schickt sie nach
						     der Nummer, und das Exemplar steht dann auf dem Bogen. -->
						<form
							onsubmit={(e) => {
								e.preventDefault();
								labelStore.uebernimmNummer();
							}}
						>
							<Suchfeld
								bind:wert={labelStore.exemplarSuche}
								platzhalter="Nummer eingeben oder scannen …"
								etikett="Exemplar nach Nummer suchen"
							/>
						</form>
					{/if}
					<div
						class="max-h-40 space-y-1 overflow-y-auto rounded-xl border border-outline-variant p-2"
					>
						{#each labelStore.sichtbareExemplare as copy (copy.barcode_id)}
							<label
								class="flex cursor-pointer items-center gap-3 rounded-lg p-1.5 text-xs text-on-surface hover:bg-on-surface/8"
							>
								<Kaestchen bind:checked={copy.checked} />
								<span class="font-bold">{copy.barcode_id}</span>
								<span class="text-label-small text-on-surface-variant font-sans"
									>({copy.zustand_notiz || 'Neuwertig'})</span
								>
							</label>
						{:else}
							<p class="p-1.5 text-xs text-on-surface-variant">
								Kein Exemplar dieses Titels passt zu „{gesucht}“.
							</p>
						{/each}
					</div>
				{/if}
				{#if !labelStore.loadingCopies && labelStore.ausgesondertAnzahl > 0}
					<p class="text-xs text-on-surface-variant">{ausgesondertText}</p>
				{/if}
			</div>
		{:else}
			<!-- Generating new sequential labels -->
			<div class="grid grid-cols-2 gap-3">
				<Feld label="Menge" type="number" min="1" max="100" bind:value={labelStore.newQuantity} />
				<Feld label="Start-Ziffer (B-)" type="number" min="1" bind:value={labelStore.newStartNum} />
			</div>
		{/if}
	</div>
{:else}
	<!-- Platzhalter, damit die Schrittfolge nicht von 1 auf 3 springt (wirkt sonst
	     wie ein übersprungener Schritt). Wird aktiv, sobald ein Titel gewählt ist. -->
	<!-- Gedämpft über die Textfarbe, nicht über opacity: Mit 60 % Deckung lag der Kontrast des
	     inaktiven Schritts bei 2,5:1. -->
	<div class="space-y-2 border-b border-outline-variant py-5">
		<h3 class="text-base font-semibold text-on-surface-variant">2. Barcodes generieren</h3>
		<p class="text-xs text-on-surface-variant">Zuerst oben einen Titel oder Klassensatz wählen.</p>
	</div>
{/if}
