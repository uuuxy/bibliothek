<script>
  import { apiFetch, apiClient } from "./apiFetch.js";
  import { onMount } from "svelte";
  import UserManagement from "./UserManagement.svelte";

  // Categories and their mapped permissions
  const permissionsMetadata = [
    {
      category: "Schülerverwaltung",
      icon: "👤",
      items: [
        { key: "view_students", label: "Schülerdatei anzeigen", desc: "Erlaubt das Suchen und Einsehen von Schülerdaten und Klassen" },
        { key: "create_students", label: "Schüler hinzufügen", desc: "Ermöglicht das manuelle Anlegen neuer Schüler" },
        { key: "delete_students", label: "Schüler löschen", desc: "Erlaubt das Entfernen von Schülern aus der Datenbank" },
        { key: "import_students", label: "LUSD / CSV Import", desc: "Ermöglicht den Import von Schülerdaten per CSV-Datei" },
        { key: "upload_photos", label: "Ausweisfotos hochladen", desc: "Erlaubt die Aufnahme und Zuweisung von Ausweisfotos per Webcam" }
      ]
    },
    {
      category: "Medien & Inventar",
      icon: "📚",
      items: [
        { key: "view_books", label: "Medienkatalog anzeigen", desc: "Erlaubt das Suchen und Anzeigen von Buchtiteln und Exemplaren" },
        { key: "edit_books", label: "Bücher / Notizen bearbeiten", desc: "Ermöglicht das Hinzufügen von Schadensnotizen an Exemplaren" },
        { key: "delete_books", label: "Bücher & Exemplare löschen", desc: "Erlaubt das Löschen von Exemplaren und Buchtiteln" },
        { key: "inventory_scan", label: "Inventur durchführen", desc: "Ermöglicht das Einscannen von Büchern während einer aktiven Inventur" }
      ]
    },
    {
      category: "Bestellungen & Kiosk",
      icon: "🛒",
      items: [
        { key: "view_orders", label: "Bestellungen anzeigen", desc: "Erlaubt das Einsehen von Buchbestellungen und Lieferanten-Order" },
        { key: "create_orders", label: "Bestellungen verwalten", desc: "Ermöglicht das Bestellen neuer Bücher und Freigeben von Lieferungen" },
        { key: "view_graduates", label: "Abgängerliste einsehen", desc: "Erlaubt das Einsehen von Schulabgängern mit ausstehenden Büchern" }
      ]
    },
    {
      category: "Administration & System",
      icon: "⚙️",
      items: [
        { key: "view_stats", label: "Statistiken anzeigen", desc: "Zeigt Systemstatistiken und Ausleih-Auswertungen" },
        { key: "audit_logs", label: "Sicherheits-Logbuch einsehen", desc: "Ermöglicht den Zugriff auf das Enterprise Audit-Logbuch" },
        { key: "manage_users", label: "Benutzer & Rechte verwalten", desc: "Ermöglicht die Verwaltung von Benutzern und Berechtigungen" }
      ]
    }
  ];

  // State Runes (Svelte 5)
  let activeSubTab = $state("permissions"); // "permissions" | "users"
  
  // Permissions State
  /** @type {Record<string, Record<string, boolean>>} */
  let permissionsState = $state({});
  let loadingPermissions = $state(true);
  
  // Common UI State
  /** @type {string | null} */
  let error = $state(null);
  /** @type {Record<string, boolean>} */
  let updatingKeys = $state({});
  /** @type {string | null} */
  let successMessage = $state(null);

  // Load permissions
  async function fetchPermissions() {
    loadingPermissions = true;
    error = null;
    try {
      const res = await apiFetch("/api/admin/permissions");
      if (!res.ok) {
        if (res.status === 403) throw new Error("Zugriff verweigert: Nur für System-Administratoren.");
        throw new Error(await res.text() || "Fehler beim Laden der Berechtigungen");
      }
      const data = await res.json();
      
      /** @type {Record<string, Record<string, boolean>>} */
      const newState = { admin: {}, mitarbeiter: {}, lehrer: {} };
      data.forEach((/** @type {any} */ item) => {
        if (!newState[item.role]) newState[item.role] = {};
        newState[item.role][item.permission] = item.allowed;
      });
      permissionsState = newState;
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      loadingPermissions = false;
    }
  }

  // Toggle a single permission
  /**
   * @param {string} role
   * @param {string} permission
   * @param {boolean} currentVal
   */
  async function togglePermission(role, permission, currentVal) {
    if (role === "admin") return;
    
    const updateKey = `${role}-${permission}`;
    updatingKeys = { ...updatingKeys, [updateKey]: true };
    const newVal = !currentVal;

    try {
      const res = await apiClient.put("/api/admin/permissions", { role, permission, allowed: newVal });

      if (!res.ok) throw new Error("Fehler beim Speichern der Berechtigung.");
      permissionsState[role][permission] = newVal;
      
      showToast("Rechte erfolgreich aktualisiert.");
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
      setTimeout(() => { error = null; }, 5000);
    } finally {
      const copy = { ...updatingKeys };
      delete copy[updateKey];
      updatingKeys = copy;
    }
  }

  // Helper: Flash message toast
  /** @param {string} msg */
  function showToast(msg) {
    successMessage = msg;
    setTimeout(() => {
      if (successMessage === msg) successMessage = null;
    }, 3000);
  }

  onMount(fetchPermissions);
