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
      const item = await peekOfflineAction();
      if (!item) break; // queue empty

      try {
        const payload = {
          query: item.barcode_id
        };
        if (item.action_type === 'checkout' && item.schueler_id) {
          payload.active_student_id = item.schueler_id;
        }

        const res = await apiClient.post("/api/action", payload);
        
        if (res.ok) {
          await dequeueOfflineAction(item.id);
          await updateCount();
          syncedAny = true;
        } else {
          // If server rejects with 4xx, it might be a business logic error.
          // For a strict offline queue, if it's a 4xx (e.g. book already checked out), 
          // we might have to dequeue it anyway to not block the queue forever, 
          // or mark it as failed. 
          // Assuming 400-499 means the server processed it but rejected it:
          if (res.status >= 400 && res.status < 500 && res.status !== 429) {
             console.warn(`Permanent error for offline action ${item.id} (Status ${res.status}). Dropping from queue.`);
             await dequeueOfflineAction(item.id);
             await updateCount();
          } else {
             // 5xx or network error, stop syncing and retry later
             break;
          }
        }
      } catch (err) {
        // network error
        break;
      }
    }

    if (syncedAny && pendingCount === 0) {
      // Play sound if we synced something and queue is now empty
      playSoundSuccess();
    }

    isSyncing = false;
  }

  function init() {
    if (typeof window !== 'undefined') {
      updateCount();
      window.addEventListener('online', startSync);
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
    init
  };
}

export const offlineSync = createOfflineSyncStore();
