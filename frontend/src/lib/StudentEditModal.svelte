<script>
  import { apiClient } from "./apiFetch.js";
  import Modal from "./Modal.svelte";

  /** @type {{ student: any, onClose: () => void, onSave: () => void }} */
  let { student, onClose, onSave } = $props();

  let loading = $state(false);
  let errorMsg = $state("");

  // Create a reactive deep copy of the student data for editing
  let formData = $state({
    strasse: "",
    hausnummer: "",
    plz: "",
    ort: "",
    eltern_email: ""
  });

  $effect(() => {
    if (student) {
      formData.strasse = student.strasse || "";
      formData.hausnummer = student.hausnummer || "";
      formData.plz = student.plz || "";
      formData.ort = student.ort || "";
      formData.eltern_email = student.eltern_email || "";
    }
  });

  async function save() {
    loading = true;
    errorMsg = "";
    try {
      const payload = {
        strasse: formData.strasse || null,
        hausnummer: formData.hausnummer || null,
        plz: formData.plz || null,
        ort: formData.ort || null,
        eltern_email: formData.eltern_email || null,
      };

      const res = await apiClient.patch(`/api/schueler/${student.id}`, payload);
      if (!res.ok) {
        const data = await res.json();
        throw new Error(data.error || "Speichern fehlgeschlagen");
      }
      onSave(); // notify parent to refresh and show toast
    } catch (e) {
      errorMsg = String(e);
    } finally {
      loading = false;
    }
  }
</script>

<Modal open={true} onclose={onClose} size="md">
  {#snippet header()}
    <h3 class="text-sm font-bold text-slate-800">Schüler bearbeiten</h3>
  {/snippet}
  {#snippet children()}
    <div class="p-6 overflow-y-auto">
      <div class="mb-6">
        <h3 class="text-sm font-bold text-slate-400 uppercase tracking-wider mb-2">Stammdaten (LUSD)</h3>
        <p class="text-slate-800 font-semibold">{student.vorname} {student.nachname}</p>
        <p class="text-slate-500 text-sm">Klasse: {student.klasse}</p>
      </div>

      {#if errorMsg}
        <div class="mb-4 p-3 bg-rose-50 border border-rose-200 text-rose-700 rounded-xl text-sm font-semibold">
          {errorMsg}
        </div>
      {/if}

      <div class="space-y-4">
        <div>
          <h3 class="text-sm font-bold text-slate-400 uppercase tracking-wider mb-3">Postanschrift</h3>
          <div class="grid grid-cols-3 gap-3">
            <div class="col-span-2">
              <label class="block text-xs font-semibold text-slate-600 mb-1" for="strasse">Straße</label>
              <input id="strasse" type="text" bind:value={formData.strasse} class="w-full px-3 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-all text-sm outline-none" />
            </div>
            <div class="col-span-1">
              <label class="block text-xs font-semibold text-slate-600 mb-1" for="hausnr">Hausnr.</label>
              <input id="hausnr" type="text" bind:value={formData.hausnummer} class="w-full px-3 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-all text-sm outline-none" />
            </div>
          </div>
          <div class="grid grid-cols-3 gap-3 mt-3">
            <div class="col-span-1">
              <label class="block text-xs font-semibold text-slate-600 mb-1" for="plz">PLZ</label>
              <input id="plz" type="text" bind:value={formData.plz} class="w-full px-3 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-all text-sm outline-none" />
            </div>
            <div class="col-span-2">
              <label class="block text-xs font-semibold text-slate-600 mb-1" for="ort">Ort</label>
              <input id="ort" type="text" bind:value={formData.ort} class="w-full px-3 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-all text-sm outline-none" />
            </div>
          </div>
        </div>

        <div class="pt-2">
          <h3 class="text-sm font-bold text-slate-400 uppercase tracking-wider mb-3">Kontakt</h3>
          <div>
            <label class="block text-xs font-semibold text-slate-600 mb-1" for="email">Eltern E-Mail</label>
            <input id="email" type="email" bind:value={formData.eltern_email} class="w-full px-3 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-all text-sm outline-none" placeholder="z.B. mail@example.com" />
          </div>
        </div>
      </div>
    </div>

    <div class="flex justify-end gap-3 pt-4 border-t border-slate-100 mt-2">
      <button onclick={onClose} disabled={loading} class="rounded-xl bg-slate-100 px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-200 disabled:opacity-60 transition-colors cursor-pointer font-sans">Abbrechen</button>
      <button onclick={save} disabled={loading} class="rounded-xl bg-blue-600 px-4 py-2 text-sm font-bold text-white hover:bg-blue-750 disabled:opacity-60 transition-colors cursor-pointer font-sans flex items-center gap-2">
        {#if loading}<div class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>{/if}
        Speichern
      </button>
    </div>
  {/snippet}
</Modal>
