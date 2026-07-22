<script>
	import OrderSearch from './OrderSearch.svelte';
	import OrderCart from './OrderCart.svelte';
</script>

<div class="bg-white rounded-2xl border border-slate-200/80 shadow-sm">
	<div class="px-5 pt-5 pb-4 border-b border-slate-100 flex items-center justify-between">
		<h2 class="text-lg font-bold text-slate-900 tracking-tight">Deine Bestellung</h2>
		<span
			class="text-[10px] bg-blue-50 text-blue-700 px-2 py-0.5 rounded-md font-bold uppercase tracking-wider"
			>Entwurf</span
		>
	</div>
	<div class="p-5 space-y-5">
		<OrderSearch />
		<OrderCart />
	</div>
</div>
