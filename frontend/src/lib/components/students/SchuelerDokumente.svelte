<!--
  @component
  SchuelerDokumente — Kontoauszug, Ersatzforderung und DSGVO-Auskunft.

  Eigene Datei, weil diese drei einem KOLLEGEN nicht gehören: Sie lesen alle die Sicht
  `schueler` (api/print.go, api/dsgvo_auskunft.go) und beantworten seine ID mit „nicht
  gefunden". Beieinander statt dreimal dieselbe Bedingung — und StudentProfileActions
  bleibt unter der 200-Zeilen-Grenze.
-->
<script>
	import Button from '../ui/Button.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import { apiFetch } from '../../apiFetch.js';
	import { Printer, AlertTriangle, ShieldCheck } from '@lucide/svelte';

	/**
	 * @type {{
	 *   profile: any,
	 *   darfAuskunft?: boolean,
	 *   kontoauszugPdfLoading: boolean,
	 *   rechnungPdfLoading: boolean,
	 *   downloadKontoauszugPDF: () => void,
	 *   downloadRechnungPDF: () => void
	 * }}
	 */
	let {
		profile,
		darfAuskunft = false,
		kontoauszugPdfLoading,
		rechnungPdfLoading,
		downloadKontoauszugPDF,
		downloadRechnungPDF
	} = $props();

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
			console.error(e);
			toastStore.addToast('Netzwerkfehler beim Herunterladen der Auskunft.', 'error');
		}
	}
</script>

{#snippet spinner()}
	<Ladekreis size="sm" />
{/snippet}

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
