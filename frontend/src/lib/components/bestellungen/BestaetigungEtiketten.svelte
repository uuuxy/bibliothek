<!-- @component BestaetigungEtiketten — der Etikettenblock der Lieferanten-Seite: kleine Etiketten
     im Raster seiner Bögen oder das große Lernmittel-Etikett. Beide gibt es zu jeder Bestellung,
     gleich aus welchem Topf; der Händler wählt, was er druckt. -->
<script>
	import Button from '../ui/Button.svelte';
	import { anzahlText } from '../../utils/format.js';
	import Select from '../ui/Select.svelte';
	import Icon from '../ui/Icon.svelte';

	/** @type {{ bestellung: any, token: string, formatId?: string, geoeffneteGroesse?: string }} */
	let { bestellung, token, formatId = $bindable(''), geoeffneteGroesse = $bindable('') } = $props();

	const rasterBeschriftung = 'Bogenraster der kleinen Etiketten';

	/** @param {'klein' | 'gross'} groesse */
	function etikettenOeffnen(groesse) {
		geoeffneteGroesse = groesse;
		// Das Raster gilt nur für die kleinen Etiketten. Das große Lernmittel-Etikett hat
		// ein festes Raster (4 Stück auf A4) und wird ausgeschnitten, nicht auf
		// vorgestanzte Bögen gedruckt.
		const parameterText =
			groesse === 'klein' && formatId ? `?format=${encodeURIComponent(formatId)}` : '';
		window.open(
			`/api/public/bestellung/${encodeURIComponent(token)}/etiketten/${groesse}${parameterText}`,
			'_blank',
			'noopener'
		);
	}
</script>

<div class="bg-surface-container-lowest rounded-xl p-8 shadow-level1">
	<h2 class="text-on-surface text-title-medium">Etiketten drucken</h2>
	<p class="text-on-surface-variant mt-1 text-sm">
		Beide Bögen enthalten dieselben Barcodes wie der Anhang der Bestellmail. Sie wählen nur das
		Format.
	</p>

	{#if bestellung.etiketten_formate?.length}
		<div class="mt-5 max-w-md space-y-1.5">
			<label for="etikettenformat" class="text-on-surface-variant block text-sm font-medium">
				{rasterBeschriftung}
			</label>
			<Select
				id="etikettenformat"
				bind:value={formatId}
				options={bestellung.etiketten_formate.map((/** @type {any} */ f) => ({
					value: f.id,
					label: f.name,
					zusatz: `${anzahlText(f.pro_seite, 'Etikett', 'Etiketten')}, ${f.spalten} × ${f.zeilen}`
				}))}
				aria-label={rasterBeschriftung}
			/>
			<p class="text-on-surface-variant text-xs">
				Passend zu den Etikettenbögen in Ihrem Drucker. Gilt nicht für die großen
				Lernmittel-Etiketten, die zu viert auf einem A4-Blatt liegen und ausgeschnitten werden.
			</p>
		</div>
	{/if}

	<div class="mt-4 flex flex-wrap gap-3">
		<Button size="lg" variant="secondary" onclick={() => etikettenOeffnen('klein')}>
			{#if geoeffneteGroesse === 'klein'}<Icon name="check" />{/if}
			Kleine Etiketten auf A4-Bogen
		</Button>
		<Button size="lg" variant="secondary" onclick={() => etikettenOeffnen('gross')}>
			{#if geoeffneteGroesse === 'gross'}<Icon name="check" />{/if}
			Große Lernmittel-Etiketten, 4 je A4-Blatt
		</Button>
	</div>
	{#if geoeffneteGroesse}
		<p class="text-on-surface-variant mt-3 text-xs">
			Der Bogen wurde in einem neuen Tab geöffnet. Erscheint er nicht, ist er vom Browser blockiert
			worden, dann bitte den Knopf erneut drücken.
		</p>
	{/if}
</div>
