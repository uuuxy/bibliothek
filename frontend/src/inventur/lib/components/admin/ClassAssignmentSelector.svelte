<script>
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

<label for="class-input" class="block text-xs uppercase text-gray-500 font-bold mb-1"
	>ZIELKLASSEN</label
>

<div
	class="flex flex-wrap items-center gap-2 border border-surface-variant/20 rounded-2xl p-2 px-4 w-full sm:w-fit min-w-0 sm:min-w-[300px] bg-white hover:border-emerald-300 focus-within:border-emerald-500 focus-within:ring-2 focus-within:ring-emerald-200 transition-all cursor-text shadow-sm mb-4 sm:mb-6"
>
	<!-- Group Icon -->
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width="20"
		height="20"
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		class="text-gray-500 mr-1"
		><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path><circle cx="9" cy="7" r="4"
		></circle><path d="M23 21v-2a4 4 0 0 0-3-3.87"></path><path d="M16 3.13a4 4 0 0 1 0 7.75"
		></path></svg
	>

	{#each selectedClasses as selectedClass (selectedClass)}
		<span
			class="inline-flex items-center gap-1.5 px-4 py-1.5 bg-emerald-100 text-emerald-800 rounded-full text-sm font-semibold shadow-sm animate-in zoom-in-90 duration-200"
		>
			{selectedClass}
			<button
				onclick={() => removeClass(selectedClass)}
				class="hover:opacity-70 rounded-full transition-opacity ml-1"
				aria-label="Klasse {selectedClass} entfernen"
				title="Klasse entfernen"
			>
				<svg
					xmlns="http://www.w3.org/2000/svg"
					width="16"
					height="16"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2.5"
					stroke-linecap="round"
					stroke-linejoin="round"
					><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"
					></line></svg
				>
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
		class="flex-1 bg-transparent border-none outline-none focus:ring-0 px-1 min-w-[120px] text-gray-900 placeholder:text-gray-400 font-medium"
	/>

	<!-- Chevron Down Icon -->
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width="20"
		height="20"
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		class="text-gray-400 ml-auto pointer-events-none"
		><polyline points="6 9 12 15 18 9"></polyline></svg
	>
</div>
