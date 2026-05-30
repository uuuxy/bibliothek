<script>
  import { onMount } from "svelte";
  import WebcamCapture from "./WebcamCapture.svelte";
  import { studentTabExtensions } from "./plugins.svelte.js";

  /** @type {{ student: any, onDeselect: () => void, role?: string }} */
  let { student, onDeselect, role = "" } = $props();

  /** @type {any} */
  let profile = $state(null);
  let loading = $state(true);
  let showWebcam = $state(false);
  let timestamp = $state(Date.now());

  let showDeleteConfirm = $state(false);
  let deleteError = $state("");
  let isDeleting = $state(false);

  async function fetchProfile() {
    if (!student) return;
    loading = true;
    try {
      const res = await fetch(`/api/schueler/${student.id}`);
      if (res.ok) {
        profile = await res.json();
      }
    } catch (err) {
      console.error("Fehler beim Laden des Schüler-Profils:", err);
    } finally {
      loading = false;
    }
  }

  async function deleteStudent() {
    if (profile.entliehene_buecher && profile.entliehene_buecher.length > 0) {
      deleteError = "Löschen nicht möglich: Schüler hat noch entliehene Bücher";
      return;
    }

    deleteError = "";
    isDeleting = true;
    try {
      const res = await fetch(`/api/schueler/${profile.id}`, {
        method: "DELETE"
      });
      if (res.ok) {
        showDeleteConfirm = false;
        onDeselect(); // Close profile and reload directory list
      } else {
        const errText = await res.text();
        try {
          const errObj = JSON.parse(errText);
          deleteError = errObj.error || "Fehler beim Löschen des Schülers.";
        } catch {
          deleteError = errText || "Fehler beim Löschen des Schülers.";
        }
      }
    } catch (err) {
      deleteError = "Netzwerkfehler beim Löschen des Schülers.";
      console.error(err);
    } finally {
      isDeleting = false;
    }
  }

  // Reload profile when the student prop changes
  $effect(() => {
    if (student) {
      fetchProfile();
    }
  });

  // Public reload method
  export function reloadProfile() {
    fetchProfile();
  }

  function handlePhotoCaptured() {
    timestamp = Date.now();
    showWebcam = false;
    fetchProfile();
  }
</script>

