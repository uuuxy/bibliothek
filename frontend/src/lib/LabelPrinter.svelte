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
	<div class="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
		<LabelSettings />
		<LabelPreview />
	</div>

	<!-- Der Druckknopf stand vorher in einer eigenen KOPFzeile mit eigener Trennlinie —
	     allein, rechtsbuendig und dauerhaft grau, weil er erst scharf wird, wenn unten
	     ein Titel gewaehlt ist. Gemessen lagen zwischen Reiterzeile und erster
	     Ueberschrift 154 px, in denen nichts weiter stand: Das Erste, was die Seite
	     zeigte, war ein toter Knopf, 700 px entfernt von dem, was ihn scharf macht.
	     Jetzt steht er am ENDE — Material 3 setzt die bestaetigende Aktion ans Ende des
	     Flusses, nicht davor. -->
	<div class="flex justify-end border-t border-slate-200 pt-5">
		<Button
			size="lg"
			onclick={labelStore.triggerPrint}
			disabled={labelStore.finalLabels.filter((lbl) => !lbl.isBlank).length === 0}
			class="px-5 disabled:bg-slate-200 disabled:text-slate-400 disabled:opacity-100"
		>
			<Printer class="h-4 w-4" aria-hidden="true" />
			A4-Bogen drucken
		</Button>
	</div>
</div>
