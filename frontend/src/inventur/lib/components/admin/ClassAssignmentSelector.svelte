<script>
	import { ChevronDown, Users, X } from '@lucide/svelte';
	let { selectedClasses = $bindable([]) } = $props();

	const classInput = $state({ value: '' });

	/** @param {string} name */
	function formatClassName(name) {
		let formatted = name.trim().toUpperCase();
		// Adds leading zero if the string starts with a single digit not followed by another digit
		formatted = formatted.replace(/^(\d)(?!\d)/, '0$1');
		return formatted;
	}

	/** @param {string} inputString */
	function addClass(inputString) {
		const parts = inputString.split(',');
		let added = false;

		for (let part of parts) {
			const formatted = formatClassName(part);
			if (formatted && !selectedClasses.includes(formatted)) {
				selectedClasses = [...selectedClasses, formatted];
				added = true;
			}
		}

		if (added || parts.length > 1) {
			classInput.value = '';
		}
	}

	/** @param {string} name */
	function removeClass(name) {
		selectedClasses = selectedClasses.filter((c) => c !== name);
	}

	/** @param {KeyboardEvent} e */
	function handleKeyDown(e) {
		if (e.key === 'Enter' || e.key === ',') {
			e.preventDefault();
			addClass(classInput.value);
		}
	}
</script>

<label for="class-input" class="block text-xs text-slate-500 font-medium mb-1">ZIELKLASSEN</label>

<div
	class="flex flex-wrap items-center gap-2 border border-surface-variant/20 rounded-xl p-2 px-4 w-full sm:w-fit min-w-0 sm:min-w-75 bg-white hover:border-blue-300 focus-within:border-blue-500 focus-within:ring-2 focus-within:ring-blue-500/20 transition-all cursor-text shadow-sm mb-4 sm:mb-6"
>
	<!-- Group Icon -->
	<Users class="text-slate-500 mr-1" aria-hidden="true" />

	{#each selectedClasses as selectedClass (selectedClass)}
		<span
			class="inline-flex items-center gap-1.5 px-4 py-1.5 bg-secondary-container text-on-secondary-container rounded-full text-sm font-semibold shadow-sm animate-in zoom-in-90 duration-200"
		>
			{selectedClass}
			<button
				onclick={() => removeClass(selectedClass)}
				class="hover:opacity-70 rounded-full transition-opacity ml-1"
				aria-label="Klasse {selectedClass} entfernen"
				title="Klasse entfernen"
			>
				<X class="w-4 h-4" aria-hidden="true" />
			</button>
		</span>
	{/each}
	<input
		id="class-input"
		name="class-input-hidden"
		type="text"
		autocomplete="off"
		spellcheck="false"
		data-lpignore="true"
		placeholder={selectedClasses.length === 0 ? 'Klasse wählen...' : ''}
		bind:value={classInput.value}
		onkeydown={handleKeyDown}
		class="h-auto flex-1 bg-transparent border-none outline-none focus:ring-0 px-1 min-w-30 text-slate-900 placeholder:text-slate-400 font-medium"
	/>

	<!-- Chevron Down Icon -->
	<ChevronDown class="text-slate-400 ml-auto pointer-events-none" aria-hidden="true" />
</div>
