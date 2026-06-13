<script>
  import { onMount } from "svelte";
  import { mahnwesenStore } from "./stores/mahnwesen.svelte.js";
  import MahnwesenFilters from "./components/mahnwesen/MahnwesenFilters.svelte";
  import MahnwesenTable from "./components/mahnwesen/MahnwesenTable.svelte";

  $effect(() => {
    mahnwesenStore.fetchData();
  });
</script>

{#if mahnwesenStore.globalErrorToast}
  <div class="fixed top-6 right-6 z-50 px-5 py-3 rounded-2xl shadow-xl text-sm font-semibold animate-fade-in bg-rose-600 text-white flex items-center gap-2">
    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
        <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm-1-9a1 1 0 112 0v4a1 1 0 11-2 0v-4zm1-3a1 1 0 100 2 1 1 0 000-2z" clip-rule="evenodd" />
    </svg>
    {mahnwesenStore.globalErrorToast}
  </div>
{/if}

<div class="max-w-5xl mx-auto space-y-6">
  <!-- Header und Filter -->
  <MahnwesenFilters />

  <!-- Tabellen und Modals -->
  <MahnwesenTable />
</div>
