<script>
  import { apiFetch } from "./apiFetch.js";
  import { onMount } from "svelte";
  import WebcamCapture from "./WebcamCapture.svelte";
  import KioskChecklistModal from "./KioskChecklistModal.svelte";
  import KioskDamageModal from "./KioskDamageModal.svelte";
  import DamageReportModal from "./DamageReportModal.svelte";
  import { studentTabExtensions } from "./plugins.svelte.js";
  import { idStore } from "./idLayoutStore.svelte.js";
  import BorrowedBooksCard from "./BorrowedBooksCard.svelte";
  import StudentEditModal from "./StudentEditModal.svelte";

  /** @type {{ student: any, onDeselect: () => void, role?: string, onReturnClick?: (barcode: string) => void, leftActions?: import('svelte').Snippet, rightTop?: import('svelte').Snippet }} */
  let { student, onDeselect, role = "", onReturnClick = undefined, leftActions, rightTop } = $props();

  /** @type {any} */
  let profile = $state(null);
  let loading = $state(true);
  let showWebcam = $state(false);
  let timestamp = $state(Date.now());

  let showDeleteConfirm = $state(false);
  let deleteError = $state("");
  let isDeleting = $state(false);

  // Active Tab for Right Side ('ausleihen' | 'stammdaten')
  let activeTab = $state("ausleihen");

  // Edit Modal State
  let showEditModal = $state(false);

  // Damage Report State
  let showDamageModal = $state(false);
  /** @type {any} */
  let damageBook = $state(null);
  let isSubmittingDamage = $state(false);

  function openDamageModal(book) {
    damageBook = book;
    showDamageModal = true;
  }

  async function submitDamageReport(reason, amount) {
    if (!damageBook) return;
    isSubmittingDamage = true;
    try {
      const res = await apiFetch(`/api/damage/report`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          loan_id: /** @type {any} */ (damageBook).ausleihe_id,
          schueler_id: student.id,
          copy_id: /** @type {any} */ (damageBook).exemplar_id,
          beschreibung: reason,
          betrag: amount
        })
      });
      if (res.ok) {
        const json = await res.json();
        const damageSuccessId = json.schadens_id;
        window.open(`/api/schadensfaelle/${damageSuccessId}/pdf`, '_blank');
        showDamageModal = false;
        fetchProfile(); // reload data
      } else {
        const err = await res.json().catch(() => ({}));
        alert(err.error || "Fehler beim Melden.");
      }
    } catch (e) {
      alert("Netzwerkfehler.");
    } finally {
      isSubmittingDamage = false;
    }
  }

  async function fetchProfile() {
    if (!student) return;
    loading = true;
    try {
      const res = await apiFetch(`/api/schueler/${student.id}`);
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
      const res = await apiFetch(`/api/schueler/${profile.id}`, {
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

  // ── Abgangsjahr inline editing ────────────────────────────────────────────
  let editingAbgang = $state(false);
  let abgangInput = $state(0);
  let abgangSaving = $state(false);
  let abgangError = $state("");

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
      const res = await apiFetch(`/api/schueler/${profile.id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ abgaenger_jahr: year })
      });
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
  // ──────────────────────────────────────────────────────────────────────────

  export function reloadProfile() {
    fetchProfile();
  }

  function handleSaveEdit() {
    showEditModal = false;
    fetchProfile(); // reload data
  }
  
  function formatDate(dateString) {
    if (!dateString) return "Keine Angabe";
    try {
      const d = new Date(dateString);
      return d.toLocaleDateString("de-DE", { day: '2-digit', month: '2-digit', year: 'numeric' });
    } catch {
      return dateString;
    }
  }

  function handlePhotoCaptured() {
    timestamp = Date.now();
    showWebcam = false;
    fetchProfile();
  }

  // ── Single-card print ─────────────────────────────────────────────────────
  // Always uses Scheckkarte ID-1 format (85.60 mm × 53.98 mm) without prompting.
  function printCard() {
    const styleEl = document.createElement("style");
    // ID-1 / ISO 7810 borderless card page — never substitute A4 here
    styleEl.textContent = "@media print { @page { size: 85.6mm 53.98mm; margin: 0; } }";
    document.head.appendChild(styleEl);
    document.body.setAttribute("data-print-mode", "card-single");
    window.print();
    document.head.removeChild(styleEl);
    document.body.removeAttribute("data-print-mode");
  }
  // ──────────────────────────────────────────────────────────────────────────
</script>

{#if loading}
  <div class="w-full py-12 flex justify-center items-center">
    <div class="w-8 h-8 border-4 border-slate-800 border-t-transparent rounded-full animate-spin"></div>
  </div>
{:else if profile}
  <div class="w-full grid grid-cols-1 lg:grid-cols-3 gap-6 items-start text-slate-800 animate-fade-in no-print font-sans">
    <!-- Left: Profile Card (1 col) -->
    <div class="lg:col-span-1 bg-white rounded-2xl border border-slate-100 shadow-xl p-8 flex flex-col items-center text-center space-y-6">
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
        <h3 class="text-2xl md:text-3xl font-extrabold font-sans text-slate-900 leading-tight">{profile.vorname} {profile.nachname}</h3>
        <p class="text-base md:text-lg text-slate-700 font-bold">Klasse {profile.klasse}</p>
        <!-- Abgangsjahr: inline editable (Nur Admin) -->
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

      <!-- Action Buttons -->
      <div class="w-full pt-2 flex flex-col gap-2">
        <button onclick={printCard} class="w-full py-3 bg-indigo-600 hover:bg-indigo-700 text-white rounded-2xl text-sm font-bold transition-all cursor-pointer flex items-center justify-center gap-2 shadow-sm">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" /></svg>
          Ausweis drucken
        </button>

        <button onclick={onDeselect} class="w-full py-3 bg-slate-50 hover:bg-slate-100 border border-slate-200 text-slate-700 rounded-2xl text-sm font-bold transition-all cursor-pointer">
          Schüler schließen (ESC)
        </button>

        {#if role === 'admin'}
          <button onclick={() => showDeleteConfirm = true} class="w-full py-3 bg-rose-50 hover:bg-rose-100/80 border border-rose-200 text-rose-600 rounded-2xl text-sm font-bold transition-all cursor-pointer">
            Schüler löschen
          </button>
        {/if}
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

    <!-- Right: Timeline / Loans List / Stammdaten (2 cols) -->
    <div class="lg:col-span-2 space-y-6 flex flex-col h-full">
      <!-- Tabs -->
      <div class="flex gap-4 border-b border-slate-200 px-2 pt-2">
        <button 
          onclick={() => activeTab = "ausleihen"}
          class="pb-3 px-2 text-sm font-bold transition-all border-b-2 {activeTab === 'ausleihen' ? 'border-blue-600 text-blue-600' : 'border-transparent text-slate-500 hover:text-slate-800'}"
        >
          Ausleihen & Historie
        </button>
        <button 
          onclick={() => activeTab = "stammdaten"}
          class="pb-3 px-2 text-sm font-bold transition-all border-b-2 {activeTab === 'stammdaten' ? 'border-blue-600 text-blue-600' : 'border-transparent text-slate-500 hover:text-slate-800'}"
        >
          Stammdaten & Adresse
        </button>
      </div>

      <div class="flex-1 relative">
        {#if activeTab === "ausleihen"}
          {@render rightTop?.()}
          <div class="col-span-1 md:col-span-1 relative flex flex-col h-full min-h-[400px] animate-fade-in">
            <BorrowedBooksCard 
              books={profile.entliehene_buecher || []} 
              {onReturnClick} 
              onDamageClick={role === 'admin' || role === 'mitarbeiter' ? openDamageModal : undefined}
            />
          </div>
        {:else if activeTab === "stammdaten"}
          <div class="bg-white rounded-2xl border border-slate-100 shadow-sm p-8 animate-fade-in space-y-8">
            <div class="flex justify-between items-center border-b border-slate-100 pb-4">
              <h3 class="text-xl font-bold text-slate-800 flex items-center gap-2">
                <svg class="w-6 h-6 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V8a2 2 0 00-2-2h-5m-4 0V5a2 2 0 114 0v1m-4 0a2 2 0 104 0m-5 8a2 2 0 100-4 2 2 0 000 4zm0 0c1.306 0 2.417.835 2.83 2M9 14a3.001 3.001 0 00-2.83 2M15 11h3m-3 4h2"/></svg>
                Stammdaten & Adresse
              </h3>
              {#if role === 'admin'}
                <button onclick={() => showEditModal = true} class="px-4 py-2 bg-blue-50 text-blue-600 hover:bg-blue-100 rounded-xl text-sm font-bold transition-colors cursor-pointer flex items-center gap-2">
                  <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/></svg>
                  Bearbeiten
                </button>
              {/if}
            </div>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-8">
              <!-- Left Column: Personal Data -->
              <div class="space-y-6">
                <div>
                  <p class="text-xs font-bold text-slate-400 uppercase tracking-wider mb-1">Geburtsdatum</p>
                  <p class="text-slate-800 font-semibold">{formatDate(profile.geburtsdatum)}</p>
                </div>
                
                <div>
                  <p class="text-xs font-bold text-slate-400 uppercase tracking-wider mb-1">LUSD ID</p>
                  <p class="text-slate-800 font-semibold">{profile.lusd_id || 'Keine Angabe'}</p>
                </div>

                <div>
                  <p class="text-xs font-bold text-slate-400 uppercase tracking-wider mb-1">System-ID</p>
                  <p class="text-slate-500 font-mono text-xs">{profile.id}</p>
                </div>
              </div>

              <!-- Right Column: Address & Contact -->
              <div class="space-y-6">
                <div>
                  <p class="text-xs font-bold text-slate-400 uppercase tracking-wider mb-1">Postanschrift</p>
                  {#if profile.strasse}
                    <p class="text-slate-800 font-semibold">{profile.strasse} {profile.hausnummer}</p>
                    <p class="text-slate-800 font-semibold">{profile.plz} {profile.ort}</p>
                  {:else}
                    <p class="text-slate-400 italic text-sm">Keine Adresse hinterlegt</p>
                  {/if}
                </div>

                <div>
                  <p class="text-xs font-bold text-slate-400 uppercase tracking-wider mb-1">Eltern E-Mail</p>
                  {#if profile.eltern_email}
                    <a href="mailto:{profile.eltern_email}" class="text-blue-600 hover:underline font-semibold">{profile.eltern_email}</a>
                  {:else}
                    <p class="text-slate-400 italic text-sm">Keine E-Mail hinterlegt</p>
                  {/if}
                </div>
              </div>
            </div>
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}

{#if showWebcam}
  <WebcamCapture studentId={profile.id} onCapture={handlePhotoCaptured} onClose={() => showWebcam = false} />
{/if}

<!--
  Single-card print section.
  Hidden on screen (display:none inline), shown via @media print when
  body[data-print-mode="card-single"] is set by printCard().
  Rendered outside the .no-print wrapper so it survives print suppression.
  Uses idStore so it always reflects the latest Ausweis-Designer settings.
-->
{#if profile}
<div class="single-card-print-section" style="display:none" aria-hidden="true">
  <div class="print-card-box {idStore.cardTheme}">
    {#if idStore.layout?.header?.show}
      <div class="absolute font-black text-center tracking-tight leading-none truncate text-black"
        style="left: {idStore.layout.header.x}mm; top: {idStore.layout.header.y}mm; transform: scale({idStore.layout.header.scale}); transform-origin: top left; font-size: 7.5pt; width: {85.6 - idStore.layout.header.x * 2}mm;">
        {idStore.layout.header.text}
      </div>
    {/if}
    {#if idStore.layout?.logo?.show && idStore.layout.logo.url}
      <div class="absolute overflow-hidden flex items-center justify-center"
        style="left: {idStore.layout.logo.x}mm; top: {idStore.layout.logo.y}mm; width: {12 * idStore.layout.logo.scale}mm; height: {12 * idStore.layout.logo.scale}mm;">
        <img src={idStore.layout.logo.url} class="w-full h-full object-contain" alt="Logo" />
      </div>
    {/if}
    {#if idStore.layout?.address?.show}
      <div class="absolute font-semibold tracking-tight opacity-75 leading-none text-zinc-800"
        style="left: {idStore.layout.address.x}mm; top: {idStore.layout.address.y}mm; transform: scale({idStore.layout.address.scale}); transform-origin: top left; font-size: 6.5pt; width: {85.6 - idStore.layout.address.x - 2}mm; max-height: 12mm; overflow: hidden;">
        {idStore.layout.address.text}
      </div>
    {/if}
    {#if idStore.layout?.photo?.show}
      <div class="absolute border border-solid border-zinc-300 bg-zinc-50 flex items-center justify-center overflow-hidden rounded-sm"
        style="left: {idStore.layout.photo.x}mm; top: {idStore.layout.photo.y}mm; width: {22 * idStore.layout.photo.scale}mm; height: {28 * idStore.layout.photo.scale}mm;">
        <img src="/uploads/fotos/{profile.barcode_id}.jpg?t={timestamp}"
          onerror={(e) => { (/** @type {any} */ (e.currentTarget)).style.display = 'none'; }}
          class="w-full h-full object-cover" alt="Passbild" />
      </div>
    {/if}
    {#if idStore.layout?.name?.show}
      <div class="absolute font-extrabold tracking-tight leading-none text-black"
        style="left: {idStore.layout.name.x}mm; top: {idStore.layout.name.y}mm; transform: scale({idStore.layout.name.scale}); transform-origin: top left; font-size: 9pt;">
        {profile.vorname} {profile.nachname}
      </div>
    {/if}
    {#if idStore.layout?.details?.show}
      <div class="absolute font-semibold tracking-tight opacity-75 leading-none text-zinc-800"
        style="left: {idStore.layout.details.x}mm; top: {idStore.layout.details.y}mm; transform: scale({idStore.layout.details.scale}); transform-origin: top left; font-size: 7.5pt;">
        Klasse: {profile.klasse}
      </div>
    {/if}
    {#if idStore.layout?.validity?.show}
      <div class="absolute font-semibold tracking-tight opacity-75 leading-none text-zinc-800"
        style="left: {idStore.layout.validity.x}mm; top: {idStore.layout.validity.y}mm; transform: scale({idStore.layout.validity.scale}); transform-origin: top left; font-size: 7pt;">
        Gültig bis: 31.07.{profile.abgaenger_jahr ?? '–'}
      </div>
    {/if}
    {#if idStore.layout?.barcode?.show}
      <div class="absolute flex flex-col items-center leading-none"
        style="left: {idStore.layout.barcode.x}mm; top: {idStore.layout.barcode.y}mm; transform: scale({idStore.layout.barcode.scale}); transform-origin: top left;">
        <img src="/api/barcode?content={profile.barcode_id}&qr={idStore.barcodeType === 'qr'}&width={idStore.barcodeType === 'qr' ? 80 : 200}&height={idStore.barcodeType === 'qr' ? 80 : 50}"
          class="{idStore.barcodeType === 'qr' ? 'h-[11mm] w-[11mm]' : 'h-[8mm]'} object-contain" alt="Barcode" />
        <span class="font-bold mt-1 text-[6.5pt] tracking-widest text-zinc-800">{profile.barcode_id}</span>
      </div>
    {/if}
  </div>
</div>
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

{#if showDamageModal && damageBook}
  <DamageReportModal
    book={damageBook}
    isSubmitting={isSubmittingDamage}
    onCancel={() => showDamageModal = false}
    onSubmit={submitDamageReport}
  />
{/if}

{#if showEditModal}
  <StudentEditModal 
    student={profile} 
    onClose={() => showEditModal = false} 
    onSave={handleSaveEdit} 
  />
{/if}
