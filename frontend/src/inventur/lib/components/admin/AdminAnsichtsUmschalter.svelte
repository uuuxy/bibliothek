<script>
	/**
	 * @type {{
	 *   ansichtsModus: string,
	 *   onWechsel: (modus: string) => void
	 * }}
	 */
	let { ansichtsModus, onWechsel } = $props();
</script>

<div class="flex justify-center mb-6">
	<div
		class="bg-slate-100 p-1 rounded-xl border border-slate-200 inline-flex shadow-inner"
		role="tablist"
		aria-label="Ansichtsmodus"
	>
		<button
			role="tab"
			aria-selected={ansichtsModus === 'list'}
			onclick={() => onWechsel('list')}
			class="px-6 py-2 rounded-lg text-xs font-semibold transition-all duration-200 cursor-pointer {ansichtsModus ===
			'list'
				? 'bg-white text-blue-600 font-bold shadow-sm'
				: 'text-slate-600 hover:text-slate-900'}"
		>
			Listenansicht
		</button>
		<button
			role="tab"
			aria-selected={ansichtsModus === 'classes'}
			onclick={() => onWechsel('classes')}
			class="px-6 py-2 rounded-lg text-xs font-semibold transition-all duration-200 cursor-pointer {ansichtsModus ===
			'classes'
				? 'bg-white text-blue-600 font-bold shadow-sm'
				: 'text-slate-600 hover:text-slate-900'}"
		>
			Klassenansicht
		</button>
	</div>
</div>
