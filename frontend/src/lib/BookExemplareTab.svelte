<script>
  import { appState, showToast } from "../inventur/lib/store.svelte.js";
  import { apiFetch } from "./apiFetch.js";
  import BookExemplarCard from "./components/BookExemplarCard.svelte";

  /** @type {{ exemplare: any[], book: any, loadAll: (id: string) => void }} */
  let { exemplare = $bindable([]), book, loadAll } = $props();

  let selectedExemplare = $state(new Set());

  /** @param {string} id */
  function toggleSelect(id) {
    if (selectedExemplare.has(id)) {
      const newSet = new Set(selectedExemplare);
      newSet.delete(id);
      selectedExemplare = newSet;
    } else {
      selectedExemplare = new Set(selectedExemplare).add(id);
    }
  }

  /** @param {any} ex */
  async function deleteCopy(ex) {
    if (!confirm(`Möchtest du das Exemplar ${ex.barcode_id} wirklich unwiderruflich löschen?`)) return;
    try {
      const res = await apiFetch(`/api/buecher/exemplare/${ex.id}`, { method: "DELETE", credentials: "include" });
      if (res.ok) {
        exemplare = exemplare.filter((e) => e.id !== ex.id);
        if (book) {
          book.gesamt = Math.max(0, (book.gesamt || 0) - 1);
          if (book.verfuegbar !== undefined && ex.ist_ausleihbar) {
             book.verfuegbar = Math.max(0, (book.verfuegbar || 0) - 1);
          }
        }
        showToast("Exemplar erfolgreich gelöscht", "success");
      } else {
        const err = await res.json().catch(() => ({}));
        alert(err.error || "Fehler beim Löschen des Exemplars.");
      }
    } catch (e) {
      alert("Netzwerkfehler beim Löschen.");
    }
  }

  async function deleteSelectedCopies() {
    if (selectedExemplare.size === 0) return;
    if (!confirm(`Möchtest du die ${selectedExemplare.size} ausgewählten Exemplare unwiderruflich löschen?`)) return;

    try {
      const res = await apiFetch(`/api/buecher/exemplare/bulk-delete`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify({ copy_ids: Array.from(selectedExemplare) }),
        credentials: "include"
      });

      const data = await res.json().catch(() => ({}));

      if (!res.ok && (!data.deleted_ids || data.deleted_ids.length === 0)) {
        alert(data.error || "Fehler beim Löschen der Exemplare.");
        return;
      }

      const deletedIds = data.deleted_ids || [];
      const deletedSet = new Set(deletedIds);

      if (deletedIds.length > 0) {
        exemplare = exemplare.filter((e) => !deletedSet.has(e.id));
        showToast(`${deletedIds.length} Exemplare erfolgreich gelöscht`, "success");
        if (book && book.id) loadAll(book.id);
      }

      if (data.errors && Object.keys(data.errors).length > 0) {
        console.error("Fehler beim Löschen einiger Exemplare:", data.errors);
        alert(`Einige Exemplare konnten nicht gelöscht werden. Überprüfe die Konsole für Details.`);
      }

      // Deselect only the ones that were successfully deleted
      const remainingSelected = new Set();
      for (const id of selectedExemplare) {
        if (!deletedSet.has(id)) {
          remainingSelected.add(id);
        }
      }
      selectedExemplare = remainingSelected;

    } catch (e) {
      console.error(e);
      alert("Netzwerkfehler beim Löschen der Exemplare.");
    }
  }
</script>

{#if exemplare.length === 0}
  <div class="py-16 flex flex-col items-center text-slate-400 gap-3">
    <svg class="w-10 h-10" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" /></svg>
    <p class="font-semibold text-sm">Keine physischen Exemplare mit Barcodes angelegt.</p>
  </div>
{:else}
  {#if selectedExemplare.size > 0 && appState.adminAuthenticated}
    <div class="mb-4 p-3 bg-rose-50 border border-rose-100 rounded-xl flex items-center justify-between animate-fade-in">
      <span class="text-sm font-semibold text-rose-800">{selectedExemplare.size} Exemplare ausgewählt</span>
      <button class="px-3 py-1.5 bg-rose-600 hover:bg-rose-700 text-white text-xs font-bold rounded-lg shadow-sm transition-colors cursor-pointer" onclick={deleteSelectedCopies}>
        Ausgewählte löschen
      </button>
    </div>
  {/if}
  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
    {#each exemplare as ex (ex.id)}
      <BookExemplarCard
        {ex}
        selected={selectedExemplare.has(ex.id)}
        onToggleSelect={() => toggleSelect(ex.id)}
        onDelete={() => deleteCopy(ex)}
      />
    {/each}
  </div>
{/if}
