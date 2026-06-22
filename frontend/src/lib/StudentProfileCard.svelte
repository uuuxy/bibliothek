<script>
  import { apiFetch, apiClient } from "./apiFetch.js";
  import { studentTabExtensions } from "./plugins.svelte.js";

  /** @type {{ profile: any, role: string, timestamp: number, showWebcam: boolean, showDeleteConfirm: boolean, onDeselect: () => void, onPrint: () => void, leftActions?: import('svelte').Snippet }} */
  let { 
    profile = $bindable(), 
    role = "", 
    timestamp, 
    showWebcam = $bindable(), 
    showDeleteConfirm = $bindable(), 
    onDeselect, 
    onPrint, 
    leftActions 
  } = $props();

  // ── Abgangsjahr inline editing ────────────────────────────────────────────
  let editingAbgang = $state(false);
  let abgangInput = $state(0);
  let abgangSaving = $state(false);
  let abgangError = $state("");
  let imageFailed = $state(false);

  function startEditAbgang() {
    abgangInput = profile.abgaenger_jahr;
    abgangError = "";
    editingAbgang = true;
  }

  /** Calculates the expected graduation year from a class string (mirrors backend logic) */
  function calcAbgangFromKlasse(klasse) {
    const kl = (klasse || "").toLowerCase().trim();
    const m = kl.match(/^(\d+)(.*)/);
    if (!m) return new Date().getFullYear() + 5;
    const grade = parseInt(m[1], 10);
    const suffix = m[2] || "";
    let maxGrade;
    if (suffix.startsWith("h")) maxGrade = 9;
    else if (grade >= 11) maxGrade = 13;
    else maxGrade = 10;
    const yearsLeft = Math.max(0, maxGrade - grade);
    const now = new Date();
    const base = now.getMonth() >= 7 ? now.getFullYear() + 1 : now.getFullYear();
    return base + yearsLeft;
  }

  async function saveAbgang() {
    const year = parseInt(String(abgangInput), 10);
    if (isNaN(year) || year < 2000 || year > 2100) {
      abgangError = "Bitte ein gültiges Jahr eingeben (2000–2100)";
      return;
    }
    abgangSaving = true;
    abgangError = "";
    try {
      const res = await apiClient.patch(`/api/schueler/${profile.id}`, { abgaenger_jahr: year });
      if (res.ok) {
        profile.abgaenger_jahr = year;
        editingAbgang = false;
      } else {
        const d = await res.json().catch(() => ({}));
        abgangError = d.error || "Fehler beim Speichern";
      }
    } catch {
      abgangError = "Netzwerkfehler";
    } finally {
      abgangSaving = false;
    }
  }

  async function handleBlockChange() {
    try {
      const res = await apiClient.patch(`/api/schueler/${profile.id}`, { 
        is_manually_blocked: profile.is_manually_blocked,
        block_reason: profile.block_reason || ""
      });
      if (res.ok) {
        // Lokales Update des abgeleiteten Status "Gesperrt" für sofortiges Feedback
        profile.ist_gesperrt = profile.is_manually_blocked || profile.has_open_damages;
      }
    } catch (e) {
      console.error("Fehler beim Speichern der manuellen Sperre", e);
    }
  }
</script>

