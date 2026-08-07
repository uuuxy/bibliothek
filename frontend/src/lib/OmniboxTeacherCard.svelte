<script>
	import { GraduationCap, X } from '@lucide/svelte';
	let { teacher, onDeselect } = $props();
</script>

<div
	class="w-full max-w-xl p-5 rounded-2xl bg-blue-50 border border-blue-100 flex items-center justify-between shadow-sm animate-slide-up"
>
	<div class="flex items-center space-x-4">
		<div
			class="w-12 h-12 rounded-xl bg-blue-100/50 border border-blue-200/50 flex items-center justify-center text-blue-600"
		>
			<GraduationCap class="h-6 w-6" aria-hidden="true" />
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
			title="Lehrer abwählen (ESC)"><X class="h-5 w-5" aria-hidden="true" /></button
		>
	</div>
</div>
