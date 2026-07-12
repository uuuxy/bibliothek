<script>
	import StudentIdDesigner from './StudentIdDesigner.svelte';
	import LabelPrinter from './LabelPrinter.svelte';
	let activeTab = $state('labels');
</script>

<div class="w-full flex flex-col h-full bg-slate-50">
	<div class="px-8 pt-6 pb-4 border-b border-slate-200 bg-white shadow-sm shrink-0">
		<div class="max-w-6xl mx-auto flex gap-6">
			<button
				onclick={() => (activeTab = 'labels')}
				class="pb-3 text-sm font-semibold transition-colors border-b-2 {activeTab === 'labels'
					? 'border-blue-600 text-blue-700'
					: 'border-transparent text-slate-500 hover:text-slate-800'}"
			>
				Buch-Etiketten
			</button>
			<button
				onclick={() => (activeTab = 'ids')}
				class="pb-3 text-sm font-semibold transition-colors border-b-2 {activeTab === 'ids'
					? 'border-blue-600 text-blue-700'
					: 'border-transparent text-slate-500 hover:text-slate-800'}"
			>
				Schülerausweise
			</button>
		</div>
	</div>

	<div class="flex-1 overflow-y-auto">
		{#if activeTab === 'labels'}
			<div class="animate-fade-in h-full">
				<LabelPrinter />
			</div>
		{:else}
			<div class="animate-fade-in h-full">
				<StudentIdDesigner />
			</div>
		{/if}
	</div>
</div>
