<script>
	let {
		classNames = $bindable(),
		classInput = $bindable(),
		onKeydown,
		onBlur,
		onRemove,
	} = $props();
</script>

<div class="mb-6">
	<label for="classNameInput" class="block text-sm font-medium text-gray-700 mb-2">
		Klassennamen <span class="text-gray-500 font-normal">(Mehrere möglich: Mit Enter oder Komma bestätigen)</span>
	</label>

	<div class="flex flex-wrap gap-2 p-2 border border-gray-300 rounded-lg focus-within:ring-2 focus-within:ring-emerald-500 focus-within:border-emerald-500 bg-white transition-all shadow-sm min-h-[46px]">
		{#each classNames as name, index (`${name}-${index}`)}
			<span class="flex items-center gap-1.5 px-3 py-1 bg-emerald-100 text-emerald-800 rounded-full text-sm font-medium shadow-sm transition-transform hover:scale-105 duration-200">
				{name}
				<button
					type="button"
					class="text-emerald-600 hover:text-white bg-emerald-200 hover:bg-red-500 rounded-full p-0.5 transition-colors focus:outline-none focus:ring-2 focus:ring-offset-1 focus:ring-emerald-500"
					onclick={() => onRemove(index)}
					aria-label="{name} entfernen"
				>
					<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</span>
		{/each}

		<input
			id="classNameInput"
			type="text"
			bind:value={classInput}
			onkeydown={onKeydown}
			onblur={onBlur}
			class="flex-1 outline-none min-w-[150px] bg-transparent text-gray-800 placeholder-gray-400 py-1 px-1"
			placeholder={classNames.length === 0 ? "z.B. 9G1, 9G2..." : "Weitere hinzufügen..."}
		/>
	</div>
</div>
