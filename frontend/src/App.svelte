<script>
  import { onMount } from "svelte";
  import Omnibox from "./lib/Omnibox.svelte";
  import BookDetails from "./lib/BookDetails.svelte";
  import Graduates from "./lib/Graduates.svelte";
  import StudentIdDesigner from "./lib/StudentIdDesigner.svelte";
  import LabelPrinter from "./lib/LabelPrinter.svelte";
  import OrderDashboard from "./lib/OrderDashboard.svelte";
  import UnifiedInventory from "./lib/UnifiedInventory.svelte";
  import StatsDashboard from "./lib/StatsDashboard.svelte";
  import AuditLog from "./lib/AuditLog.svelte";
  import StudentDirectory from "./lib/StudentDirectory.svelte";
  import { appState } from "./inventur/lib/store.svelte.js";
  import { icons, menuGroups } from "./lib/menu.js";

  let isLoggedIn = $state(false);
  let currentUser = $state(/** @type {any} */ (null));
  let heartbeatOk = $state(true);
  let lastHeartbeatTime = $state(Date.now());
  let loginBarcode = $state("");
  let sseSource = $state(/** @type {any} */ (null));
  let loginError = $state(/** @type {string | null} */ (null));

  let activeTab = $state("kiosk"); 
  let selectedBook = $state(/** @type {any} */ (null));
  let isSidebarCollapsed = $state(false);

  // Focus login input initially if not logged in
  $effect(() => {
    if (!isLoggedIn) {
      setTimeout(() => document.getElementById("login-input")?.focus(), 50);
    }
  });

  // Focus the omnibox input if we switch back to kiosk tab and it is idle
  $effect(() => {
    if (isLoggedIn && activeTab === "kiosk") {
      setTimeout(() => document.getElementById("omnibox-input")?.focus(), 50);
    }
  });

  $effect(() => {
    /** @param {KeyboardEvent} e */
    function handleGlobalKeyDown(e) {
      if (e.key === "Escape" && activeTab !== "kiosk") {
        activeTab = "kiosk";
      }
    }
    window.addEventListener("keydown", handleGlobalKeyDown);
    return () => window.removeEventListener("keydown", handleGlobalKeyDown);
  });

  $effect(() => {
    if (!isLoggedIn) return;
    const checker = setInterval(() => {
      if (Date.now() - lastHeartbeatTime > 2000) heartbeatOk = false;
    }, 500);
    return () => clearInterval(checker);
  });

  function connectSSE() {
    if (sseSource) sseSource.close();
    const source = new EventSource("/events");
    sseSource = source;
    lastHeartbeatTime = Date.now();
    heartbeatOk = true;
    source.addEventListener("ping", () => {
      lastHeartbeatTime = Date.now();
      heartbeatOk = true;
    });
    source.onerror = () => { heartbeatOk = false; };
  }

  /** @param {Event} [e] */
  async function handleLogin(e) {
    if (e) e.preventDefault();
    if (!loginBarcode.trim()) return;
    loginError = null;

    try {
      const res = await fetch("/login/barcode", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ barcode_id: loginBarcode })
      });
      if (!res.ok) throw new Error(await res.text() || "Login fehlgeschlagen");
      currentUser = await res.json();
      isLoggedIn = true;
      loginBarcode = "";
      connectSSE();

      if (currentUser && (currentUser.rolle === "admin" || currentUser.rolle === "mitarbeiter")) {
        appState.adminAuthenticated = true;
        appState.guestAuthenticated = true;
      } else if (currentUser && currentUser.rolle === "lehrer") {
        appState.guestAuthenticated = true;
      }
    } catch (err) {
      const errorMessage = /** @type {any} */ (err).message || String(err);
      loginError = errorMessage;
      loginBarcode = "";
      setTimeout(() => { loginError = null; }, 4000);
    }
  }

  function handleLogout() {
    isLoggedIn = false;
    currentUser = null;
    loginBarcode = "";
    loginError = null;
    activeTab = "kiosk";
    appState.adminAuthenticated = false;
    appState.guestAuthenticated = false;
    if (sseSource) {
      sseSource.close();
      sseSource = null;
    }
  }

  /** @param {any} book */
  function handleSelectBook(book) {
    selectedBook = book;
    activeTab = "books";
  }

  const typedIcons = /** @type {any} */ (icons);
</script>

