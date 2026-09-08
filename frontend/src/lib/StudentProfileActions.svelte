<script>
	import Button from './components/ui/Button.svelte';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import { toastStore } from './stores/toastStore.svelte.js';
	import Menue from './components/ui/Menue.svelte';
	import AusweisGueltigkeit from './components/AusweisGueltigkeit.svelte';
	import { apiFetch } from './apiFetch.js';
	import { idStore } from './designer/idDesignerStore.svelte.js';
	import {
		Printer,
		FileText,
		AlertTriangle,
		IdCard,
		ShieldCheck,
		ChevronDown,
		Layers
	} from '@lucide/svelte';

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

	async function downloadDsgvoAuskunft() {
		try {
			const res = await apiFetch(`/api/schueler/${profile.id}/dsgvo-auskunft/pdf`);
			if (res.ok) {
				const blob = await res.blob();
				const url = URL.createObjectURL(blob);
				const a = document.createElement('a');
				a.href = url;
				a.download = `dsgvo-auskunft-${profile.nachname || 'Unbekannt'}-${profile.vorname || 'Unbekannt'}.pdf`;
				document.body.appendChild(a);
				a.click();
				document.body.removeChild(a);
				URL.revokeObjectURL(url);
			} else {
				const text = await res.text();
				console.error('Auskunft Error:', text);
				toastStore.addToast('Fehler beim Herunterladen der Auskunft.', 'error');
			}
		} catch (e) {
			console.error('Netzwerkfehler DSGVO Auskunft:', e);
			toastStore.addToast('Netzwerkfehler beim Herunterladen der Auskunft.', 'error');
		}
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
	{#snippet spinner()}
		<Ladekreis size="sm" />
	{/snippet}

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
				<Button variant="primary" onclick={() => doPrint('both')}>
					<IdCard class="w-4 h-4" />
					Ausweis drucken
				</Button>
			{/if}
		</div>

		<!-- Das Ablaufjahr steht NEBEN dem Druckknopf, nicht hinter ihm in einem Dialog:
		     Wer druckt, soll sehen, was auf die Karte kommt, bevor die Karte im Drucker
		     liegt. Ein Bestaetigungsdialog haette denselben Wert erst nach dem Klick. -->
		<AusweisGueltigkeit
			vorschlag={profile.ausweis_gueltig_bis ?? null}
			wert={gueltigBis}
			klasse={profile.klasse ?? ''}
			onWert={onGueltigBis}
		/>

		<!-- Kontoauszug: das (einzige) Ausleih-Dokument als archivierbares Server-PDF. -->
		<Button
			variant="secondary"
			onclick={downloadKontoauszugPDF}
			disabled={kontoauszugPdfLoading || !(profile.entliehene_buecher?.length > 0)}
		>
			{#if kontoauszugPdfLoading}{@render spinner()}{:else}<Printer
					class="w-4 h-4 text-blue-600"
				/>{/if}
			Kontoauszug
		</Button>

		<!-- Ersatzforderung: Rechnung an die Eltern über offene Schadensfälle.

		     data-tip am UMSCHLAG, nicht am Knopf: Ein disabled-Element bekommt keine
		     Zeigerereignisse — weder für den nativen title noch für die Blase dieses
		     Projekts (tooltip.js hört delegiert auf mouseover). Die Begründung stand
		     also im Code und erreichte genau in dem Zustand niemanden, in dem man sie
		     braucht: wenn der Knopf grau ist und man wissen will, warum. Der Umschlag
		     fängt das Ereignis ab, das der graue Knopf durchlässt. -->
		<span
			class="inline-flex"
			data-tip={!profile.has_open_damages
				? 'Kein offener Schadensfall — eine Ersatzforderung gibt es erst, wenn ein Schaden erfasst ist'
				: 'Ersatzforderung über offene Schäden drucken'}
		>
			<Button
				variant="secondary"
				onclick={downloadRechnungPDF}
				disabled={rechnungPdfLoading || !profile.has_open_damages}
			>
				{#if rechnungPdfLoading}{@render spinner()}{:else}<AlertTriangle
						class="w-4 h-4 text-rose-600"
					/>{/if}
				Ersatzforderung
			</Button>
		</span>

		{#if darfAuskunft}
			<Button
				variant="secondary"
				onclick={downloadDsgvoAuskunft}
				title="DSGVO-Auskunft (Art. 15) als PDF exportieren"
			>
				<ShieldCheck class="w-4 h-4 text-slate-500" />
				DSGVO-Auskunft
			</Button>
		{/if}
	</div>
</div>
