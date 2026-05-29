<script>
	let { onLogin } = $props();
</script>

<div class="flex h-[80vh] w-full items-center justify-center p-8">
	<div class="w-full h-full bg-white rounded-3xl shadow-2xl border border-gray-100 flex flex-col items-center justify-center space-y-12">
		<div class="text-center">
			<h1 class="text-5xl font-bold text-gray-900 tracking-tight">
				Inventur-Verwaltung
			</h1>
			<p class="text-2xl text-gray-500 mt-6">Administratorbereich</p>
		</div>

		<button
			onclick={onLogin}
			class="px-16 py-8 bg-emerald-600 hover:bg-emerald-700 text-white text-3xl font-bold rounded-2xl transition duration-200 shadow-2xl shadow-emerald-200 cursor-pointer"
		>
			Verwaltung öffnen
		</button>
	</div>
</div>

