<script>
  import { onMount } from "svelte";

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
  
  // Users State
  /** @type {any[]} */
  let users = $state([]);
  let loadingUsers = $state(false);
  let userSearchQuery = $state("");

  // Common UI State
  /** @type {string | null} */
  let error = $state(null);
  /** @type {Record<string, boolean>} */
  let updatingKeys = $state({});
  /** @type {string | null} */
  let successMessage = $state(null);

  // User Dialog Modal State
  let showUserModal = $state(false);
  let isEditingUser = $state(false);
  /** @type {any} */
  let userForm = $state({
    id: "",
    barcode_id: "",
    vorname: "",
    nachname: "",
    email: "",
    rolle: "mitarbeiter",
    aktiv: true,
    password: ""
  });
  let submittingUser = $state(false);
  
  // Delete Confirmation State
  let showDeleteConfirm = $state(false);
  /** @type {any} */
  let userToDelete = $state(null);
  let deletingUser = $state(false);

  // Load permissions
  async function fetchPermissions() {
    loadingPermissions = true;
    error = null;
    try {
      const res = await fetch("/api/admin/permissions");
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

  // Load users
  async function fetchUsers() {
    loadingUsers = true;
    error = null;
    try {
      const res = await fetch("/api/benutzer");
      if (!res.ok) {
        if (res.status === 403) throw new Error("Zugriff verweigert: Nur für System-Administratoren.");
        throw new Error(await res.text() || "Fehler beim Laden der Benutzer");
      }
      users = await res.json();
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      loadingUsers = false;
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
      const res = await fetch("/api/admin/permissions", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ role, permission, allowed: newVal })
      });

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

  // Save user details (Create or Update)
  /** @param {SubmitEvent} e */
  async function handleSaveUser(e) {
    e.preventDefault();
    submittingUser = true;
    error = null;

    try {
      const url = isEditingUser ? `/api/benutzer/${userForm.id}` : "/api/benutzer";
      const method = isEditingUser ? "PUT" : "POST";
      
      // Build payload
      const payload = {
        barcode_id: userForm.barcode_id,
        vorname: userForm.vorname,
        nachname: userForm.nachname,
        email: userForm.email,
        rolle: userForm.rolle,
        aktiv: userForm.aktiv,
        password: userForm.password
      };

      const res = await fetch(url, {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload)
      });

      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || "Fehler beim Speichern des Benutzers.");
      }

      showUserModal = false;
      showToast(isEditingUser ? "Benutzer erfolgreich aktualisiert." : "Benutzer erfolgreich angelegt.");
      fetchUsers();
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      submittingUser = false;
    }
  }

  // Delete User
  async function confirmDeleteUser() {
    if (!userToDelete) return;
    deletingUser = true;
    error = null;
    try {
      const res = await fetch(`/api/benutzer/${userToDelete.id}`, { method: "DELETE" });
      if (!res.ok) throw new Error(await res.text() || "Benutzer konnte nicht gelöscht werden.");
      
      showDeleteConfirm = false;
      userToDelete = null;
      showToast("Benutzer erfolgreich gelöscht.");
      fetchUsers();
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
      setTimeout(() => { error = null; }, 5000);
    } finally {
      deletingUser = false;
    }
  }

  // Helper: Open Modal for New User
  function openNewUserModal() {
    isEditingUser = false;
    userForm = {
      id: "",
      barcode_id: "",
      vorname: "",
      nachname: "",
      email: "",
      rolle: "mitarbeiter",
      aktiv: true,
      password: ""
    };
    error = null;
    showUserModal = true;
  }

  // Helper: Open Modal for Editing User
  /** @param {any} user */
  function openEditUserModal(user) {
    isEditingUser = true;
    userForm = {
      id: user.id,
      barcode_id: user.barcode_id || "",
      vorname: user.vorname,
      nachname: user.nachname,
      email: user.email,
      rolle: user.rolle,
      aktiv: user.aktiv,
      password: "" // Don't prefill password
    };
    error = null;
    showUserModal = true;
  }

  // Helper: Open delete confirmation dialog
  /** @param {any} user */
  function openDeleteConfirm(user) {
    userToDelete = user;
    showDeleteConfirm = true;
  }

  // Helper: Flash message toast
  /** @param {string} msg */
  function showToast(msg) {
    successMessage = msg;
    setTimeout(() => {
      if (successMessage === msg) successMessage = null;
    }, 3000);
  }

  // Derived filtered users list (Svelte 5 runic derivation)
  let filteredUsers = $derived.by(() => {
    const query = userSearchQuery.trim().toLowerCase();
    if (!query) return users;
    return users.filter(u => 
      u.vorname.toLowerCase().includes(query) ||
      u.nachname.toLowerCase().includes(query) ||
      u.email.toLowerCase().includes(query) ||
      (u.barcode_id && u.barcode_id.toLowerCase().includes(query)) ||
      u.rolle.toLowerCase().includes(query)
    );
  });

  onMount(() => {
    fetchPermissions();
    fetchUsers();
  });
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
  {#if error && !showUserModal && !showDeleteConfirm}
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
                      <span class="text-[10px] font-bold text-slate-400 font-mono tracking-wider w-16 text-right">ADMIN</span>
                      <label class="relative inline-flex items-center opacity-60 cursor-not-allowed">
                        <input type="checkbox" checked disabled class="sr-only peer" />
                        <div class="w-10 h-6 bg-blue-100 rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
                      </label>
                    </div>

                    <!-- Mitarbeiter -->
                    <div class="flex items-center gap-3">
                      <span class="text-[10px] font-bold text-slate-500 font-mono tracking-wider w-16 text-right">MITARBEITER</span>
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
                      <span class="text-[10px] font-bold text-slate-500 font-mono tracking-wider w-16 text-right">LEHRER</span>
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

  <!-- Tab 2: User CRUD Administration -->
  {#if activeSubTab === "users"}
    <div class="space-y-4">
      <!-- Toolbar: Search and Create User -->
      <div class="flex flex-col sm:flex-row items-center justify-between gap-4">
        <div class="relative w-full sm:max-w-xs">
          <input 
            type="text" 
            bind:value={userSearchQuery} 
            placeholder="Benutzer suchen..." 
            class="w-full bg-white border border-slate-200 rounded-xl py-2 px-3 pl-9 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400 transition-all font-medium text-slate-800"
          />
          <span class="absolute left-3 top-2.5 text-slate-400">🔍</span>
        </div>

        <button 
          onclick={openNewUserModal} 
          class="w-full sm:w-auto px-4 py-2 text-xs font-bold text-white bg-blue-600 hover:bg-blue-700 rounded-xl transition-all shadow-xs flex items-center justify-center gap-1.5 cursor-pointer"
        >
          ➕ Benutzer anlegen
        </button>
      </div>

      {#if loadingUsers}
        <div class="p-12 text-center text-slate-400 font-medium animate-pulse">Lade Systembenutzer...</div>
      {:else if filteredUsers.length === 0}
        <div class="p-12 rounded-3xl border border-dashed border-slate-200 bg-white text-center text-slate-400">
          <span class="text-2xl block mb-2">👥</span>
          Keine Systembenutzer gefunden.
        </div>
      {:else}
        <div class="border border-slate-100 bg-white rounded-3xl overflow-hidden shadow-xs">
          <div class="overflow-x-auto">
            <table class="w-full text-left border-collapse">
              <thead>
                <tr class="bg-slate-50 border-b border-slate-100 text-xs font-bold text-slate-400 uppercase font-mono tracking-wider">
                  <th class="p-4">Name</th>
                  <th class="p-4">E-Mail</th>
                  <th class="p-4">Barcode</th>
                  <th class="p-4">Rolle</th>
                  <th class="p-4">Status</th>
                  <th class="p-4 text-right">Aktionen</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-100 text-sm text-slate-600 font-medium">
                {#each filteredUsers as user}
                  <tr class="hover:bg-slate-50/50 transition-colors">
                    <td class="p-4">
                      <span class="font-semibold text-slate-800">{user.vorname} {user.nachname}</span>
                    </td>
                    <td class="p-4 text-slate-500 font-mono text-xs">{user.email}</td>
                    <td class="p-4">
                      {#if user.barcode_id}
                        <span class="font-mono text-xs bg-slate-50 border border-slate-200/60 text-slate-600 py-0.5 px-2 rounded-md">{user.barcode_id}</span>
                      {:else}
                        <span class="text-xs text-slate-400 italic">Keine</span>
                      {/if}
                    </td>
                    <td class="p-4">
                      <span class="inline-flex px-2 py-0.5 rounded-md font-bold text-xs uppercase tracking-wide
                        {user.rolle === 'admin' ? 'bg-blue-50 text-blue-700 border border-blue-100' : ''}
                        {user.rolle === 'mitarbeiter' ? 'bg-amber-50 text-amber-700 border border-amber-100' : ''}
                        {user.rolle === 'lehrer' ? 'bg-emerald-50 text-emerald-700 border border-emerald-100' : ''}
                      ">
                        {user.rolle}
                      </span>
                    </td>
                    <td class="p-4">
                      {#if user.aktiv}
                        <span class="inline-flex items-center gap-1.5 text-xs text-emerald-600">
                          <span class="w-1.5 h-1.5 rounded-full bg-emerald-500"></span> Aktiv
                        </span>
                      {:else}
                        <span class="inline-flex items-center gap-1.5 text-xs text-slate-400">
                          <span class="w-1.5 h-1.5 rounded-full bg-slate-350"></span> Inaktiv
                        </span>
                      {/if}
                    </td>
                    <td class="p-4 text-right space-x-2 shrink-0">
                      <button 
                        onclick={() => openEditUserModal(user)} 
                        class="px-2.5 py-1 text-xs font-semibold text-slate-600 bg-slate-50 border border-slate-200 rounded-lg hover:bg-slate-100 hover:text-slate-800 transition-colors cursor-pointer"
                      >
                        Bearbeiten
                      </button>
                      <button 
                        onclick={() => openDeleteConfirm(user)} 
                        class="px-2.5 py-1 text-xs font-semibold text-rose-600 bg-rose-50 border border-rose-100 rounded-lg hover:bg-rose-100 transition-colors cursor-pointer"
                      >
                        Löschen
                      </button>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        </div>
      {/if}
    </div>
  {/if}
</div>

<!-- Modal Dialog: Create / Edit User -->
{#if showUserModal}
  <div class="fixed inset-0 bg-slate-900/40 backdrop-blur-xs z-50 flex items-center justify-center p-4 animate-fade-in">
    <div class="bg-white border border-slate-200 w-full max-w-md rounded-3xl shadow-2xl overflow-hidden animate-scale-up">
      <div class="p-6 border-b border-slate-100 bg-slate-550/5 flex items-center justify-between">
        <h3 class="font-bold text-slate-800 text-sm">{isEditingUser ? "Benutzer bearbeiten" : "Neuen Benutzer anlegen"}</h3>
        <button onclick={() => showUserModal = false} class="text-slate-400 hover:text-slate-600 font-bold text-lg cursor-pointer">×</button>
      </div>

      <form onsubmit={handleSaveUser} class="p-6 space-y-4">
        {#if error}
          <div class="p-3.5 rounded-xl bg-rose-50 border border-rose-100 text-rose-600 text-xs font-semibold leading-relaxed animate-slide-up">
            ⚠️ {error}
          </div>
        {/if}

        <div class="grid grid-cols-2 gap-4">
          <div class="space-y-1.5">
            <label for="vorname" class="block text-xs font-bold text-slate-400 uppercase tracking-wider">Vorname</label>
            <input 
              id="vorname"
              type="text" 
              bind:value={userForm.vorname} 
              required 
              class="w-full bg-slate-50 border border-slate-200 rounded-xl py-2.5 px-3 text-xs focus:outline-none focus:ring-2 focus:ring-blue-500/10 focus:border-blue-300 transition-all font-medium text-slate-800"
            />
          </div>
          <div class="space-y-1.5">
            <label for="nachname" class="block text-xs font-bold text-slate-400 uppercase tracking-wider">Nachname</label>
            <input 
              id="nachname"
              type="text" 
              bind:value={userForm.nachname} 
              required 
              class="w-full bg-slate-50 border border-slate-200 rounded-xl py-2.5 px-3 text-xs focus:outline-none focus:ring-2 focus:ring-blue-500/10 focus:border-blue-300 transition-all font-medium text-slate-800"
            />
          </div>
        </div>

        <div class="space-y-1.5">
          <label for="email" class="block text-xs font-bold text-slate-400 uppercase tracking-wider">E-Mail Adresse</label>
          <input 
            id="email"
            type="email" 
            bind:value={userForm.email} 
            required 
            class="w-full bg-slate-50 border border-slate-200 rounded-xl py-2.5 px-3 text-xs focus:outline-none focus:ring-2 focus:ring-blue-500/10 focus:border-blue-300 transition-all font-medium text-slate-800"
          />
        </div>

        <div class="space-y-1.5">
          <label for="barcode_id" class="block text-xs font-bold text-slate-400 uppercase tracking-wider">Barcode (Anmelde-ID)</label>
          <input 
            id="barcode_id"
            type="text" 
            bind:value={userForm.barcode_id} 
            placeholder="Z. B. L-001, MA-04 (optional)"
            class="w-full bg-slate-50 border border-slate-200 rounded-xl py-2.5 px-3 text-xs focus:outline-none focus:ring-2 focus:ring-blue-500/10 focus:border-blue-300 transition-all font-medium text-slate-800"
          />
        </div>

        <div class="space-y-1.5">
          <label for="rolle" class="block text-xs font-bold text-slate-400 uppercase tracking-wider">Benutzer-Rolle</label>
          <select 
            id="rolle"
            bind:value={userForm.rolle} 
            class="w-full bg-slate-50 border border-slate-200 rounded-xl py-2.5 px-3 text-xs focus:outline-none focus:ring-2 focus:ring-blue-500/10 focus:border-blue-300 transition-all font-medium text-slate-800"
          >
            <option value="mitarbeiter">Mitarbeiter</option>
            <option value="lehrer">Lehrer</option>
            <option value="admin">Administrator</option>
          </select>
        </div>

        <div class="space-y-1.5">
          <label for="password" class="block text-xs font-bold text-slate-400 uppercase tracking-wider">
            {isEditingUser ? "Passwort ändern" : "Passwort"}
          </label>
          <input 
            id="password"
            type="password" 
            bind:value={userForm.password} 
            required={!isEditingUser}
            placeholder={isEditingUser ? "Unverändert lassen..." : "••••••••"}
            class="w-full bg-slate-50 border border-slate-200 rounded-xl py-2.5 px-3 text-xs focus:outline-none focus:ring-2 focus:ring-blue-500/10 focus:border-blue-300 transition-all font-medium text-slate-800"
          />
        </div>

        {#if isEditingUser}
          <div class="flex items-center gap-3 py-1.5">
            <label class="relative inline-flex items-center cursor-pointer">
              <input type="checkbox" bind:checked={userForm.aktiv} class="sr-only peer" />
              <div class="w-10 h-6 bg-slate-200 rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-350 after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
            </label>
            <span class="text-xs font-bold text-slate-650">Benutzerkonto ist aktiv</span>
          </div>
        {/if}

        <div class="flex items-center justify-end gap-3 pt-3 border-t border-slate-100">
          <button 
            type="button" 
            onclick={() => showUserModal = false} 
            class="px-4 py-2 text-xs font-semibold border border-slate-200 rounded-xl hover:bg-slate-50 cursor-pointer"
          >
            Abbrechen
          </button>
          <button 
            type="submit" 
            disabled={submittingUser}
            class="px-4 py-2 text-xs font-bold text-white bg-blue-600 hover:bg-blue-700 rounded-xl shadow-xs transition-colors flex items-center justify-center cursor-pointer disabled:opacity-60"
          >
            {#if submittingUser}
              <div class="w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full animate-spin mr-1"></div>
            {/if}
            Speichern
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}

<!-- Modal Dialog: Delete Confirmation -->
{#if showDeleteConfirm && userToDelete}
  <div class="fixed inset-0 bg-slate-900/40 backdrop-blur-xs z-50 flex items-center justify-center p-4 animate-fade-in">
    <div class="bg-white border border-slate-200 w-full max-w-sm rounded-3xl shadow-2xl overflow-hidden animate-scale-up">
      <div class="p-6 space-y-4">
        <div class="w-12 h-12 rounded-full bg-rose-50 border border-rose-150 text-rose-600 flex items-center justify-center text-xl mx-auto">
          ⚠️
        </div>
        <div class="text-center space-y-1.5">
          <h3 class="font-bold text-slate-800 text-sm">Benutzer unwiderruflich löschen?</h3>
          <p class="text-xs text-slate-450 leading-relaxed font-medium">
            Sind Sie sicher, dass Sie den Benutzer <strong>{userToDelete.vorname} {userToDelete.nachname}</strong> löschen möchten? Diese Aktion wird im Logbuch vermerkt.
          </p>
        </div>

        <div class="flex items-center justify-center gap-3 pt-3 border-t border-slate-100">
          <button 
            onclick={() => showDeleteConfirm = false} 
            class="px-4 py-2 text-xs font-semibold border border-slate-200 rounded-xl hover:bg-slate-50 cursor-pointer"
            disabled={deletingUser}
          >
            Abbrechen
          </button>
          <button 
            onclick={confirmDeleteUser} 
            disabled={deletingUser}
            class="px-4 py-2 text-xs font-bold text-white bg-rose-600 hover:bg-rose-700 rounded-xl shadow-xs transition-colors flex items-center justify-center cursor-pointer"
          >
            {#if deletingUser}
              <div class="w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full animate-spin mr-1"></div>
            {/if}
            Löschen
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}

<style>
  .peer-checked\:bg-blue-600:checked + div {
    background-color: rgb(37 99 235);
  }
</style>
