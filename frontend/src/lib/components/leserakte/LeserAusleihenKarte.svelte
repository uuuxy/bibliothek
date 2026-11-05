<script>
	import LeserAusleihenListe from './LeserAusleihenListe.svelte';
	import { BookOpen } from '@lucide/svelte';
	import SuchZustand from '../ui/SuchZustand.svelte';

	/** @type {{ ausleihen: any[], onReturnClick?: (barcode: string) => void, onDamageClick?: (book: any) => void }} */
	let { ausleihen = [], onReturnClick = undefined, onDamageClick = undefined } = $props();
</script>

<div class="w-full h-full pt-2">
	<div class="flex items-center justify-between pb-3 border-b border-outline-variant mb-6">
		<h3 class="text-base font-medium text-on-surface-variant">
			Entliehene Bücher ({ausleihen?.length || 0})
		</h3>
	</div>

	{#if !ausleihen || ausleihen.length === 0}
		<SuchZustand symbol={BookOpen} titel="Keine Ausleihen" />
	{:else}
		<LeserAusleihenListe {ausleihen} {onReturnClick} {onDamageClick} />
	{/if}
</div>