<main class="min-h-screen bg-slate-50 text-slate-800 font-sans selection:bg-slate-200 selection:text-slate-900">
  {#if isLoggedIn && !heartbeatOk}
    <div class="fixed inset-0 bg-white/45 backdrop-blur-lg z-50 flex flex-col items-center justify-center space-y-4">
      <div class="w-12 h-12 border-4 border-t-slate-800 border-slate-200/50 rounded-full animate-spin"></div>
      <h2 class="text-lg font-bold text-slate-800 tracking-wide">VERBINDUNG VERLOREN</h2>
      <p class="text-slate-500 text-xs font-medium">Reconnecting...</p>
    </div>
  {/if}

  {#if !isLoggedIn}
    <div class="min-h-screen flex items-center justify-center p-6 bg-slate-50">
      <form onsubmit={handleLogin} class="w-full max-w-md p-8 rounded-3xl bg-white border border-slate-100 shadow-xl flex flex-col items-center space-y-6 animate-fade-in no-print">
        <div class="w-16 h-16 rounded-2xl bg-slate-50 border border-slate-100 flex items-center justify-center text-slate-600"><svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h4M4 12h4m12 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z" /></svg></div>
        <div class="text-center space-y-1.5">
          <h2 class="text-base font-bold text-slate-800">Scanner-Login erforderlich</h2>
          <p class="text-xs text-slate-400 font-medium">Scanne deine Barcode-Karte, um die Kiosk-Station freizuschalten.</p>
        </div>
        <input id="login-input" type="password" bind:value={loginBarcode} class="w-full bg-slate-50 border border-slate-200 rounded-xl py-3.5 px-4 text-center tracking-widest text-slate-800 focus:outline-none focus:ring-2 focus:ring-slate-500/20 focus:border-slate-300 transition-all font-mono" placeholder="••••••••••••" />
        {#if loginError}
          <p class="text-xs text-rose-500 font-semibold animate-slide-up">{loginError}</p>
        {/if}
      </form>
    </div>
  {:else}
    <div class="min-h-screen flex">
      <aside class="bg-white border-r border-slate-200 flex flex-col justify-between transition-all duration-300 no-print shrink-0 {isSidebarCollapsed ? 'w-16' : 'w-64'}">
        <div class="flex flex-col h-full justify-between">
          <div>
            <div class="h-16 px-4 flex items-center border-b border-slate-100 shrink-0 {isSidebarCollapsed ? 'justify-center' : 'justify-between'}">
              {#if !isSidebarCollapsed}
                <div class="flex items-center gap-3 overflow-hidden">
                  <div class="w-8 h-8 rounded-xl bg-blue-600 flex items-center justify-center text-white shrink-0 shadow-sm animate-fade-in">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" /></svg>
                  </div>
                  <span class="font-bold text-slate-800 tracking-tight animate-fade-in">Bibliothek</span>
                </div>
                <button onclick={() => isSidebarCollapsed = true} class="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 hover:bg-slate-50 transition-colors cursor-pointer" aria-label="Navigation einklappen">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-4.5 w-4.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M11 19l-7-7 7-7m8 14l-7-7 7-7" /></svg>
                </button>
              {:else}
                <button onclick={() => isSidebarCollapsed = false} class="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 hover:bg-slate-50 transition-colors cursor-pointer" aria-label="Navigation ausklappen">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-4.5 w-4.5 rotate-180" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M11 19l-7-7 7-7m8 14l-7-7 7-7" /></svg>
                </button>
              {/if}
            </div>

            <nav class="py-6 px-3 space-y-6">
              {#each menuGroups as group}
                <div class="space-y-1">
                  {#if !isSidebarCollapsed}
                    <span class="px-3 text-[10px] font-bold text-slate-400 uppercase tracking-wider block mb-2 animate-fade-in">{group.name}</span>
                  {/if}
                  {#each group.items as item}
                    {#if !item.adminOnly || (currentUser && currentUser.rolle === 'admin')}
                      <button onclick={() => { activeTab = item.id; selectedBook = null; }} class="w-full flex items-center rounded-xl text-sm font-semibold transition-all {isSidebarCollapsed ? 'justify-center py-2.5 px-0' : 'gap-3 px-3 py-2'} {activeTab === item.id ? 'bg-blue-50 text-blue-700 font-bold' : 'text-slate-600 hover:bg-slate-50 cursor-pointer'}" title={item.label}>
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                          {@html typedIcons[item.icon]}
                        </svg>
                        {#if !isSidebarCollapsed}
                          <span class="animate-fade-in">{item.label}</span>
                        {/if}
                      </button>
                    {/if}
                  {/each}
                </div>
              {/each}
            </nav>
          </div>

          <div>
            {#if !isSidebarCollapsed}
              <div class="p-4 border-t border-slate-100 text-center no-print animate-fade-in shrink-0">
                <div class="inline-flex items-center gap-1.5 py-1 px-3 rounded-full bg-emerald-50 border border-emerald-100/50 text-emerald-700 text-[10px] font-semibold tracking-wide">
                  <span>🛡️ DSGVO anonymisiert</span>
                </div>
              </div>
            {:else}
              <div class="p-4 border-t border-slate-100 text-center no-print flex justify-center shrink-0">
                <span class="text-emerald-600 text-sm cursor-default" title="Scans nach 14 Tagen anonymisiert">🛡️</span>
              </div>
            {/if}
          </div>
        </div>
      </aside>

      <div class="flex-1 flex flex-col min-w-0 bg-slate-50 p-8 w-full">
        <header class="h-16 px-6 bg-white border border-slate-200 rounded-2xl flex items-center justify-between no-print shrink-0 shadow-xs mb-8">
          <div class="flex items-center gap-3">
            <span class="text-xs font-semibold text-slate-400 uppercase tracking-wider font-mono">
              {#if activeTab === "kiosk"}Kiosk / Ausleihe
              {:else if activeTab === "students_dir"}Verwaltung / Schülerdatei
              {:else if activeTab === "books"}Verwaltung / Klassen (LMF)
              {:else if activeTab === "inventory"}Verwaltung / Inventur
              {:else if activeTab === "orders"}Verwaltung / Bestellungen
              {:else if activeTab === "graduates"}Verwaltung / Abgänger
              {:else if activeTab === "stats"}Verwaltung / Statistiken
              {:else if activeTab === "audit"}Verwaltung / Logbuch
              {:else if activeTab === "student_ids"}Druck / Schülerausweise
              {:else if activeTab === "labels"}Druck / Buch-Etiketten
              {/if}
            </span>
          </div>
          
          <div class="flex items-center gap-4">
            <span class="text-xs text-slate-650 font-medium">
              {#if currentUser}
                {currentUser.vorname} {currentUser.nachname} <span class="text-slate-400 font-mono">({currentUser.rolle})</span>
              {/if}
            </span>
            <div class="h-4 w-px bg-slate-200"></div>
            <button onclick={handleLogout} class="text-xs font-bold text-rose-600 hover:text-rose-700 bg-rose-50 hover:bg-rose-100/60 px-3 py-1.5 rounded-lg transition-all cursor-pointer">
              Abmelden
            </button>
          </div>
        </header>

        <main class="flex-1 overflow-y-auto flex flex-col w-full">
          {#if activeTab === "kiosk"}
            <div class="flex-1 flex items-center justify-center w-full max-w-4xl mx-auto animate-fade-in">
              <Omnibox onSelectBook={handleSelectBook} />
            </div>
          {:else if activeTab === "books"}
            <div class="w-full animate-fade-in"><BookDetails title={selectedBook || undefined} /></div>
          {:else if activeTab === "graduates"}
            <div class="w-full animate-fade-in"><Graduates /></div>
          {:else if activeTab === "orders"}
            <div class="w-full animate-fade-in"><OrderDashboard /></div>
          {:else if activeTab === "stats"}
            <div class="w-full animate-fade-in"><StatsDashboard /></div>
          {:else if activeTab === "audit"}
            <div class="w-full animate-fade-in"><AuditLog /></div>
          {:else if activeTab === "student_ids"}
            <div class="w-full animate-fade-in"><StudentIdDesigner /></div>
          {:else if activeTab === "labels"}
            <div class="w-full animate-fade-in"><LabelPrinter /></div>
          {:else if activeTab === "inventory"}
            <div class="w-full animate-fade-in"><UnifiedInventory /></div>
          {:else if activeTab === "students_dir"}
            <div class="w-full animate-fade-in"><StudentDirectory /></div>
          {/if}
        </main>
      </div>
    </div>
  {/if}
</main>

<style>
  @keyframes fadeIn {
    from { opacity: 0; transform: scale(0.98); }
    to { opacity: 1; transform: scale(1); }
  }
  @keyframes slideUp {
    from { opacity: 0; transform: translateY(8px); }
    to { opacity: 1; transform: translateY(0); }
  }
  .animate-fade-in {
    animation: fadeIn 0.4s cubic-bezier(0.16, 1, 0.3, 1) forwards;
  }
  .animate-slide-up {
    animation: slideUp 0.3s cubic-bezier(0.16, 1, 0.3, 1) forwards;
  }

  @media print {
    :global(body) {
      background: white !important;
      color: black !important;
    }
    main {
      background: white !important;
    }
    .no-print {
      display: none !important;
    }
  }
</style>
