<script>
  /**
   * UserManagement — self-contained component for staff user CRUD.
   *
   * Responsibilities: listing, creating, editing, and deleting system users
   * (benutzer/benutzer_rollen tables). Isolated from role-permission management.
   *
   * State:
   *   $state users         — loaded from /api/benutzer
   *   $state userForm      — controlled form for create/edit modal
   *   $state showUserModal — controls the create/edit dialog
   *   $state showDeleteConfirm — controls the delete confirmation dialog
   *   $derived filteredUsers — search-filtered view of users
   */
  import { onMount } from "svelte";
  import Modal from "./Modal.svelte";
  import { apiFetch, apiClient } from "./apiFetch.js";

  /** @type {any[]} */
  let users = $state.raw([]);
  let loadingUsers = $state(false);
  let userSearchQuery = $state("");

  /** @type {string | null} */
  let error = $state(null);
  /** @type {string | null} */
  let successMessage = $state(null);

  // Create / Edit modal state
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

  // Delete confirmation state
  let showDeleteConfirm = $state(false);
  /** @type {any} */
  let userToDelete = $state(null);
  let deletingUser = $state(false);

  // Search-filtered view — recomputed reactively whenever users or query changes
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

  async function fetchUsers() {
    loadingUsers = true;
    error = null;
    try {
      const res = await apiFetch("/api/benutzer");
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

  /** @param {SubmitEvent} e */
  async function handleSaveUser(e) {
    e.preventDefault();
    submittingUser = true;
    error = null;
    try {
      const url = isEditingUser ? `/api/benutzer/${userForm.id}` : "/api/benutzer";
      const method = isEditingUser ? "PUT" : "POST";
      const payload = {
        barcode_id: userForm.barcode_id,
        vorname: userForm.vorname,
        nachname: userForm.nachname,
        email: userForm.email,
        rolle: userForm.rolle,
        aktiv: userForm.aktiv,
        password: userForm.password
      };
      const res = await apiFetch(url, {
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

  async function confirmDeleteUser() {
    if (!userToDelete) return;
    deletingUser = true;
    error = null;
    try {
      const res = await apiFetch(`/api/benutzer/${userToDelete.id}`, { method: "DELETE" });
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

  function openNewUserModal() {
    isEditingUser = false;
    userForm = { id: "", barcode_id: "", vorname: "", nachname: "", email: "", rolle: "mitarbeiter", aktiv: true, password: "" };
    error = null;
    showUserModal = true;
  }

  /** @param {any} user */
  function openEditUserModal(user) {
    isEditingUser = true;
    userForm = { id: user.id, barcode_id: user.barcode_id || "", vorname: user.vorname, nachname: user.nachname, email: user.email, rolle: user.rolle, aktiv: user.aktiv, password: "" };
    error = null;
    showUserModal = true;
  }

  /** @param {any} user */
  function openDeleteConfirm(user) {
    userToDelete = user;
    showDeleteConfirm = true;
  }

  /** @param {string} msg */
  function showToast(msg) {
    successMessage = msg;
    setTimeout(() => { if (successMessage === msg) successMessage = null; }, 3000);
  }

  onMount(fetchUsers);
</script>

<!-- Error Banner -->
{#if error && !showUserModal && !showDeleteConfirm}
  <div class="p-4 rounded-2xl bg-rose-50 border border-rose-100 text-rose-600 text-sm font-medium animate-slide-up flex items-center justify-between">
    <span>⚠️ {error}</span>
    <button onclick={() => error = null} class="text-rose-450 hover:text-rose-650 font-bold ml-2">×</button>
  </div>
{/if}

<!-- Success Toast -->
{#if successMessage}
  <div class="fixed bottom-6 right-6 z-50 p-4 rounded-xl bg-emerald-50 border border-emerald-100 text-emerald-700 text-xs font-semibold shadow-lg animate-slide-up flex items-center gap-2">
    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-emerald-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
      <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
    </svg>
    <span>{successMessage}</span>
  </div>
{/if}

<!-- Toolbar -->
<div class="flex flex-col sm:flex-row items-center justify-between gap-4 mb-4">
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

<!-- User Table -->
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
          <tr class="bg-slate-50 border-b border-slate-100 text-xs font-bold text-slate-400 uppercase tracking-wider">
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
            {@const roleBadge = user.rolle === 'admin'
              ? 'bg-blue-50 text-blue-700 border border-blue-100'
              : user.rolle === 'lehrer'
                ? 'bg-emerald-50 text-emerald-700 border border-emerald-100'
                : user.rolle === 'helfer'
                  ? 'bg-purple-50 text-purple-700 border border-purple-100'
                  : 'bg-amber-50 text-amber-700 border border-amber-100'}
            <tr class="hover:bg-slate-50/50 transition-colors">
              <td class="p-4"><span class="font-semibold text-slate-800">{user.vorname} {user.nachname}</span></td>
              <td class="p-4 text-slate-500 text-xs">{user.email}</td>
              <td class="p-4">
                {#if user.barcode_id}
                  <span class="text-xs bg-slate-50 border border-slate-200/60 text-slate-600 py-0.5 px-2 rounded-md">{user.barcode_id}</span>
                {:else}
                  <span class="text-xs text-slate-400 italic">Keine</span>
                {/if}
              </td>
              <td class="p-4">
                <span class="inline-flex px-2 py-0.5 rounded-md font-bold text-xs uppercase tracking-wide {roleBadge}">
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

<!-- Modal: Create / Edit User -->
<Modal open={showUserModal} onclose={() => showUserModal = false} size="md">
  {#snippet header()}
    <h3 class="font-bold text-slate-800 text-sm">{isEditingUser ? "Benutzer bearbeiten" : "Neuen Benutzer anlegen"}</h3>
  {/snippet}
  {#snippet children()}
    <form onsubmit={handleSaveUser} class="p-6 space-y-4">
      {#if error}
        <div class="p-3.5 rounded-xl bg-rose-50 border border-rose-100 text-rose-600 text-xs font-semibold leading-relaxed animate-slide-up">⚠️ {error}</div>
      {/if}
      <div class="grid grid-cols-2 gap-4">
        {@render inputField("vorname", "Vorname", "text", userForm.vorname, (/** @type {any} */ v) => userForm.vorname = v, true)}
        {@render inputField("nachname", "Nachname", "text", userForm.nachname, (/** @type {any} */ v) => userForm.nachname = v, true)}
      </div>
      {@render inputField("email", "E-Mail Adresse", "email", userForm.email, (/** @type {any} */ v) => userForm.email = v, true)}
      {@render inputField("barcode_id", "Barcode (Anmelde-ID)", "text", userForm.barcode_id, (/** @type {any} */ v) => userForm.barcode_id = v, false, "Z. B. L-001, MA-04 (optional)")}
      <div class="space-y-1.5">
        <label for="rolle" class="block text-xs font-bold text-slate-400 uppercase tracking-wider">Benutzer-Rolle</label>
        <select id="rolle" bind:value={userForm.rolle} class="w-full bg-slate-50 border border-slate-200 rounded-xl py-2.5 px-3 text-xs focus:outline-none focus:ring-2 focus:ring-blue-500/10 focus:border-blue-300 transition-all font-medium text-slate-800">
          <option value="mitarbeiter">Mitarbeiter</option>
          <option value="lehrer">Lehrer</option>
          <option value="admin">Administrator</option>
          <option value="helfer">Helfer</option>
        </select>
      </div>
      {@render inputField("password", isEditingUser ? "Passwort ändern" : "Passwort", "password", userForm.password, (/** @type {any} */ v) => userForm.password = v, !isEditingUser, isEditingUser ? "Unverändert lassen..." : "••••••••")}
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
        <button type="button" onclick={() => showUserModal = false} class="px-4 py-2 text-xs font-semibold border border-slate-200 rounded-xl hover:bg-slate-50 cursor-pointer">Abbrechen</button>
        <button type="submit" disabled={submittingUser} class="px-4 py-2 text-xs font-bold text-white bg-blue-600 hover:bg-blue-700 rounded-xl shadow-xs transition-colors flex items-center justify-center cursor-pointer disabled:opacity-60">
          {#if submittingUser}<div class="w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full animate-spin mr-1"></div>{/if}
          Speichern
        </button>
      </div>
    </form>
  {/snippet}
</Modal>

<!-- Modal: Delete Confirmation -->
<Modal open={showDeleteConfirm && !!userToDelete} onclose={() => showDeleteConfirm = false} size="sm">
  {#snippet children()}
    <div class="p-6 space-y-4">
      <div class="w-12 h-12 rounded-full bg-rose-50 border border-rose-150 text-rose-600 flex items-center justify-center text-xl mx-auto">⚠️</div>
      <div class="text-center space-y-1.5">
        <h3 class="font-bold text-slate-800 text-sm">Benutzer unwiderruflich löschen?</h3>
        <p class="text-xs text-slate-450 leading-relaxed font-medium">
          Sind Sie sicher, dass Sie den Benutzer <strong>{userToDelete?.vorname} {userToDelete?.nachname}</strong> löschen möchten? Diese Aktion wird im Logbuch vermerkt.
        </p>
      </div>
      <div class="flex items-center justify-center gap-3 pt-3 border-t border-slate-100">
        <button onclick={() => showDeleteConfirm = false} disabled={deletingUser} class="px-4 py-2 text-xs font-semibold border border-slate-200 rounded-xl hover:bg-slate-50 cursor-pointer">Abbrechen</button>
        <button onclick={confirmDeleteUser} disabled={deletingUser} class="px-4 py-2 text-xs font-bold text-white bg-rose-600 hover:bg-rose-700 rounded-xl shadow-xs transition-colors flex items-center justify-center cursor-pointer">
          {#if deletingUser}<div class="w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full animate-spin mr-1"></div>{/if}
          Löschen
        </button>
      </div>
    </div>
  {/snippet}
</Modal>

{#snippet inputField(id, label, type, value, onInput, required, placeholder)}
  <div class="space-y-1.5">
    <label for={id} class="block text-xs font-bold text-slate-400 uppercase tracking-wider">{label}</label>
    <input
      {id}
      {type}
      value={value}
      oninput={e => onInput(e.currentTarget.value)}
      required={required}
      placeholder={placeholder ?? ""}
      class="w-full bg-slate-50 border border-slate-200 rounded-xl py-2.5 px-3 text-xs focus:outline-none focus:ring-2 focus:ring-blue-500/10 focus:border-blue-300 transition-all font-medium text-slate-800"
    />
  </div>
{/snippet}
