<!-- @component LmfPlanZeileAktionen — die Knöpfe einer Zeile im Planer: hoch, runter,
     zusammenlegen oder trennen, davor einfügen, festlegen oder lösen, entfernen. Nur
     Knöpfe und ihre Namen; was sie tun, steht in LmfPlanReihenfolge. -->
<script>
	import { ArrowDown, ArrowUp, Merge, Pin, PinOff, Plus, Split, Trash2 } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';

	/** @type {{ nummer: number, anzahl: number, mehrere: boolean, fest: boolean, onhoch: () => void, onrunter: () => void, onzusammen: () => void, ontrennen: () => void, oneinfuegen: () => void, onfest: () => void, onentfernen: () => void }} */
	let {
		nummer,
		anzahl,
		mehrere,
		fest,
		onhoch,
		onrunter,
		onzusammen,
		ontrennen,
		oneinfuegen,
		onfest,
		onentfernen
	} = $props();
</script>

<Button
	variant="ghost"
	size="sm"
	onclick={onhoch}
	disabled={nummer === 1}
	title="Nach oben"
	aria-label="Zeile {nummer} nach oben"
>
	<ArrowUp class="h-4 w-4" aria-hidden="true" />
</Button>
<Button
	variant="ghost"
	size="sm"
	onclick={onrunter}
	disabled={nummer === anzahl}
	title="Nach unten"
	aria-label="Zeile {nummer} nach unten"
>
	<ArrowDown class="h-4 w-4" aria-hidden="true" />
</Button>
{#if mehrere}
	<Button
		variant="ghost"
		size="sm"
		onclick={ontrennen}
		title="In einzelne Stunden trennen"
		aria-label="Zeile {nummer} trennen"
	>
		<Split class="h-4 w-4" aria-hidden="true" />
	</Button>
{:else}
	<Button
		variant="ghost"
		size="sm"
		onclick={onzusammen}
		disabled={nummer === 1}
		title="Mit der Zeile davor in eine Stunde legen"
		aria-label="Zeile {nummer} mit voriger zusammenlegen"
	>
		<Merge class="h-4 w-4" aria-hidden="true" />
	</Button>
{/if}
<Button
	variant="ghost"
	size="sm"
	onclick={oneinfuegen}
	title="Zeile ohne Klasse davor einfügen"
	aria-label="Vor Zeile {nummer} einfügen"
>
	<Plus class="h-4 w-4" aria-hidden="true" />
</Button>
{#if fest}
	<Button
		variant="ghost"
		size="sm"
		onclick={onfest}
		title="Festen Platz lösen — die Zeile fließt wieder mit"
		aria-label="Zeile {nummer} lösen"
	>
		<PinOff class="h-4 w-4" aria-hidden="true" />
	</Button>
{:else}
	<Button
		variant="ghost"
		size="sm"
		onclick={onfest}
		title="Datum und Stunde festlegen (Ausflug, Projekttag)"
		aria-label="Zeile {nummer} festlegen"
	>
		<Pin class="h-4 w-4" aria-hidden="true" />
	</Button>
{/if}
<Button
	variant="ghost"
	size="sm"
	onclick={onentfernen}
	title="Zeile aus dem Plan nehmen"
	aria-label="Zeile {nummer} entfernen"
>
	<Trash2 class="h-4 w-4" aria-hidden="true" />
</Button>
