<!--
  @component
  SchuelerDokumente — Kontoauszug, Ersatzforderung und DSGVO-Auskunft.

  Die ersten beiden gehören einem Kollegen nicht: Sie lesen die Sicht `schueler`
  (api/leser/kontoauszug.go, api/schaden/rechnung.go) und beantworten seine ID mit „nicht
  gefunden"; die Ersatzforderung ist ohnehin ein Schreiben an die Eltern. Die DSGVO-Auskunft
  gibt es für jeden Leser. Beieinander statt dreimal dieselbe Bedingung.
-->
<script>
	import Button from '../ui/Button.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import { snackbars } from '../../stores/snackbars.svelte.js';
	import { apiFetch, leseApiFehler } from '../../apiFetch.js';
	import { zeigePdf, PDF_BLOCKIERT } from '../../utils/pdfImFenster.js';
	import { Printer, TriangleAlert, ShieldCheck } from '@lucide/svelte';

	/**
	 * @type {{
	 *   leser: any,
	 *   kollege?: boolean,
	 *   darfAuskunft?: boolean,
	 *   kontoauszugLaedt: boolean,
	 *   rechnungLaedt: boolean,
	 *   zeigeKontoauszug: () => void,
	 *   zeigeRechnung: () => void
	 * }}
	 */
	let {
		leser,
		kollege = false,
		darfAuskunft = false,
		kontoauszugLaedt,
		rechnungLaedt,
		zeigeKontoauszug,
		zeigeRechnung
	} = $props();

	async function zeigeDsgvoAuskunft() {
		try {
			const res = await zeigePdf(() => apiFetch(`/api/schueler/${leser.id}/dsgvo-auskunft/pdf`));
			if (!res) {
				snackbars.addToast(PDF_BLOCKIERT, 'warning');
			} else if (!res.ok) {
				snackbars.addToast(
					await leseApiFehler(res, 'Die Auskunft ließ sich nicht erzeugen.'),
					'error'
				);
			}
		} catch (e) {
			console.error(e);
			snackbars.addToast('Netzwerkfehler beim Erzeugen der Auskunft.', 'error');
		}
	}
</script>

{#snippet spinner()}
	<Ladekreis size="sm" />
{/snippet}

{#if !kollege}
	<!-- Kontoauszug: das (einzige) Ausleih-Dokument als archivierbares Server-PDF. -->
	<Button
		variant="secondary"
		onclick={zeigeKontoauszug}
		disabled={kontoauszugLaedt || !(leser.entliehene_buecher?.length > 0)}
	>
		{#if kontoauszugLaedt}{@render spinner()}{:else}<Printer class="w-4 h-4 text-primary" />{/if}
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
		data-tip={!leser.has_open_damages
			? 'Kein offener Schadensfall — eine Ersatzforderung gibt es erst, wenn ein Schaden erfasst ist'
			: 'Ersatzforderung über offene Schäden drucken'}
	>
		<Button
			variant="secondary"
			onclick={zeigeRechnung}
			disabled={rechnungLaedt || !leser.has_open_damages}
		>
			{#if rechnungLaedt}{@render spinner()}{:else}<TriangleAlert class="w-4 h-4 text-error" />{/if}
			Ersatzforderung
		</Button>
	</span>
{/if}

{#if darfAuskunft}
	<Button
		variant="secondary"
		onclick={zeigeDsgvoAuskunft}
		title="DSGVO-Auskunft (Art. 15) als PDF anzeigen"
	>
		<ShieldCheck class="w-4 h-4 text-on-surface-variant" />
		DSGVO-Auskunft
	</Button>
{/if}
