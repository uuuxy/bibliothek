<script>
	let { teacher, onDeselect } = $props();
</script>

<div
	class="w-full max-w-xl p-5 rounded-2xl bg-blue-50 border border-blue-100 flex items-center justify-between shadow-sm animate-slide-up"
>
	<div class="flex items-center space-x-4">
		<div
			class="w-12 h-12 rounded-xl bg-blue-100/50 border border-blue-200/50 flex items-center justify-center text-blue-600"
		>
			<svg
				xmlns="http://www.w3.org/2000/svg"
				class="h-6 w-6"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
				><path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M12 14l9-5-9-5-9 5 9 5z"
				/><path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M12 14l6.16-3.422a12.083 12.083 0 01.665 6.479A11.952 11.952 0 0012 20.055a11.952 11.952 0 00-6.824-2.998 12.078 12.078 0 01.665-6.479L12 14z"
				/></svg
			>
		</div>
		<div>
			<h3 class="font-bold text-blue-800">{teacher.vorname} {teacher.nachname}</h3>
			<p class="text-xs text-blue-600/80 font-medium">
				Handapparat-Modus aktiv · <span class="underline font-semibold"
					>Ausleihe erfolgt als dauerhafter Handapparat</span
				>
			</p>
		</div>
	</div>
	<div class="flex items-center space-x-3">
		<span
			class="text-xs px-2.5 py-1 rounded-full bg-blue-100/80 border border-blue-200 text-blue-700 font-semibold tracking-wide uppercase"
			>Handapparat</span
		>
		<button
			onclick={onDeselect}
			class="p-1 text-blue-500 hover:text-blue-700 transition-colors cursor-pointer"
			title="Lehrer abwählen (ESC)"
			><svg
				xmlns="http://www.w3.org/2000/svg"
				class="h-5 w-5"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
				><path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M6 18L18 6M6 6l12 12"
				/></svg
			></button
		>
	</div>
</div>
