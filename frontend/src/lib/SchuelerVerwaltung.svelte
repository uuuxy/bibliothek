<script>
  import { apiFetch } from "./apiFetch.js";
  import StudentEditModal from "./StudentEditModal.svelte";

  let students = $state.raw([]);
  let loading = $state(true);
  let errorMsg = $state("");
  let searchQuery = $state("");
  let selectedStudent = $state(null);
  let showToast = $state(false);
  
  // Performance: Begrenzung der gerenderten DOM-Elemente
  let maxVisible = $state(50);

  async function fetchStudents() {
    loading = true;
    errorMsg = "";
    try {
      const res = await apiFetch("/api/schueler");
      if (!res.ok) throw new Error("Fehler beim Laden der Schüler");
      students = await res.json();
    } catch (e) {
      errorMsg = String(e);
    } finally {
      loading = false;
    }
  }

  // Reactive filtering using $derived for extremely fast client-side searches
  let filteredStudents = $derived(
    students.filter(s => {
      if (!searchQuery) return true;
      const q = searchQuery.toLowerCase();
      const fullName = `${s.vorname} ${s.nachname}`.toLowerCase();
      return fullName.includes(q) || s.klasse.toLowerCase().includes(q);
    })
  );

  // Reset maxVisible when search changes
  $effect(() => {
    searchQuery;
    maxVisible = 50;
  });

  let sendingMailId = $state(null);
  let mailStatus = $state({});

  async function sendNotification(schuelerId) {
    sendingMailId = schuelerId;
    mailStatus[schuelerId] = null;
    try {
      const res = await apiFetch(`/api/mail/send-notification/${schuelerId}`, {
        method: "POST",
        body: JSON.stringify({ templateType: "MAHNUNG_ELTERN" })
      });
      const data = await res.json();
      if (res.ok) {
        mailStatus[schuelerId] = { type: 'success', text: "Mail erfolgreich versendet" };
      } else {
        mailStatus[schuelerId] = { type: 'error', text: data.error || data.message || "Fehler" };
      }
    } catch (e) {
      mailStatus[schuelerId] = { type: 'error', text: "Netzwerkfehler" };
    } finally {
      sendingMailId = null;
      setTimeout(() => {
        if (mailStatus[schuelerId]?.type === 'success') {
           mailStatus[schuelerId] = null;
        }
      }, 3000);
    }
  }

  // $effect runs exactly once on mount to fetch initial data
  $effect(() => {
    fetchStudents();
  });

  function handleSave() {
    selectedStudent = null;
    showToast = true;
    setTimeout(() => showToast = false, 3000);
    fetchStudents(); // Refresh data to show the new changes
  }
</script>

