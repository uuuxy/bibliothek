<script>
  import OpacSearch from "./lib/OpacSearch.svelte";
  import Monitor from "./lib/Monitor.svelte";
  const _currentPath = window.location.pathname;

  import Omnibox from "./lib/Omnibox.svelte";
  import BookDetails from "./lib/BookDetails.svelte";
  import Graduates from "./lib/Graduates.svelte";
  import StudentIdDesigner from "./lib/StudentIdDesigner.svelte";
  import LabelPrinter from "./lib/LabelPrinter.svelte";
  import BestellWorkspace from "./lib/BestellWorkspace.svelte";
  import UnifiedInventory from "./lib/UnifiedInventory.svelte";
  import MediaCatalog from "./lib/MediaCatalog.svelte";
  import StatsDashboard from "./lib/StatsDashboard.svelte";
  import AuditLog from "./lib/AuditLog.svelte";
  import StudentDirectory from "./lib/StudentDirectory.svelte";
  import PermissionManager from "./lib/PermissionManager.svelte";
  import LehrerPortal from "./lib/LehrerPortal.svelte";
  import Mahnwesen from "./lib/Mahnwesen.svelte";
  import SystemSettings from "./lib/SystemSettings.svelte";
  import KioskMode from "./lib/KioskMode.svelte";
  import { appState } from "./inventur/lib/store.svelte.js";
  import { menuGroups, canSeeItem } from "./lib/menu.js";
  import { sidebarExtensions } from "./lib/plugins.svelte.js";

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
  let pendingReservierungen = $state(0);
  let isInitialRouteMatched = false;

  /** Polls for open class-set reservations (badge for admin/mitarbeiter). */
  async function fetchPendingReservierungen() {
    try {
      const res = await fetch("/api/reservierungen/klassensatz/anzahl");
      if (res.ok) {
        const data = await res.json();
        pendingReservierungen = data.anzahl ?? 0;
      }
    } catch { /* ignore */ }
  }

  $effect(() => {
    if (!isLoggedIn || !currentUser) {
      pendingReservierungen = 0;
      return;
    }
    if (currentUser.rolle !== "admin" && currentUser.rolle !== "mitarbeiter") return;
    fetchPendingReservierungen();
    const id = setInterval(fetchPendingReservierungen, 30_000);
    return () => clearInterval(id);
  });

  $effect(() => {
    if (appState.pendingPrintCopies) {
      activeTab = "labels";
    }
  });

  // FRONTEND ROUTE-GUARD & INTERACTIVE TAB-URL SYNC
  // 1. Sperrt die Svelte-Routen für unbefugte Benutzer ab: Helfer dürfen nur den Kiosk-Modus und den Medienkatalog aufrufen.
  // 2. Leitet Helfer sofort zum Kiosk-Modus um, falls eine unbefugte URL wie /einstellungen oder /inventur manuell eingetippt wird.
  // 3. Mappt die URL beim Login/Seitenaufruf für berechtigte Benutzer auf den jeweiligen Tab und synchronisiert den Tab-Zustand.
  $effect(() => {
    if (isLoggedIn && currentUser) {
      const role = currentUser.rolle ? currentUser.rolle.toLowerCase() : "";
      const path = window.location.pathname;
      const isHelfer = role === "helfer";

      if (isHelfer) {
        // Helfer darf ausschließlich kiosk und media_catalog sehen
        if (activeTab !== "kiosk" && activeTab !== "media_catalog") {
          activeTab = "kiosk";
        }
        if (path !== "/" && path !== "/kiosk" && path !== "/katalog") {
          window.history.replaceState(null, "", "/kiosk");
        }
      } else {
        // Tab-URL-Mapping Tabelle für berechtigte Rollen (Admin / Mitarbeiter)
        /** @type {Record<string, string>} */
        const tabToPath = {
          settings: "/einstellungen",
          inventory: "/inventur",
          students_dir: "/schuelerdatei",
          orders: "/bestellungen",
          media_catalog: "/katalog",
          graduates: "/abgaenger",
          stats: "/statistiken",
          mahnwesen: "/mahnwesen",
          audit: "/logbuch",
          permissions: "/berechtigungen",
          student_ids: "/ausweise",
          labels: "/etiketten",
          kiosk: "/kiosk"
        };

        // Initiales Mapping des Pfads beim ersten Login/Seitenaufruf
        if (!isInitialRouteMatched && path !== "/") {
          const matchedTab = Object.keys(tabToPath).find(key => tabToPath[key] === path);
          if (matchedTab) activeTab = matchedTab;
          isInitialRouteMatched = true;
        } else if (!isInitialRouteMatched) {
          isInitialRouteMatched = true;
        }

        // Synchronisiert die URL-Anzeige im Browser bei Klick-Navigationen
        const targetPath = tabToPath[activeTab];
        if (targetPath && path !== targetPath && isInitialRouteMatched) {
          window.history.pushState(null, "", targetPath);
        }
      }
    }
  });

  // Focus login input initially if not logged in
  $effect(() => {
    if (!isLoggedIn) {
      setTimeout(() => document.getElementById("login-input")?.focus(), 50);
    }
  });

  // Focus the kiosk student input if we switch back to kiosk tab and it is idle
  $effect(() => {
    if (isLoggedIn && activeTab === "kiosk") {
      setTimeout(() => document.getElementById("kiosk-student-input")?.focus(), 50);
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
      if (!res.ok) {
        let msg = "Login fehlgeschlagen";
        try { const d = await res.json(); msg = d.error || msg; } catch { try { msg = (await res.text()) || msg; } catch {} }
        throw new Error(msg);
      }
      currentUser = await res.json();
      isLoggedIn = true;
      loginBarcode = "";
      connectSSE();

      if (currentUser && (currentUser.rolle === "admin" || currentUser.rolle === "mitarbeiter")) {
        appState.adminAuthenticated = true;
        appState.guestAuthenticated = true;
      } else if (currentUser && currentUser.rolle === "lehrer") {
        appState.guestAuthenticated = true;
        activeTab = "lehrer_portal";
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
    appState.selectedBook = book;
    activeTab = "media_catalog";
  }


</script>

<main class="min-h-screen bg-slate-50 text-slate-800 font-sans selection:bg-slate-200 selection:text-slate-900">
{#if _currentPath === '/katalog'}
  <OpacSearch />
{:else if _currentPath === '/monitor'}
  <Monitor />
{:else}
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
        <input id="login-input" type="password" bind:value={loginBarcode} class="w-full bg-slate-50 border border-slate-200 rounded-xl py-3.5 px-4 text-center tracking-widest text-slate-800 focus:outline-none focus:ring-2 focus:ring-slate-500/20 focus:border-slate-300 transition-all" placeholder="••••••••••••" />
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
                {#if group.items.some(item => canSeeItem(item, currentUser?.rolle))}
                  <div class="space-y-1">
                    {#if !isSidebarCollapsed}
                      <span class="px-3 text-[10px] font-bold text-slate-400 uppercase tracking-wider block mb-2 animate-fade-in">{group.name}</span>
                    {/if}
                    {#each group.items as item}
                      {#if canSeeItem(item, currentUser?.rolle)}
                        <button onclick={() => { activeTab = item.id; selectedBook = null; }} class="relative w-full flex items-center rounded-xl text-sm font-semibold transition-all {isSidebarCollapsed ? 'justify-center py-2.5 px-0' : 'gap-3 px-3 py-2'} {activeTab === item.id ? 'bg-blue-50 text-blue-700 font-bold' : 'text-slate-600 hover:bg-slate-50 cursor-pointer'}" title={item.label}>
                          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                            {#if item.icon === 'kiosk'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h4M4 12h4m12 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z" />
                            {:else if item.icon === 'users'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                            {:else if item.icon === 'book'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
                            {:else if item.icon === 'clipboard'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
                            {:else if item.icon === 'shopping-bag'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M16 11V7a4 4 0 00-8 0v4M5 9h14l1 12H4L5 9z" />
                            {:else if item.icon === 'academic-cap'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M12 14l9-5-9-5-9 5 9 5z" /><path stroke-linecap="round" stroke-linejoin="round" d="M12 14l6.16-3.422a12.083 12.083 0 01.665 6.479A11.952 11.952 0 0012 20.055a11.952 11.952 0 00-6.824-2.998 12.078 12.078 0 01.665-6.479L12 14z" />
                            {:else if item.icon === 'chart-bar'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10a2 2 0 002 2h2a2 2 0 002-2V5a2 2 0 00-2-2h-2a2 2 0 00-2 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                            {:else if item.icon === 'clock'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                            {:else if item.icon === 'identification'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
                            {:else if item.icon === 'printer'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M7 7h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                            {:else if item.icon === 'catalog'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                            {:else if item.icon === 'shield'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75 11.25 15 15 9.75m-3-7.036A11.959 11.959 0 0 1 3.598 6 11.99 11.99 0 0 0 3 9.749c0 5.592 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285Z" />
                            {:else if item.icon === 'bell'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M14.857 17.082a23.848 23.848 0 0 0 5.454-1.31A8.967 8.967 0 0 1 18 9.75V9A6 6 0 0 0 6 9v.75a8.967 8.967 0 0 1-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 0 1-5.714 0m5.714 0a3 3 0 1 1-5.714 0" />
                            {:else if item.icon === 'cog'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.325.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 0 1 1.37.49l1.296 2.247a1.125 1.125 0 0 1-.26 1.431l-1.003.827c-.293.241-.438.613-.43.992a7.723 7.723 0 0 1 0 .255c-.008.378.137.75.43.991l1.004.827c.424.35.534.955.26 1.43l-1.298 2.247a1.125 1.125 0 0 1-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.47 6.47 0 0 1-.22.128c-.331.183-.581.495-.644.869l-.213 1.281c-.09.543-.56.94-1.11.94h-2.594c-.55 0-1.019-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 0 1-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 0 1-1.369-.49l-1.297-2.247a1.125 1.125 0 0 1 .26-1.431l1.004-.827c.292-.24.437-.613.43-.991a6.932 6.932 0 0 1 0-.255c.007-.38-.138-.751-.43-.992l-1.004-.827a1.125 1.125 0 0 1-.26-1.43l1.297-2.247a1.125 1.125 0 0 1 1.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.086.22-.128.332-.183.582-.495.644-.869l.214-1.28Z" /><path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z" />
                            {/if}
                          </svg>
                          {#if !isSidebarCollapsed}
                            <span class="animate-fade-in flex-1 text-left">{item.label}</span>
                            {#if item.id === 'orders' && pendingReservierungen > 0}
                              <span class="ml-auto min-w-5 h-5 flex items-center justify-center rounded-full bg-rose-500 text-white text-[10px] font-bold px-1">{pendingReservierungen}</span>
                            {/if}
                          {:else if item.id === 'orders' && pendingReservierungen > 0}
                            <span class="absolute top-0.5 right-0.5 w-2.5 h-2.5 rounded-full bg-rose-500 ring-2 ring-white"></span>
                          {/if}
                        </button>
                      {/if}
                    {/each}
                  </div>
                {/if}
              {/each}

              {#if sidebarExtensions.length > 0}
                <div class="pt-4 border-t border-slate-100 space-y-1">
                  {#if !isSidebarCollapsed}
                    <span class="px-3 text-[10px] font-bold text-slate-400 uppercase tracking-wider block mb-2">Erweiterungen</span>
                  {/if}
                  {#each sidebarExtensions as ext}
                    {@const Component = ext.component}
                    <Component {...ext.props} collapsed={isSidebarCollapsed} />
                  {/each}
                </div>
              {/if}
            </nav>
          </div>

          <div class="border-t border-slate-100 mt-auto">
            {#if !isSidebarCollapsed}
              <div class="p-4 flex flex-col gap-3 animate-fade-in no-print shrink-0 text-left">
                <div class="flex flex-col">
                  {#if currentUser}
                    <span class="text-xs font-bold text-slate-800 truncate">
                      {currentUser.vorname} {currentUser.nachname}
                    </span>
                    <span class="text-[10px] text-slate-500 font-medium capitalize mt-0.5">
                      Rolle: {currentUser.rolle}
                    </span>
                  {/if}
                </div>
                <button onclick={handleLogout} class="w-full flex items-center justify-center gap-1.5 px-3 py-2 bg-rose-50 hover:bg-rose-100/60 border border-rose-100 text-rose-600 hover:text-rose-700 font-bold text-xs rounded-xl transition-all cursor-pointer">
                  <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"></path></svg>
                  <span>Abmelden</span>
                </button>
              </div>
              <div class="px-4 pb-4 text-center no-print animate-fade-in shrink-0">
                <div class="inline-flex items-center gap-1.5 py-1 px-3 rounded-full bg-emerald-50 border border-emerald-100/50 text-emerald-700 text-[10px] font-semibold tracking-wide">
                  <span>🛡️ DSGVO anonymisiert</span>
                </div>
              </div>
            {:else}
              <div class="p-4 flex flex-col items-center gap-3 no-print shrink-0">
                <div class="w-8 h-8 rounded-full bg-slate-100 border border-slate-200 flex items-center justify-center text-xs font-bold text-slate-650 cursor-default" title="{currentUser?.vorname} {currentUser?.nachname} ({currentUser?.rolle})">
                  {currentUser ? (currentUser.vorname[0] + (currentUser.nachname ? currentUser.nachname[0] : '')) : 'U'}
                </div>
                <button onclick={handleLogout} class="p-2 bg-rose-50 hover:bg-rose-100/60 border border-rose-100 text-rose-600 hover:text-rose-700 rounded-xl transition-all cursor-pointer" title="Abmelden">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"></path></svg>
                </button>
              </div>
              <div class="px-4 pb-4 text-center no-print flex justify-center shrink-0">
                <span class="text-emerald-600 text-sm cursor-default" title="Scans nach 14 Tagen anonymisiert">🛡️</span>
              </div>
            {/if}
          </div>
        </div>
      </aside>

      <div class="flex-1 flex flex-col min-w-0 bg-slate-50 p-8 w-full">

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
            <div class="w-full animate-fade-in"><BestellWorkspace /></div>
          {:else if activeTab === "stats"}
            <div class="w-full animate-fade-in"><StatsDashboard /></div>
          {:else if activeTab === "audit"}
            <div class="w-full animate-fade-in"><AuditLog /></div>
          {:else if activeTab === "student_ids"}
            <div class="w-full animate-fade-in"><StudentIdDesigner /></div>
          {:else if activeTab === "labels"}
            <div class="w-full animate-fade-in"><LabelPrinter /></div>
          {:else if activeTab === "media_catalog"}
            <div class="w-full animate-fade-in"><MediaCatalog /></div>
          {:else if activeTab === "inventory"}
            <div class="w-full animate-fade-in"><UnifiedInventory /></div>
          {:else if activeTab === "students_dir"}
            <div class="w-full animate-fade-in"><StudentDirectory role={currentUser?.rolle} /></div>
          {:else if activeTab === "permissions"}
            <div class="w-full animate-fade-in"><PermissionManager /></div>
          {:else if activeTab === "mahnwesen"}
            <div class="w-full animate-fade-in"><Mahnwesen /></div>
          {:else if activeTab === "lehrer_portal"}
            <div class="w-full animate-fade-in"><LehrerPortal user={currentUser} /></div>
          {:else if activeTab === "settings"}
            <div class="w-full animate-fade-in"><SystemSettings /></div>
          {/if}
        </main>
      </div>
    </div>
  {/if}
{/if}
</main>

<style>
  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }
  @keyframes slideUp {
    from { opacity: 0; transform: translateY(8px); }
    to { opacity: 1; transform: none; }
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
