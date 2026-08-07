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

<div class="w-full space-y-6 no-print text-slate-800 animate-fade-in">
	<!-- Header Info -->
	<div
		class="flex flex-col sm:flex-row sm:items-center justify-end gap-4 border-b border-slate-100 pb-5"
	>
		<Button
			size="lg"
			onclick={labelStore.triggerPrint}
			disabled={labelStore.finalLabels.filter((lbl) => !lbl.isBlank).length === 0}
			class="px-5 disabled:bg-slate-200 disabled:text-slate-400 disabled:opacity-100"
		>
			<span><Printer class="h-4 w-4" aria-hidden="true" /> A4-Bogen drucken</span>
		</Button>
	</div>

	<div class="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
		<LabelSettings />
		<LabelPreview />
	</div>
</div>
