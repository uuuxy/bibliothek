import { loadQueue, peekOfflineAction, dequeueOfflineAction } from "../offlineQueue.js";
import { apiClient } from "../apiFetch.js";
import { playSoundSuccess } from "../audio.js";

function createOfflineSyncStore() {
  let pendingCount = $state(0);
  let isSyncing = $state(false);

  async function updateCount() {
    const q = await loadQueue();
    pendingCount = q.length;
  }

  async function startSync() {
    if (isSyncing || !navigator.onLine) return;
    isSyncing = true;
    
    let syncedAny = false;

    while (navigator.onLine) {
      const q = await loadQueue();
      if (q.length === 0) break;

      // Ensure they are processed in order of creation
      q.sort((a, b) => a.timestamp - b.timestamp);
      
      const batchItems = q.slice(0, 50);
      const payload = batchItems.map(item => {
        const req = { query: item.barcode_id };
        if (item.action_type === 'checkout' && item.schueler_id) {
          req.active_student_id = item.schueler_id;
        }
        return req;
      });

      try {
        const res = await apiClient.post("/api/action/batch", payload);
        
        if (res.ok) {
          const data = await res.json();
          for (let i = 0; i < batchItems.length; i++) {
            const item = batchItems[i];
            const result = data.results?.find(r => r.index === i);
            
            // Dequeue if successful, or if server rejected with a permanent client error (4xx) except Too Many Requests (429)
            if (result && (result.success || (result.status >= 400 && result.status < 500 && result.status !== 429))) {
              await dequeueOfflineAction(item.id);
            } else if (!result) {
              // Failsafe: if backend doesn't return an index for some reason but 200 OK overall
              await dequeueOfflineAction(item.id);
            }
          }
          await updateCount();
          syncedAny = true;
          
          // Network Jitter: wait 200-500ms before sending the next batch
          if (q.length > 50) {
            const jitter = 200 + Math.random() * 300;
            await new Promise(resolve => setTimeout(resolve, jitter));
          }
        } else {
          // Batch request failed completely (e.g. 502 Bad Gateway), stop syncing
          break;
        }
      } catch (err) {
        // Network error
        break;
      }
    }

    if (syncedAny && pendingCount === 0) {
      playSoundSuccess();
    }
    isSyncing = false;
  }

  async function exportQueueAsJSON() {
    const q = await loadQueue();
    if (q.length === 0) return;
    const blob = new Blob([JSON.stringify(q, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `offline_scans_backup_${new Date().toISOString().slice(0,10)}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }

  async function importQueueFromJSON(file) {
    try {
      const text = await file.text();
      const items = JSON.parse(text);
      if (!Array.isArray(items)) throw new Error("Invalid format");
      
      const { enqueueOfflineAction } = await import("../offlineQueue.js");
      let importedCount = 0;
      for (const item of items) {
        if (!item.action_type || !item.barcode_id) continue;
        await enqueueOfflineAction(item.action_type, item.barcode_id, item.schueler_id || null);
        importedCount++;
      }
      
      await updateCount();
      startSync();
      return importedCount;
    } catch (e) {
      console.error(e);
      throw new Error("Fehler beim Einlesen der Backup-Datei.");
    }
  }

  function handleBeforeUnload(e) {
    if (pendingCount > 0) {
      e.preventDefault();
      const msg = "Es gibt noch ungespeicherte Daten (Offline-Queue). Datenverlust droht! Bitte Browser nicht schließen.";
      e.returnValue = msg;
      return msg;
    }
  }

  function init() {
    if (typeof window !== 'undefined') {
      updateCount();
      window.addEventListener('online', startSync);
      window.addEventListener('beforeunload', handleBeforeUnload);
      // Periodic check every 30s just in case online event missed or transient 5xx errors
      setInterval(() => {
        if (pendingCount > 0) startSync();
      }, 30000);
    }
  }

  return {
    get pendingCount() { return pendingCount; },
    get isSyncing() { return isSyncing; },
    updateCount,
    startSync,
    exportQueueAsJSON,
    importQueueFromJSON,
    init
  };
}

export const offlineSync = createOfflineSyncStore();
