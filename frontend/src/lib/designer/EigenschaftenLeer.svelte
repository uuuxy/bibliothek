<!-- @component EigenschaftenLeer — das Eigenschaften-Feld, solange kein Element gewählt ist:
     der Hinweis und darunter die Elemente der Seite, jedes mit seinem Schalter „Sichtbar".

     Ein ausgeblendetes Element steht nicht auf der Leinwand und ließe sich sonst nie wieder
     anklicken; für die Grundelemente ist „Sichtbar aus" zugleich der einzige Weg von der Karte.
     M3, Switch: „Use switches to select standalone or more verbose options in a list", „paired
     with an inline label", „The effects of a switch should start immediately". Die Zeilen sind
     48 px hoch, damit die Trefferflächen der Schalter sich nicht überdecken. -->
<script>
	import { idStore } from './idDesignerStore.svelte.js';
	import { elementName } from './elementName.js';
	import Switch from '../components/ui/Switch.svelte';
	import { Lock, MousePointer2 } from '@lucide/svelte';

	/** @type {{ side: 'front'|'back', gesperrt?: boolean }} */
	const { side, gesperrt = false } = $props();

	const eigen = $props.id();
	const elemente = $derived(side === 'front' ? idStore.front.elements : idStore.back.elements);
	const Symbol = $derived(gesperrt ? Lock : MousePointer2);
</script>

<div class="flex flex-col items-center justify-center py-12 text-center gap-3">
	<Symbol class="w-10 h-10 text-outline-variant" aria-hidden="true" />
	<span class="text-sm text-on-surface-variant"
		>{gesperrt
			? 'Das Design ändern kann, wer die Einstellungen verwalten darf. Ansehen und Testdruck gehen auch so.'
			: 'Element auf der Karte anklicken'}</span
	>
</div>
{#if !gesperrt && elemente.length > 0}
	<div class="border-t border-outline-variant pt-4" data-testid="designer-elemente">
		<span class="block text-xs font-medium text-on-surface-variant">
			Sichtbar auf der {side === 'front' ? 'Vorderseite' : 'Rückseite'}
		</span>
		{#each elemente as e (e.id)}
			<div class="flex min-h-12 items-center justify-between gap-3">
				<label for="{eigen}-{e.id}" class="min-w-0 wrap-break-word text-sm text-on-surface"
					>{elementName(e)}</label
				>
				<Switch id="{eigen}-{e.id}" bind:checked={e.show} />
			</div>
		{/each}
	</div>
{/if}
