<!-- @component EtikettBarcodeSchritt — Schritt 2 des Druck-Centers: WELCHE Exemplare aufs
     Blatt kommen. Gedruckt wird nur die Nummer eines Exemplars, das es gibt: Jede Nummer
     vergibt das Programm beim Anlegen (Bestellung, Maske, Übernahme), und eine Nummer, die
     nur auf Papier steht, könnte es später ein zweites Mal vergeben.

     Der Platzhalter im else-Zweig ist Absicht: Ohne ihn spränge die Schrittfolge von
     1 auf 3 und sähe aus wie ein übersprungener Schritt. -->
<script>
	import { etikettenStore } from '../../stores/etiketten.svelte.js';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import Kaestchen from '../ui/Kaestchen.svelte';
	import Suchfeld from '../ui/Suchfeld.svelte';

	// Eine Stelle für den Namen des Schritts: Er steht auch im Platzhalter.
	const SCHRITT = '2. Exemplare wählen';

	// Der Kasten zeigt fünf Zeilen. Was ganz hineinpasst, braucht kein Feld zum Suchen.
	const ZEILEN_IM_KASTEN = 5;

	// Die Zahl nennt alle gewählten Exemplare, auch die, die das Nummernfeld gerade ausblendet.
	const ueberschrift = $derived(
		etikettenStore.laedtExemplare || etikettenStore.auswahl.gesamt === 0
			? 'Exemplare auswählen'
			: `Exemplare auswählen: ${etikettenStore.auswahl.gewaehlt} von ${etikettenStore.auswahl.gesamt}`
	);
	const gesucht = $derived(etikettenStore.exemplarSuche.trim());
	const sichtbar = $derived(etikettenStore.sichtbareExemplare.length);
	const alleText = $derived.by(() => {
		if (!gesucht) return `Alle ${etikettenStore.auswahl.gesamt} Exemplare`;
		return sichtbar === 1 ? 'Der eine Treffer' : `Alle ${sichtbar} Treffer`;
	});
	const ausgesondertText = $derived(
		etikettenStore.ausgesondertAnzahl === 1
			? '1 ausgesondertes Exemplar steht nicht in der Liste.'
			: `${etikettenStore.ausgesondertAnzahl} ausgesonderte Exemplare stehen nicht in der Liste.`
	);
</script>

{#if etikettenStore.gewaehlterTitel}
	<div class="space-y-4 border-b border-outline-variant py-5">
		<h3 class="text-base font-semibold text-on-surface-variant">{SCHRITT}</h3>

		<div class="space-y-2">
			<span class="block text-xs font-medium text-on-surface-variant">{ueberschrift}</span>
			{#if etikettenStore.laedtExemplare}
				<div class="flex items-center justify-center py-4">
					<Ladekreis size="md" />
				</div>
			{:else if etikettenStore.exemplareNichtGeladen}
				<LadeFehler
					titel="Exemplare nicht geladen"
					text="Die Exemplare dieses Titels konnten nicht abgerufen werden. Welche es schon gibt, ist deshalb nicht zu sehen."
					onerneut={etikettenStore.ladeExemplare}
				/>
			{:else if etikettenStore.vorhandeneExemplare.length === 0}
				<p class="text-xs text-on-surface-variant">
					Zu diesem Titel gibt es kein Exemplar, das ein Etikett bekommen kann.
				</p>
			{:else}
				<!-- Ein Kästchen für alle (M3: „A parent checkbox allows for easy selection or
				     deselection of all items“). Es wirkt auf das, was zu sehen ist. -->
				{#if etikettenStore.vorhandeneExemplare.length > 1 && sichtbar > 0}
					<Kaestchen
						etikett={alleText}
						checked={etikettenStore.auswahlSichtbar === 'alle'}
						indeterminate={etikettenStore.auswahlSichtbar === 'teil'}
						onchange={() =>
							etikettenStore.setzeSichtbare(etikettenStore.auswahlSichtbar !== 'alle')}
					/>
				{/if}
				{#if etikettenStore.vorhandeneExemplare.length > ZEILEN_IM_KASTEN}
					<!-- Als Formular, damit die Eingabetaste zählt: Ein Handscanner schickt sie nach
					     der Nummer, und das Exemplar steht dann auf dem Bogen. -->
					<form
						onsubmit={(e) => {
							e.preventDefault();
							etikettenStore.uebernimmNummer();
						}}
					>
						<Suchfeld
							bind:wert={etikettenStore.exemplarSuche}
							platzhalter="Nummer eingeben oder scannen …"
							etikett="Exemplar nach Nummer suchen"
						/>
					</form>
				{/if}
				<div
					class="max-h-40 space-y-1 overflow-y-auto rounded-xl border border-outline-variant p-2"
				>
					{#each etikettenStore.sichtbareExemplare as copy (copy.barcode_id)}
						<label
							class="flex cursor-pointer items-center gap-3 rounded-lg p-1.5 text-xs text-on-surface hover:bg-on-surface/8"
						>
							<Kaestchen bind:checked={copy.checked} />
							<span class="font-bold">{copy.barcode_id}</span>
							<!-- Ohne Notiz steht ein Wort für den Zustand; ein bestelltes Exemplar ist
							     noch nicht eingetroffen und heißt wie in der Buchakte „Bestellt". -->
							<span class="text-label-small text-on-surface-variant font-sans"
								>({copy.zustand_notiz ||
									(copy.im_bestand === false ? 'Bestellt' : 'Neuwertig')})</span
							>
						</label>
					{:else}
						<p class="p-1.5 text-xs text-on-surface-variant">
							Kein Exemplar dieses Titels passt zu „{gesucht}“.
						</p>
					{/each}
				</div>
			{/if}
			{#if !etikettenStore.laedtExemplare && etikettenStore.ausgesondertAnzahl > 0}
				<p class="text-xs text-on-surface-variant">{ausgesondertText}</p>
			{/if}
		</div>
	</div>
{:else}
	<!-- Platzhalter, damit die Schrittfolge nicht von 1 auf 3 springt (wirkt sonst
	     wie ein übersprungener Schritt). Wird aktiv, sobald ein Titel gewählt ist. -->
	<!-- Gedämpft über die Textfarbe, nicht über opacity: Mit 60 % Deckung lag der Kontrast des
	     inaktiven Schritts bei 2,5:1. -->
	<div class="space-y-2 border-b border-outline-variant py-5">
		<h3 class="text-base font-semibold text-on-surface-variant">{SCHRITT}</h3>
		<p class="text-xs text-on-surface-variant">Zuerst oben einen Titel oder Klassensatz wählen.</p>
	</div>
{/if}
