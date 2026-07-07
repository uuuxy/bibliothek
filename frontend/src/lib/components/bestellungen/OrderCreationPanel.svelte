<script>
  import OrderSearch from "./OrderSearch.svelte";
  import OrderCart from "./OrderCart.svelte";
</script>

<div class="lg:col-span-8 space-y-6">
  <div class="border-b border-slate-200 pb-3 flex items-center justify-between">
    <h2 class="text-lg font-bold text-slate-800">Neue Buchbestellung erstellen</h2>
    <span class="text-[10px] bg-blue-50 text-blue-700 px-2 py-0.5 rounded-md font-bold uppercase tracking-wider">Entwurf</span>
  </div>
  <OrderSearch />
  <OrderCart />
</div>
