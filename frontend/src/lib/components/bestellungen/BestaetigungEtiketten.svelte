<!-- @component BestaetigungEtiketten — der Etikettenblock der Lieferanten-Seite.
     Aus BestellBestaetigung.svelte herausgezogen (21.09.2026): Die Seite steht in der
     Größen-Ratsche und darf nicht wachsen, und der Block bekam eine zweite Form — gilt die
     Bestellung der Schülerbücherei, gibt es das große Lernmittel-Etikett nicht
     (`grosses_etikett`, entschieden in api/mittel_vermerk.go). Der Knopf entfällt dann
     ganz, statt deaktiviert stehen zu bleiben; die Tür liefert das Etikett auch nicht. -->
<script>
	import Button from '../ui/Button.svelte';
	import Select from '../ui/Select.svelte';
	import { Check } from '@lucide/svelte';

	/** @type {{ bestellung: any, token: string, formatId?: string, geoeffneteGroesse?: string }} */
	let { bestellung, token, formatId = $bindable(''), geoeffneteGroesse = $bindable('') } = $props();

	// „Klein" sagt die Seite nur, wenn es ein „Groß" daneben gibt. Bei einer Bestellung für
	// die Schülerbücherei gibt es EINE Größe — dort heißt sie schlicht „Etiketten".
	let rasterBeschriftung = $derived(
		bestellung.grosses_etikett ? 'Bogenraster der kleinen Etiketten' : 'Bogenraster der Etiketten'
	);

	/** @param {'klein' | 'gross'} groesse */
	function etikettenOeffnen(groesse) {
		geoeffneteGroesse = groesse;
		// Das Raster gilt nur für die kleinen Etiketten. Das große Lernmittel-Etikett hat
		// ein festes Raster (4 Stück auf A4) und wird ausgeschnitten, nicht auf
		// vorgestanzte Bögen gedruckt.
		const query = groesse === 'klein' && formatId ? `?format=${encodeURIComponent(formatId)}` : '';
		window.open(
			`/api/public/bestellung/${encodeURIComponent(token)}/etiketten/${groesse}${query}`,
			'_blank',
			'noopener'
		);
	}
</script>

<div class="bg-surface-container-lowest rounded-xl p-8 shadow-sm">
	<h2 class="text-on-surface text-base font-medium">Etiketten drucken</h2>
	<p class="text-on-surface-variant mt-1 text-sm">
		{#if bestellung.grosses_etikett}
			Beide Bögen enthalten dieselben Barcodes wie der Anhang der Bestellmail — Sie wählen nur das
			Format.
		{:else}
			Der Bogen enthält dieselben Barcodes wie der Anhang der Bestellmail.
		{/if}
	</p>

	{#if bestellung.etiketten_formate?.length}
		<div class="mt-5 max-w-md space-y-1.5">
			<label for="etikettenformat" class="text-on-surface-variant block text-xs font-medium">
				{rasterBeschriftung}
			</label>
			<Select
				id="etikettenformat"
				bind:value={formatId}
				options={bestellung.etiketten_formate.map((/** @type {any} */ f) => ({
					value: f.id,
					label: f.name
				}))}
				aria-label={rasterBeschriftung}
			/>
			<p class="text-on-surface-variant text-xs">
				Passend zu den Etikettenbögen in Ihrem Drucker.
				{#if bestellung.grosses_etikett}
					Gilt nicht für die großen Lernmittel-Etiketten — die liegen zu viert auf einem A4-Blatt
					und werden ausgeschnitten.
				{/if}
			</p>
		</div>
	{/if}

	<div class="mt-4 flex flex-wrap gap-3">
		<Button size="lg" variant="secondary" onclick={() => etikettenOeffnen('klein')}>
			{#if geoeffneteGroesse === 'klein'}<Check size={16} aria-hidden="true" />{/if}
			{bestellung.grosses_etikett ? 'Kleine Etiketten (Bogen A4)' : 'Etiketten (Bogen A4)'}
		</Button>
		{#if bestellung.grosses_etikett}
			<Button size="lg" variant="secondary" onclick={() => etikettenOeffnen('gross')}>
				{#if geoeffneteGroesse === 'gross'}<Check size={16} aria-hidden="true" />{/if}
				Große Lernmittel-Etiketten (4 je A4-Blatt)
			</Button>
		{/if}
	</div>
	{#if geoeffneteGroesse}
		<p class="text-on-surface-variant mt-3 text-xs">
			Der Bogen wurde in einem neuen Tab geöffnet. Erscheint er nicht, ist er vom Browser blockiert
			worden — dann bitte den Knopf erneut drücken.
		</p>
	{/if}
</div>