<div class="lg:col-span-1 relative bg-white rounded-3xl border border-slate-100 shadow-lg p-8 flex flex-col items-center text-center space-y-6">
  <button onclick={onDeselect} class="absolute top-4 right-4 p-2 text-slate-400 hover:text-slate-700 hover:bg-slate-100 rounded-full transition-colors cursor-pointer" title="Schüler schließen (ESC)">
    <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
  </button>

  <div class="relative group">
    {#if profile.foto_url && !imageFailed}
      <img 
        src="{profile.foto_url.startsWith('data:') ? profile.foto_url : profile.foto_url + '?t=' + timestamp}" 
        alt="Passbild" 
        class="w-40 h-40 object-cover rounded-3xl border border-slate-100 shadow-sm" 
        onerror={() => imageFailed = true}
      />
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
    <h3 class="text-2xl md:text-3xl font-extrabold font-sans text-slate-900 leading-tight">{profile.vorname} {profile.nachname}</h3>
    <p class="text-base md:text-lg text-slate-700 font-bold">Klasse {profile.klasse}</p>
    
    {#if role === 'admin'}
      {#if editingAbgang}
        <div class="flex items-center gap-2 justify-center flex-wrap">
          <input
            type="number"
            min="2000" max="2100"
            bind:value={abgangInput}
            class="w-24 px-2 py-1 text-sm border border-blue-400 rounded-xl text-center font-bold focus:outline-none focus:ring-2 focus:ring-blue-200"
          />
          <button
            onclick={() => { abgangInput = calcAbgangFromKlasse(profile.klasse); }}
            class="px-2 py-1 text-xs bg-slate-100 hover:bg-slate-200 border border-slate-200 rounded-xl font-semibold text-slate-600 cursor-pointer"
            title="Automatisch aus Klasse berechnen">↺ Neu berechnen</button>
          <button
            onclick={saveAbgang}
            disabled={abgangSaving}
            class="px-3 py-1 text-xs bg-blue-600 hover:bg-blue-700 text-white rounded-xl font-bold cursor-pointer disabled:opacity-50">
            {abgangSaving ? '…' : 'Speichern'}
          </button>
          <button onclick={() => editingAbgang = false} class="px-2 py-1 text-xs text-slate-500 hover:text-slate-700 cursor-pointer">✕</button>
        </div>
        {#if abgangError}<p class="text-xs text-rose-500 mt-1">{abgangError}</p>{/if}
      {:else}
        <button
          onclick={startEditAbgang}
          class="text-sm text-slate-500 font-semibold hover:text-blue-600 hover:underline cursor-pointer transition-colors"
          title="Abgangsjahr bearbeiten">
          Abgang {profile.abgaenger_jahr} ✎
        </button>
      {/if}
    {:else}
      <p class="text-sm text-slate-500 font-semibold cursor-default">
        Abgang {profile.abgaenger_jahr}
      </p>
    {/if}
    <p class="text-xs text-slate-400 tracking-wider mt-1">{profile.barcode_id}</p>
  </div>

  <div class="w-full mt-2 bg-slate-50 border border-slate-200 rounded-2xl p-4 flex flex-col gap-3 text-left">
    <div class="flex items-center justify-between">
      <span class="text-sm font-bold text-slate-800">Konto-Status</span>
      {#if profile.ist_gesperrt}
        <span class="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-bold bg-rose-100 text-rose-700">
          <span class="w-1.5 h-1.5 rounded-full bg-rose-500 mr-1.5 animate-pulse"></span>
          Gesperrt
        </span>
      {:else}
        <span class="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-bold bg-emerald-100 text-emerald-700">
          <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 mr-1.5"></span>
          Aktiv
        </span>
      {/if}
    </div>
    
    {#if role === 'admin' || role === 'mitarbeiter'}
    <div class="pt-2 border-t border-slate-200">
      <label class="flex items-center justify-between cursor-pointer group">
        <span class="text-xs font-semibold text-slate-600 group-hover:text-slate-900 transition-colors">Sperre erzwingen</span>
        <div class="relative inline-flex items-center h-5 rounded-full w-9 transition-colors duration-300 focus-within:ring-2 focus-within:ring-offset-1 focus-within:ring-rose-500 {profile.is_manually_blocked ? 'bg-rose-500' : 'bg-slate-300'}">
          <input type="checkbox" class="peer sr-only" bind:checked={profile.is_manually_blocked} onchange={handleBlockChange}>
          <span class="inline-block w-3.5 h-3.5 transform bg-white rounded-full transition-transform duration-300 ease-in-out ml-[2px] {profile.is_manually_blocked ? 'translate-x-4' : 'translate-x-0'}"></span>
        </div>
      </label>
      {#if profile.is_manually_blocked}
        <div class="mt-2 animate-fade-in">
          <textarea bind:value={profile.block_reason} onchange={handleBlockChange} class="w-full border border-rose-200 rounded-lg p-2 text-xs focus:outline-none focus:ring-2 focus:ring-rose-200 bg-white text-rose-900 placeholder-rose-300" rows="2" placeholder="Sperrgrund (z.B. Mahngebühr)..."></textarea>
        </div>
      {/if}
    </div>
    {/if}
  </div>

  <div class="w-full pt-4 flex flex-col gap-3">
    <button onclick={onPrint} class="w-full py-3.5 border border-blue-600 text-blue-600 hover:bg-blue-50 rounded-full text-sm font-bold transition-all cursor-pointer flex items-center justify-center gap-2">
      <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" /></svg>
      Ausweis drucken
    </button>
  </div>

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

  {@render leftActions?.()}
</div>
