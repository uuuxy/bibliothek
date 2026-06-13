<script>
  import Modal from "./Modal.svelte";
  import StudentProfile from "./StudentProfile.svelte";

  /** @type {{ open: boolean, student: any, role?: string, onclose: () => void }} */
  let { open = false, student = null, role = "", onclose } = $props();
</script>

<Modal {open} {onclose} size="4xl">
  {#snippet header()}
    <div class="flex items-center gap-3">
      <div class="w-8 h-8 bg-blue-100 text-blue-600 rounded-xl flex items-center justify-center">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
        </svg>
      </div>
      <h3 class="text-base font-bold text-slate-800">Schülerprofil</h3>
    </div>
  {/snippet}
  {#snippet children()}
    <div class="p-6 bg-slate-50/30 max-h-[80vh] overflow-y-auto">
      {#if student}
        <StudentProfile {student} onDeselect={onclose} {role} />
      {/if}
    </div>
  {/snippet}
</Modal>
