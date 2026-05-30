<script>
  // Props passed dynamically from extension points
  export let student = null;
  export let collapsed = false;
</script>

{#if student}
  <!-- Slot: studentTab (Student profile card) -->
  <div class="p-3 bg-blue-50 border border-blue-100 rounded-xl text-xs space-y-1">
    <p class="font-bold text-blue-800">Vorlage Schüler-Erweiterung</p>
    <p class="text-slate-600">ID: {student.id}</p>
    <p class="text-slate-600">Name: {student.vorname} {student.nachname}</p>
  </div>
{:else}
  <!-- Slot: sidebar (Navigation bar) -->
  <div class="px-3 py-2 text-xs {collapsed ? 'text-center' : 'text-left'} text-slate-500">
    {#if collapsed}
      <span title="Vorlage Plugin aktiv">🧩</span>
    {:else}
      <div class="p-2.5 bg-slate-50 border border-slate-200 rounded-xl">
        <p class="font-bold text-slate-700">Vorlage Sidebar</p>
        <p class="text-[10px] text-slate-400">Plugin-System aktiv</p>
      </div>
    {/if}
  </div>
{/if}
