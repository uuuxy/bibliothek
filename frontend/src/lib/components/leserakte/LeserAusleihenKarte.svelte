<script>
	import LeserAusleihenListe from './LeserAusleihenListe.svelte';
	import SuchZustand from '../ui/SuchZustand.svelte';
	import { anzahlText } from '../../utils/format.js';

	/** @type {{ ausleihen: any[], onReturnClick?: (barcode: string) => void, onDamageClick?: (book: any) => void }} */
	let { ausleihen = [], onReturnClick = undefined, onDamageClick = undefined } = $props();
</script>

<div class="w-full h-full pt-2">
	<div class="flex items-center justify-between pb-3 border-b border-outline-variant mb-6">
		<h3 class="text-title-medium text-on-surface-variant">
			<!-- Die Liste trägt auch verliehene Geräte, darum zählt die Überschrift Ausleihen. -->
			{anzahlText(ausleihen?.length, 'Ausleihe', 'Ausleihen')}
		</h3>
	</div>

	{#if !ausleihen || ausleihen.length === 0}
		<SuchZustand symbol="menu_book" titel="Keine Ausleihen" />
	{:else}
		<LeserAusleihenListe {ausleihen} {onReturnClick} {onDamageClick} />
	{/if}
</div>
