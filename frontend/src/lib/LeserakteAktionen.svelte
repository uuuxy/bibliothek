<script>
	import Button from './components/ui/Button.svelte';
	import Menue from './components/ui/Menue.svelte';
	import AusweisGueltigkeit from './components/AusweisGueltigkeit.svelte';
	import SchuelerDokumente from './components/leser/SchuelerDokumente.svelte';
	import { idStore } from './designer/idDesignerStore.svelte.js';
	import { istKollegium } from './leserArt.js';
	import { FileText, IdCard, ChevronDown, Layers } from '@lucide/svelte';

	/**
	 * @typedef {Object} Props
	 * @property {any} leser
	 * @property {boolean} [darfAuskunft]  manage_students_admin — DSGVO-Auskunft (Art. 15)
	 * @property {boolean} kontoauszugLaedt
	 * @property {boolean} rechnungLaedt
	 * @property {() => void} downloadKontoauszugPDF
	 * @property {() => void} downloadRechnungPDF
	 * @property {(side: 'front'|'back'|'both') => void} onPrint
	 * @property {number|null} gueltigBis        aktuell gewaehltes Ablaufjahr
	 * @property {(jahr: number|null) => void} onGueltigBis
	 */
	/** @type {Props} */
	let {
		leser,
		darfAuskunft = false,
		kontoauszugLaedt,
		rechnungLaedt,
		downloadKontoauszugPDF,
		downloadRechnungPDF,
		onPrint,
		gueltigBis,
		onGueltigBis
	} = $props();

	// Der Ausweis-Druck ist die Primäraktion. Gibt es eine gestaltete Rückseite, bietet
	// ein Material-3-Split-Button (Hauptaktion + Chevron-Menü) die Seitenwahl — ohne die
	// Toolbar mit einem Dauer-Umschalter zuzustellen.
	const hasBack = $derived(idStore.back.elements.some((/** @type {any} */ e) => e.show));

	// Zwei der Dokumente gibt es für einen Kollegen nicht: Kontoauszug und Ersatzforderung
	// lesen die Sicht `schueler` (api/leser/kontoauszug.go, api/schaden/rechnung.go) und
	// beantworten seine ID mit „nicht gefunden". Ein Knopf, der nur scheitern kann, ist
	// schlechter als keiner, und die Ersatzforderung ist ohnehin ein Schreiben an die Eltern
	// eines Schülers. Die DSGVO-Auskunft gibt es für jeden Leser (SchuelerDokumente).
	const kollege = $derived(istKollegium(leser));

	// Ohne Ausweisnummer gibt es keinen Ausweis zu drucken: Die Karte trüge ein leeres
	// Strichcodefeld und wäre an der Theke nicht scanbar. Eingetragen wird die Nummer in dieser
	// Akte unter „Stammdaten & Adresse"; ein Kollege ohne Konto steht in „Benutzer & Rechte" nicht.
	const ohneAusweis = $derived(!leser.barcode_id);

	// Das eine Menü des Hauses (ui/Menue.svelte) mit dem Split-Button als Auslöser. Die
	// Einträge tragen keine Hinweiszeile: M3-Menüs haben eine Zeile je Aktion, und die drei
	// Beschriftungen erklären sich selbst.
	/** @type {import('./components/ui/menueGeometrie.js').Eintrag[]} */
	const seiten = [
		{ id: 'both', text: 'Beides', icon: Layers },
		{ id: 'front', text: 'Nur Vorderseite', icon: IdCard },
		{ id: 'back', text: 'Nur Rückseite', icon: FileText }
	];

	/** @param {string} side */
	function druckeAusweis(side) {
		onPrint(/** @type {'front'|'back'|'both'} */ (side));
	}
</script>

<!-- Alles hier erzeugt ein PDF oder druckt den Ausweis. Die Zeile steht ohne Kasten und ohne
     sichtbare Überschrift auf der Fläche: Jeder Knopf nennt sein Dokument, und eine Karte
     gehört um Inhalt, nicht um Knöpfe. Den Namen der Gruppe hören Vorleseprogramme. -->
<div class="flex flex-wrap gap-3 items-center" role="group" aria-label="Dokumente">
	<!-- Primäraktion: Ausweis drucken. Mit Rückseite → Split-Button mit Seitenwahl. Ohne
	     Ausweisnummer gibt es nichts zu drucken und deshalb auch keine Seiten zu wählen. -->
	<div class="relative">
		{#if hasBack && !ohneAusweis}
			<Menue
				etikett="Ausweisseiten wählen"
				eintraege={seiten}
				onwahl={druckeAusweis}
				ausrichtung="links"
			>
				{#snippet ausloeser({ offen, umschalten })}
					<div class="inline-flex rounded-md shadow-sm">
						<Button type="button" onclick={() => druckeAusweis('both')} class="rounded-r-none">
							<IdCard class="w-4 h-4" />
							Ausweis drucken
						</Button>
						<!-- data-split-pfeil: Die rechte Hälfte ist Teil eines Knopfs mit Beschriftung,
						     kein Symbolknopf (M3, Split button: „The trailing button should always have a
						     menu icon"); die Gates für Symbolknöpfe lassen sie aus. -->
						<Button
							type="button"
							onclick={umschalten}
							aria-haspopup="menu"
							aria-expanded={offen}
							aria-label="Ausweisseiten wählen"
							data-tip="Ausweisseiten wählen"
							data-split-pfeil
							class="rounded-l-none border-l-on-primary/25 px-2.5"
						>
							<ChevronDown class="w-4 h-4 transition-transform {offen ? 'rotate-180' : ''}" />
						</Button>
					</div>
				{/snippet}
			</Menue>
		{:else}
			<span
				class="inline-flex"
				data-tip={ohneAusweis
					? 'Ohne Ausweisnummer lässt sich keine Karte drucken — eingetragen wird sie unter „Stammdaten & Adresse“'
					: undefined}
			>
				<Button variant="primary" onclick={() => druckeAusweis('both')} disabled={ohneAusweis}>
					<IdCard class="w-4 h-4" />
					Ausweis drucken
				</Button>
			</span>
		{/if}
	</div>

	<!-- Das Ablaufjahr steht NEBEN dem Druckknopf, nicht hinter ihm in einem Dialog:
		     Wer druckt, soll sehen, was auf die Karte kommt, bevor die Karte im Drucker
		     liegt. Ein Bestaetigungsdialog haette denselben Wert erst nach dem Klick.

		     Beim Kollegen steht die Wahl nicht: Sein Ausweis traegt keine Gueltigkeit
		     (Kartenseite), weil er mit keinem Schuljahr ablaeuft. Eine Auswahl, die auf der
		     Karte nicht erscheint, waere eine Zusage, die der Druck nicht einloest. -->
	{#if !kollege}
		<AusweisGueltigkeit
			vorschlag={leser.ausweis_gueltig_bis ?? null}
			wert={gueltigBis}
			klasse={leser.klasse ?? ''}
			onWert={onGueltigBis}
		/>
	{/if}

	<SchuelerDokumente
		{leser}
		{kollege}
		{darfAuskunft}
		{kontoauszugLaedt}
		{rechnungLaedt}
		{downloadKontoauszugPDF}
		{downloadRechnungPDF}
	/>
</div>
