<script>
	import Button from './components/ui/Button.svelte';
	import Menue from './components/ui/Menue.svelte';
	import AusweisGueltigkeit from './components/AusweisGueltigkeit.svelte';
	import SchuelerDokumente from './components/students/SchuelerDokumente.svelte';
	import { idStore } from './designer/idDesignerStore.svelte.js';
	import { istKollegium } from './leserArt.js';
	import { FileText, IdCard, ChevronDown, Layers } from '@lucide/svelte';

	/**
	 * @typedef {Object} Props
	 * @property {any} profile
	 * @property {boolean} [darfAuskunft]  manage_students_admin — DSGVO-Auskunft (Art. 15)
	 * @property {boolean} kontoauszugPdfLoading
	 * @property {boolean} rechnungPdfLoading
	 * @property {() => void} downloadKontoauszugPDF
	 * @property {() => void} downloadRechnungPDF
	 * @property {(side: 'front'|'back'|'both') => void} onPrint
	 * @property {number|null} gueltigBis        aktuell gewaehltes Ablaufjahr
	 * @property {(jahr: number|null) => void} onGueltigBis
	 */
	/** @type {Props} */
	let {
		profile,
		darfAuskunft = false,
		kontoauszugPdfLoading,
		rechnungPdfLoading,
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

	// Drei der Dokumente gibt es für einen Kollegen NICHT: Kontoauszug,
	// Ersatzforderung und DSGVO-Auskunft lesen alle die Sicht `schueler` (api/print.go,
	// api/dsgvo_auskunft.go) und beantworten seine ID mit „nicht gefunden". Ein Knopf,
	// der nur scheitern kann, ist schlechter als keiner — und die Ersatzforderung ist
	// ohnehin ein Schreiben an die Eltern eines Schülers.
	const kollege = $derived(istKollegium(profile));

	// Ohne Ausweisnummer gibt es keinen Ausweis zu drucken: Die Karte trüge ein leeres
	// Strichcodefeld und wäre an der Theke nicht scanbar. Eingetragen wird die Nummer in
	// „Benutzer & Rechte" (Kollegium) bzw. beim Anlegen (Schüler).
	const ohneAusweis = $derived(!profile.barcode_id);

	// Seit 07.09.2026 das eine Menü des Hauses (ui/Menue.svelte) mit dem Split-Button als
	// Auslöser. Die Hinweiszeilen der Einträge („Foto & Ausweisdaten") sind weg: M3-Menüs
	// tragen eine Zeile je Aktion, und die drei Beschriftungen erklären sich selbst.
	/** @type {import('./components/ui/menueGeometrie.js').Eintrag[]} */
	const seiten = [
		{ id: 'both', text: 'Beides', icon: Layers },
		{ id: 'front', text: 'Nur Vorderseite', icon: IdCard },
		{ id: 'back', text: 'Nur Rückseite', icon: FileText }
	];

	/** @param {string} side */
	function doPrint(side) {
		onPrint(/** @type {'front'|'back'|'both'} */ (side));
	}
</script>

<!-- Nur noch Dokumente: alles hier erzeugt ein PDF und lässt sich wegwerfen.
     Der Kasten hiess bis zum 08.08.2026 „Dokumente & Aktionen" und trug als einzige
     Aktion mit Folgen die Schülersperre — das „&" im Titel war das Eingeständnis,
     dass zwei Kategorien in einer Kiste lagen. Die Sperre steht jetzt bei dem
     Zustand, den sie umschaltet (StudentProfileCard, Konto-Status). -->
<div class="bg-slate-50 border border-slate-200 rounded-2xl p-4 shadow-sm flex flex-col gap-3">
	<h4 class="text-xs font-medium text-slate-500 flex items-center gap-1.5">
		<FileText class="w-3.5 h-3.5" />
		Dokumente
	</h4>
	<div class="flex flex-wrap gap-3 items-center">
		<!-- Primäraktion: Ausweis drucken. Mit Rückseite → Split-Button mit Seitenwahl. -->
		<div class="relative">
			{#if hasBack}
				<Menue
					etikett="Ausweisseiten wählen"
					eintraege={seiten}
					onwahl={doPrint}
					ausrichtung="links"
				>
					{#snippet ausloeser({ offen, umschalten })}
						<div class="inline-flex rounded-md shadow-sm">
							<Button type="button" onclick={() => doPrint('both')} class="rounded-r-none">
								<IdCard class="w-4 h-4" />
								Ausweis drucken
							</Button>
							<Button
								type="button"
								onclick={umschalten}
								aria-haspopup="menu"
								aria-expanded={offen}
								aria-label="Ausweisseiten wählen"
								class="rounded-l-none border-l-white/25 px-2.5"
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
						? 'Ohne Ausweisnummer lässt sich keine Karte drucken — die Nummer steht in „Benutzer & Rechte"'
						: undefined}
				>
					<Button variant="primary" onclick={() => doPrint('both')} disabled={ohneAusweis}>
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
		     (CardFace), weil er mit keinem Schuljahr ablaeuft. Eine Auswahl, die auf der
		     Karte nicht erscheint, waere eine Zusage, die der Druck nicht einloest. -->
		{#if !kollege}
			<AusweisGueltigkeit
				vorschlag={profile.ausweis_gueltig_bis ?? null}
				wert={gueltigBis}
				klasse={profile.klasse ?? ''}
				onWert={onGueltigBis}
			/>

			<SchuelerDokumente
				{profile}
				{darfAuskunft}
				{kontoauszugPdfLoading}
				{rechnungPdfLoading}
				{downloadKontoauszugPDF}
				{downloadRechnungPDF}
			/>
		{/if}
	</div>
</div>
