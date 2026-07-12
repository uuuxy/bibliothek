<script>
	let { recommendations, onAddToCart } = $props();
</script>

<div class="space-y-4">
	<div class="border-b border-gray-200 pb-3 flex items-center justify-between">
		<h2 class="text-base font-bold text-slate-800">Bestellbedarf</h2>
		<a
			href="/api/bestellungen/pdf"
			download
			class="flex items-center gap-1.5 text-xs font-bold text-slate-500 hover:text-slate-800 transition-colors"
		>
			<svg
				xmlns="http://www.w3.org/2000/svg"
				class="h-3.5 w-3.5 shrink-0"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
				stroke-width="2"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z"
				/>
			</svg>
			PDF-Bestellliste
		</a>
	</div>
	{#if !recommendations.length}
		<p class="text-xs text-slate-400 text-center py-4">Bestände ausreichend.</p>
	{:else}
		<div class="max-h-60 overflow-y-auto space-y-2">
			{#each recommendations as r, _i (_i)}
				<div
					class="p-2.5 bg-slate-50 border border-slate-100 rounded-lg flex items-center justify-between gap-3 text-[11px]"
				>
					<div class="flex items-center gap-2 min-w-0">
						{#if r.cover_url}
							<img
								src="/api/images/cover?isbn={r.isbn || ''}&url={encodeURIComponent(r.cover_url)}"
								class="w-7 aspect-3/4 object-cover rounded-sm shrink-0"
								alt=""
							/>
						{:else}
							<div
								class="w-7 aspect-3/4 rounded bg-slate-200 flex items-center justify-center text-slate-400 shrink-0 text-[9px]"
							>
								📖
							</div>
						{/if}
						<div class="min-w-0">
							<h4 class="font-bold text-slate-800 truncate leading-tight">{r.titel}</h4>
							<p class="text-sm text-slate-600 mt-0.5">
								Bestand: <span class="font-semibold">{r.verfuegbarer_bestand}</span> / Melde:
								<span class="font-semibold">{r.meldebestand}</span>
							</p>
						</div>
					</div>
					<button
						onclick={() => onAddToCart(r)}
						class="shrink-0 px-2 py-1 bg-blue-50 hover:bg-blue-100 text-blue-700 font-bold rounded-md text-[9px] cursor-pointer"
						>+ Add</button
					>
				</div>
			{/each}
		</div>
	{/if}
</div>
