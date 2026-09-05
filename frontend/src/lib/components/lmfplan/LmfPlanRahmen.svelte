<!-- @component LmfPlanRahmen — was der Planer vorgibt: erster Tag, Startstunde am
     ersten Tag (der Donnerstag im Juni begann in der 3. Stunde), Stunden je Tag (an
     dieser Schule 6). Alles Weitere — Wochentage, Ferien, Folgetage — rechnet der
     Server; hier steht nichts, was er auch weiß. -->
<script>
	import Feld from '../ui/Feld.svelte';
	import Select from '../ui/Select.svelte';
	import { STUNDEN } from '../../lmfplanDienst.js';

	/** @type {{ ersterTag: string, startstunde: number, stundenJeTag: number }} */
	let { ersterTag = $bindable(), startstunde = $bindable(), stundenJeTag = $bindable() } = $props();

	// Die Startstunde kann nicht hinter dem Tagesende liegen — die Auswahl endet dort.
	const startstunden = $derived(STUNDEN.filter((s) => s <= stundenJeTag));
	$effect(() => {
		if (startstunde > stundenJeTag) startstunde = stundenJeTag;
	});
</script>

<div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
	<Feld id="lmf-plan-erster-tag" label="Erster Tag" type="date" bind:value={ersterTag} />
	<div>
		<label class="mb-1 block text-xs font-medium text-on-surface-variant" for="lmf-plan-startstunde"
			>Beginn am ersten Tag</label
		>
		<Select
			id="lmf-plan-startstunde"
			bind:value={startstunde}
			options={startstunden.map((s) => ({ value: s, label: `${s}. Stunde` }))}
		/>
	</div>
	<div>
		<label class="mb-1 block text-xs font-medium text-on-surface-variant" for="lmf-plan-stunden"
			>Stunden je Tag</label
		>
		<Select
			id="lmf-plan-stunden"
			bind:value={stundenJeTag}
			options={STUNDEN.map((s) => ({ value: s, label: `${s} Stunden` }))}
		/>
	</div>
</div>