{#if loading}
  <div class="w-full py-12 flex justify-center items-center">
    <div class="w-8 h-8 border-4 border-slate-800 border-t-transparent rounded-full animate-spin"></div>
  </div>
{:else if profile}
  <div class="w-full grid grid-cols-1 md:grid-cols-12 gap-6 items-start text-slate-800 animate-fade-in no-print">
    <!-- Left: Profile Card (4 cols) -->
    <div class="md:col-span-4 bg-white rounded-2xl border border-slate-100 shadow-xl p-8 flex flex-col items-center text-center space-y-6">
      <div class="relative group">
        {#if profile.foto_url}
          <img src="{profile.foto_url}?t={timestamp}" alt="Passbild" class="w-40 h-40 object-cover rounded-3xl border border-slate-100 shadow-sm" />
        {:else}
          <div class="w-40 h-40 rounded-3xl bg-linear-to-br from-slate-50 to-slate-100 border border-slate-100 flex items-center justify-center text-slate-400">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-16 w-16" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" /></svg>
          </div>
        {/if}
        <button onclick={() => showWebcam = true} aria-label="Passbild mit Webcam aufnehmen" class="absolute bottom-1 right-1 p-2.5 rounded-full bg-slate-900/60 hover:bg-slate-900 text-white backdrop-blur-md transition-all shadow-md cursor-pointer border border-white/20" title="Passbild aufnehmen">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z" /><path stroke-linecap="round" stroke-linejoin="round" d="M15 13a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
        </button>
      </div>

      <div class="space-y-2">
        <h3 class="text-2xl md:text-3xl font-extrabold text-slate-900 leading-tight">{profile.vorname} {profile.nachname}</h3>
        <p class="text-base md:text-lg text-slate-700 font-bold">Klasse {profile.klasse} · Abgang {profile.abgaenger_jahr}</p>
        <p class="text-xs font-mono text-slate-400 tracking-wider mt-1">{profile.barcode_id}</p>
      </div>

      <div class="flex flex-col items-center gap-2 pt-2 w-full">
        {#if profile.ist_gesperrt}
          <span class="inline-flex items-center px-4 py-2 rounded-2xl text-sm font-bold bg-rose-50 border border-rose-100 text-rose-600 w-full justify-center">
            <span class="w-2 h-2 rounded-full bg-rose-500 mr-2 animate-pulse"></span>
            Gesperrt
          </span>
        {:else}
          <span class="inline-flex items-center px-4 py-2 rounded-2xl text-sm font-bold bg-emerald-50 border border-emerald-100 text-emerald-600 w-full justify-center">
            <span class="w-2 h-2 rounded-full bg-emerald-500 mr-2"></span>
            Aktiv
          </span>
        {/if}
      </div>

      <button onclick={onDeselect} class="w-full mt-4 py-3 bg-slate-50 hover:bg-slate-100 border border-slate-200 text-slate-700 rounded-2xl text-sm font-bold transition-all cursor-pointer">
        Schüler schließen (ESC)
      </button>

      {#if role === 'admin' || role === 'mitarbeiter'}
        <button onclick={() => showDeleteConfirm = true} class="w-full py-3 bg-rose-50 hover:bg-rose-100/80 border border-rose-200 text-rose-600 rounded-2xl text-sm font-bold transition-all cursor-pointer">
          Schüler löschen
        </button>
      {/if}

      {#if studentTabExtensions.length > 0}
        <div class="w-full pt-4 border-t border-slate-100 flex flex-col gap-3">
          {#each studentTabExtensions as ext}
            {@const Component = ext.component}
            <div class="text-left w-full">
              <span class="block text-[10px] font-bold text-slate-400 uppercase tracking-wider mb-2">{ext.name}</span>
              <Component student={profile} {...ext.props} />
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <!-- Right: Timeline / Loans List (8 cols) -->
    <div class="md:col-span-8 bg-white rounded-2xl border border-slate-100 shadow-xl p-8 space-y-6">
      <div class="flex items-center justify-between pb-3 border-b border-slate-100">
        <h3 class="text-base font-bold text-slate-500 uppercase tracking-wider font-mono">Entliehene Bücher ({profile.entliehene_buecher?.length || 0})</h3>
      </div>

      {#if !profile.entliehene_buecher || profile.entliehene_buecher.length === 0}
        <div class="py-16 flex flex-col items-center justify-center text-slate-500 space-y-3">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" /></svg>
          <span class="text-sm font-semibold text-slate-400">Aktuell keine Bücher entliehen.</span>
        </div>
      {:else}
        <div class="relative border-l border-slate-100 pl-8 ml-4 py-2 space-y-6">
          {#each profile.entliehene_buecher as book}
            {@const isLMF = book.titel.toLowerCase().startsWith("lmf-")}
            <div class="relative group">
              <!-- Timeline Dot -->
              <span class="absolute left-[-39px] top-4 w-3.5 h-3.5 rounded-full border-2 border-white {isLMF ? 'bg-indigo-500 ring-4 ring-indigo-55/50' : 'bg-slate-400 ring-4 ring-slate-100'}"></span>
              
              <div class="p-5 rounded-2xl border border-slate-100 bg-slate-50/50 hover:bg-slate-50 transition-all duration-200 flex flex-col sm:flex-row sm:items-center justify-between gap-6">
                <div class="flex items-center space-x-5">
                  {#if book.cover_url}
                    <img src={book.cover_url} class="w-16 aspect-3/4 object-cover rounded-xl shadow-md border border-slate-100/50 shrink-0" alt="Cover" />
                  {:else}
                    <div class="w-16 aspect-3/4 rounded-xl shadow-md shrink-0 flex items-center justify-center font-bold text-white bg-linear-to-br from-indigo-500 to-purple-650 text-xl border border-indigo-600/10">
                      {book.titel ? book.titel.charAt(0).toUpperCase() : '?'}
                    </div>
                  {/if}
                  <div class="space-y-1.5 text-left">
                    <div class="flex items-center gap-3 flex-wrap">
                      <h4 class="font-extrabold text-xl md:text-2xl text-slate-900 leading-tight">{book.titel}</h4>
                      {#if isLMF}
                        <span class="inline-flex items-center px-3 py-1 rounded-full text-xs font-bold bg-indigo-50 text-indigo-700 border border-indigo-100 uppercase tracking-wide">
                          LMF-Jahresleihe
                        </span>
                      {/if}
                    </div>
                    <p class="text-base md:text-lg text-slate-650 font-semibold">{book.autor}</p>
                    <p class="text-sm text-slate-500 font-semibold">Signatur: <span class="font-mono text-sm font-bold text-slate-700">{book.barcode_id}</span></p>
                    <p class="text-sm text-slate-500 font-semibold">Ausgeliehen am: <span class="font-bold text-slate-700">{new Date(book.ausgeliehen_am).toLocaleDateString("de-DE")}</span></p>
                  </div>
                </div>

                <div class="flex items-center gap-5 text-sm font-semibold">
                  <div class="text-left sm:text-right">
                    <span class="text-xs text-slate-400 block font-bold uppercase font-mono leading-none">Frist</span>
                    <span class="{isLMF ? 'text-indigo-600' : 'text-slate-700'} font-black text-lg md:text-xl">
                      {new Date(book.rueckgabe_frist).toLocaleDateString("de-DE")}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>
{/if}

{#if showWebcam}
  <WebcamCapture studentId={profile.id} onCapture={handlePhotoCaptured} onClose={() => showWebcam = false} />
{/if}

{#if showDeleteConfirm}
  <div class="fixed inset-0 z-50 grid place-items-center bg-slate-900/40 backdrop-blur-xs p-4 animate-fade-in" role="dialog" aria-modal="true">
    <div class="w-full max-w-md rounded-3xl border border-slate-200 bg-white p-6 shadow-2xl text-slate-800 text-left">
      <h3 class="text-lg font-bold text-rose-600 flex items-center gap-2">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-rose-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
        <span>Schüler löschen</span>
      </h3>

      {#if profile.entliehene_buecher && profile.entliehene_buecher.length > 0}
        <div class="mt-4 p-4 bg-rose-50 border border-rose-100 rounded-2xl text-sm font-semibold text-rose-700">
          Löschen nicht möglich: Schüler hat noch entliehene Bücher
        </div>
        <div class="mt-6 flex justify-end">
          <button onclick={() => { showDeleteConfirm = false; deleteError = ""; }} class="rounded-xl bg-slate-100 px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-200 transition-colors cursor-pointer">Schließen</button>
        </div>
      {:else}
        <p class="mt-4 text-sm text-slate-600 leading-relaxed font-sans">
          Sind Sie sicher, dass Sie den Schüler <strong>{profile.vorname} {profile.nachname}</strong> unwiderruflich aus der Datenbank löschen möchten? Alle historischen Ausleihen werden anonymisiert.
        </p>

        {#if deleteError}
          <div class="mt-4 p-3 bg-rose-50 border border-rose-100 rounded-xl text-xs font-semibold text-rose-600">
            {deleteError}
          </div>
        {/if}

        <div class="mt-6 flex justify-end gap-3">
          <button onclick={() => { showDeleteConfirm = false; deleteError = ""; }} disabled={isDeleting} class="rounded-xl bg-slate-100 px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-200 disabled:opacity-60 transition-colors cursor-pointer">Abbrechen</button>
          <button onclick={deleteStudent} disabled={isDeleting} class="rounded-xl bg-red-650 px-4 py-2 text-sm font-bold text-white hover:bg-red-750 disabled:opacity-60 transition-colors cursor-pointer">
            {#if isDeleting}Löschen...{:else}Unwiderruflich löschen{/if}
          </button>
        </div>
      {/if}
    </div>
  </div>
{/if}