</script>

<div class="w-full space-y-6 animate-fade-in no-print pb-12">
  <!-- Header & Tab Navigation -->
  <div class="flex flex-col md:flex-row md:items-center md:justify-end gap-4">
    <!-- Sub-tabs pills -->
    <div class="inline-flex p-1 bg-slate-100/70 border border-slate-200/50 rounded-xl self-start shrink-0">
      <button 
        onclick={() => activeSubTab = "permissions"} 
        class="px-4 py-2 text-xs font-semibold rounded-lg transition-all flex items-center gap-2 cursor-pointer {activeSubTab === 'permissions' ? 'bg-white text-slate-800 shadow-xs' : 'text-slate-500 hover:text-slate-800'}"
      >
        🛡️ Rechte
      </button>
      <button 
        onclick={() => activeSubTab = "users"} 
        class="px-4 py-2 text-xs font-semibold rounded-lg transition-all flex items-center gap-2 cursor-pointer {activeSubTab === 'users' ? 'bg-white text-slate-800 shadow-xs' : 'text-slate-500 hover:text-slate-800'}"
      >
        👥 Benutzer
      </button>
    </div>
  </div>

  <!-- Error Alerts -->
  {#if error}
    <div class="p-4 rounded-2xl bg-rose-50 border border-rose-100 text-rose-600 text-sm font-medium transition-all animate-slide-up flex items-center justify-between">
      <span>⚠️ {error}</span>
      <button onclick={() => error = null} class="text-rose-450 hover:text-rose-650 font-bold ml-2">×</button>
    </div>
  {/if}

  <!-- Success Toast -->
  {#if successMessage}
    <div class="fixed bottom-6 right-6 z-50 p-4 rounded-xl bg-emerald-50 border border-emerald-100 text-emerald-700 text-xs font-semibold shadow-lg transition-all animate-slide-up flex items-center gap-2">
      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-emerald-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
        <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
      </svg>
      <span>{successMessage}</span>
    </div>
  {/if}

  <!-- Tab 1: Permissions Editor -->
  {#if activeSubTab === "permissions"}
    {#if loadingPermissions}
      <div class="p-12 text-center text-slate-400 font-medium animate-pulse">Lade Rechtekonfiguration...</div>
    {:else}
      <div class="space-y-8">
        {#each permissionsMetadata as cat}
          <div class="border border-slate-100 bg-white rounded-3xl overflow-hidden shadow-xs">
            <div class="p-5 bg-slate-50/70 border-b border-slate-100 flex items-center gap-3">
              <span class="text-xl">{cat.icon}</span>
              <h3 class="font-bold text-slate-800 text-sm tracking-tight">{cat.category}</h3>
            </div>
            
            <div class="divide-y divide-slate-100">
              {#each cat.items as item}
                {@const isLUpdate = updatingKeys[`lehrer-${item.key}`]}
                {@const isMUpdate = updatingKeys[`mitarbeiter-${item.key}`]}
                
                <div class="p-5 flex flex-col md:flex-row md:items-center justify-between gap-4 hover:bg-slate-50/30 transition-colors">
                  <div class="max-w-md space-y-1">
                    <span class="font-semibold text-slate-850 text-sm tracking-tight">{item.label}</span>
                    <p class="text-xs text-slate-450 leading-relaxed font-medium">{item.desc}</p>
                  </div>
                  
                  <div class="flex items-center gap-8 md:gap-12 shrink-0">
                    <!-- Admin (Read-only status display) -->
                    <div class="flex items-center gap-3">
                      <span class="text-[10px] font-bold text-slate-400 tracking-wider w-16 text-right">ADMIN</span>
                      <label class="relative inline-flex items-center opacity-60 cursor-not-allowed">
                        <input type="checkbox" checked disabled class="sr-only peer" />
                        <div class="w-10 h-6 bg-blue-100 rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
                      </label>
                    </div>

                    <!-- Mitarbeiter -->
                    <div class="flex items-center gap-3">
                      <span class="text-[10px] font-bold text-slate-500 tracking-wider w-16 text-right">MITARBEITER</span>
                      <button 
                        onclick={() => togglePermission("mitarbeiter", item.key, permissionsState.mitarbeiter?.[item.key] ?? false)}
                        disabled={isMUpdate}
                        class="relative inline-flex items-center cursor-pointer group focus:outline-none"
                        aria-label="Mitarbeiter Rechte umschalten"
                      >
                        <input type="checkbox" checked={permissionsState.mitarbeiter?.[item.key] ?? false} class="sr-only peer" readonly />
                        <div class="w-10 h-6 bg-slate-200 rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-350 after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600 peer-focus:ring-2 peer-focus:ring-blue-500/20"></div>
                        {#if isMUpdate}
                          <div class="absolute inset-0 flex items-center justify-center bg-white/70 rounded-full">
                            <div class="w-3.5 h-3.5 border-2 border-slate-500 border-t-transparent rounded-full animate-spin"></div>
                          </div>
                        {/if}
                      </button>
                    </div>

                    <!-- Lehrer -->
                    <div class="flex items-center gap-3">
                      <span class="text-[10px] font-bold text-slate-500 tracking-wider w-16 text-right">LEHRER</span>
                      <button 
                        onclick={() => togglePermission("lehrer", item.key, permissionsState.lehrer?.[item.key] ?? false)}
                        disabled={isLUpdate}
                        class="relative inline-flex items-center cursor-pointer group focus:outline-none"
                        aria-label="Lehrer Rechte umschalten"
                      >
                        <input type="checkbox" checked={permissionsState.lehrer?.[item.key] ?? false} class="sr-only peer" readonly />
                        <div class="w-10 h-6 bg-slate-200 rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-350 after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600 peer-focus:ring-2 peer-focus:ring-blue-500/20"></div>
                        {#if isLUpdate}
                          <div class="absolute inset-0 flex items-center justify-center bg-white/70 rounded-full">
                            <div class="w-3.5 h-3.5 border-2 border-slate-500 border-t-transparent rounded-full animate-spin"></div>
                          </div>
                        {/if}
                      </button>
                    </div>
                  </div>
                </div>
              {/each}
            </div>
          </div>
        {/each}
      </div>
    {/if}
  {/if}

  <!-- Tab 2: User Management -->
  {#if activeSubTab === "users"}
    <UserManagement />
  {/if}
</div>


<style>
  .peer-checked\:bg-blue-600:checked + div {
    background-color: rgb(37 99 235);
  }
</style>
