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
		class="bg-zinc-950/40 p-1.5 rounded-2xl border border-zinc-800/40 inline-flex shadow-inner"
		role="tablist"
		aria-label="Ansichtsmodus"
	>
		<button
			role="tab"
			aria-selected={ansichtsModus === "list"}
			onclick={() => onWechsel("list")}
			class="px-6 py-2 rounded-xl text-xs font-semibold transition-all duration-200 cursor-pointer {ansichtsModus ===
			'list'
				? 'bg-emerald-500 text-zinc-950 font-bold shadow-md'
				: 'text-zinc-400 hover:text-zinc-200'}"
		>
			Listenansicht
		</button>
		<button
			role="tab"
			aria-selected={ansichtsModus === "classes"}
			onclick={() => onWechsel("classes")}
			class="px-6 py-2 rounded-xl text-xs font-semibold transition-all duration-200 cursor-pointer {ansichtsModus ===
			'classes'
				? 'bg-emerald-500 text-zinc-950 font-bold shadow-md'
				: 'text-zinc-400 hover:text-zinc-200'}"
		>
			Klassenansicht
		</button>
	</div>
</div>
