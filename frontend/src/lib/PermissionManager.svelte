<script>
  import { apiFetch, apiClient } from "./apiFetch.js";
  import { onMount } from "svelte";
  import UserManagement from "./UserManagement.svelte";
  import PermissionsEditor from "./PermissionsEditor.svelte";
  import { permissionsMetadata } from "./permissionMetadata.js";

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
      <PermissionsEditor metadata={permissionsMetadata} {permissionsState} {updatingKeys} onToggle={togglePermission} />
    {/if}
  {/if}

  <!-- Tab 2: User Management -->
  {#if activeSubTab === "users"}
    <UserManagement />
  {/if}
</div>