<div class="w-full mx-auto space-y-6">
  <div class="flex flex-col md:flex-row md:items-center justify-between border-b border-slate-100 pb-4 gap-4">
    <h1 class="text-2xl font-bold text-slate-800">Schülerverwaltung</h1>
    <div class="w-full md:w-80">
      <input 
        type="text" 
        bind:value={searchQuery} 
        placeholder="Suchen nach Name oder Klasse..." 
        class="w-full px-4 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-all text-sm outline-none bg-slate-50"
      />
    </div>
  </div>

  {#if showToast}
    <div class="fixed bottom-6 right-6 z-50 bg-emerald-600 text-white px-6 py-3 rounded-xl shadow-lg font-bold text-sm flex items-center gap-2">
      <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
      Änderungen erfolgreich gespeichert!
    </div>
  {/if}

  {#if loading && students.length === 0}
    <div class="py-12 flex justify-center">
      <div class="w-8 h-8 border-2 border-t-blue-500 border-blue-500/20 rounded-full animate-spin"></div>
    </div>
  {:else if errorMsg}
    <div class="p-4 bg-rose-50 text-rose-700 rounded-xl border border-rose-200 font-semibold text-sm">
      {errorMsg}
    </div>
  {:else}
    <div class="bg-white border border-slate-200 rounded-2xl shadow-xs overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-left text-sm border-collapse">
          <thead>
            <tr class="bg-slate-50 border-b border-slate-100 text-xs font-bold text-slate-400 uppercase tracking-wider">
              <th class="py-3 px-4">Name</th>
              <th class="py-3 px-4">Klasse</th>
              <th class="py-3 px-4">Postanschrift</th>
              <th class="py-3 px-4">Eltern E-Mail</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100 text-slate-600 font-medium">
            {#each filteredStudents.slice(0, maxVisible) as schueler}
              <tr 
                class="hover:bg-blue-50/50 cursor-pointer transition-colors"
                onclick={() => selectedStudent = schueler}
              >
                <td class="py-3 px-4 text-slate-800 font-bold">{schueler.vorname} {schueler.nachname}</td>
                <td class="py-3 px-4"><span class="px-2 py-1 bg-slate-100 rounded-md text-slate-500 font-semibold text-xs">{schueler.klasse}</span></td>
                <td class="py-3 px-4">
                  {#if schueler.strasse}
                    {schueler.strasse} {schueler.hausnummer}, {schueler.plz} {schueler.ort}
                  {:else}
                    <span class="text-slate-400 italic text-xs">Keine Angabe</span>
                  {/if}
                </td>
                <td class="py-3 px-4">
                  {#if schueler.eltern_email}
                    <!-- Stop propagation so the row click doesn't trigger if they click the email directly -->
                    <!-- svelte-ignore a11y_click_events_have_key_events -->
                    <!-- svelte-ignore a11y_no_static_element_interactions -->
                    <div class="flex items-center gap-3" onclick={(e) => e.stopPropagation()}>
                      <a href="mailto:{schueler.eltern_email}" class="text-blue-600 hover:underline">{schueler.eltern_email}</a>
                      <button 
                        onclick={() => sendNotification(schueler.id)}
                        disabled={sendingMailId === schueler.id}
                        class="p-1.5 rounded-lg bg-slate-100 text-slate-600 hover:bg-blue-100 hover:text-blue-600 disabled:opacity-50 transition-colors"
                        title="Benachrichtigung senden"
                        aria-label="Benachrichtigung senden"
                      >
                        {#if sendingMailId === schueler.id}
                          <div class="w-4 h-4 border-2 border-blue-600/30 border-t-blue-600 rounded-full animate-spin"></div>
                        {:else}
                          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"/></svg>
                        {/if}
                      </button>
                      {#if mailStatus[schueler.id]}
                        <span class="text-xs font-bold {mailStatus[schueler.id].type === 'success' ? 'text-emerald-600' : 'text-rose-600'}">
                          {mailStatus[schueler.id].text}
                        </span>
                      {/if}
                    </div>
                  {:else}
                    <span class="text-slate-400 italic text-xs">Keine Angabe</span>
                  {/if}
                </td>
              </tr>
            {/each}
            {#if filteredStudents.length === 0}
              <tr>
                <td colspan="4" class="py-8 text-center text-slate-400 font-semibold italic">Keine Schüler gefunden.</td>
              </tr>
            {/if}
          </tbody>
        </table>
        
        {#if filteredStudents.length > maxVisible}
          <div class="p-4 flex justify-center bg-slate-50 border-t border-slate-100">
            <button 
              class="px-6 py-2 bg-white border border-slate-300 text-slate-700 font-bold text-sm rounded-xl hover:bg-slate-50 transition-colors shadow-sm"
              onclick={() => maxVisible += 50}
            >
              Weitere Schüler laden ({filteredStudents.length - maxVisible} verbleibend)
            </button>
          </div>
        {/if}
      </div>
    </div>
  {/if}

  {#if selectedStudent}
    <StudentEditModal 
      student={selectedStudent} 
      onClose={() => selectedStudent = null} 
      onSave={handleSave} 
    />
  {/if}
</div>
