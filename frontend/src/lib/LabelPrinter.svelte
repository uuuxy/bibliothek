<script>
	import { Printer } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { labelStore } from './stores/labels.svelte.js';
	import LabelSettings from './components/labels/LabelSettings.svelte';
	import LabelPreview from './components/labels/LabelPreview.svelte';
	import Button from './components/ui/Button.svelte';

	onMount(() => {
		labelStore.loadClassGroups();
	});
</script>

<div class="w-full space-y-6 no-print text-on-surface animate-fade-in">
	<div class="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
		<LabelSettings />
		<LabelPreview />
	</div>

	<!-- Der Druckknopf steht am Ende: Er wird erst scharf, wenn ein Titel gewählt ist, und
	     Material 3 setzt die bestätigende Aktion ans Ende des Flusses. -->
	<div class="flex justify-end border-t border-outline-variant pt-5">
		<Button
			size="lg"
			onclick={labelStore.triggerPrint}
			disabled={labelStore.finalLabels.filter((lbl) => !lbl.isBlank).length === 0}
			class="px-5"
		>
			<Printer class="h-4 w-4" aria-hidden="true" />
			A4-Bogen drucken
		</Button>
	</div>
</div>
