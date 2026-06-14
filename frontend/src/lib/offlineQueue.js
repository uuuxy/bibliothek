import { openDB } from "idb";
import { apiFetch } from "./apiFetch.js";
import { playSoundSuccess, playSoundError } from "./audio.js";

const DB_NAME = "bibliothek-offline-db";
const STORE_NAME = "scans";

async function getDB() {
  return openDB(DB_NAME, 1, {
    upgrade(db) {
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        db.createObjectStore(STORE_NAME, { keyPath: "id", autoIncrement: true });
      }
    }
  });
}

/**
 * Loads the current offline queue from IndexedDB.
 * @returns {Promise<any[]>}
 */
export async function loadQueue() {
  try {
    const db = await getDB();
    return await db.getAll(STORE_NAME);
  } catch (err) {
    console.error("Failed to load offline queue from IndexedDB:", err);
    return [];
  }
}

/**
 * Enqueues a new scan to the offline queue in IndexedDB.
 * @param {string} barcode
 * @param {string|null} studentID
 * @param {string|null} teacherID
 * @returns {Promise<number>} Number of items in the queue
 */
export async function enqueueOfflineScan(barcode, studentID, teacherID) {
  try {
    const db = await getDB();
    const q = await db.getAll(STORE_NAME);
    const alreadyQueued = q.some(
      (/** @type {any} */ item) => item.barcode === barcode && item.studentID === studentID && item.teacherID === teacherID
    );
    if (!alreadyQueued) {
      await db.add(STORE_NAME, { barcode, studentID, teacherID, queuedAt: Date.now() });
    }
    const updated = await db.getAll(STORE_NAME);
    return updated.length;
  } catch (err) {
    console.error("Failed to enqueue offline scan to IndexedDB:", err);
    return 0;
  }
}

/**
 * Flushes the offline queue by sending all scans to the server.
 * @param {function(string, string): void} [showToast]
 * @returns {Promise<number>} Number of items remaining in the queue
 */
export async function flushOfflineQueue(showToast) {
  const toast = typeof showToast === "function" ? showToast : () => {};
  const q = await loadQueue();
  if (q.length === 0) return 0;

  toast(`📡 Verbindung wiederhergestellt – ${q.length} Offline-Scan(s) werden nachgesendet…`, "success");

  const db = await getDB();

  // Prepare payload for the batch API
  const batchPayload = q.map(item => ({
    query: item.barcode,
    active_student_id: item.studentID ?? undefined,
    active_teacher_id: item.teacherID ?? undefined
  }));

  try {
    const res = await apiFetch("/api/action/batch", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(batchPayload),
      signal: AbortSignal.timeout(15000) // Slightly longer timeout for batch
    });

    if (!res.ok) {
      console.error("[OfflineQueue] Batch API returned error status:", res.status);
      toast(`⚠️ Fehler beim Übertragen der Offline-Scans (Status ${res.status}).`, "error");
      return q.length;
    }

    const { results } = await res.json();
    let remainingCount = 0;

    // Process the batch results
    for (let i = 0; i < q.length; i++) {
      const item = q[i];
      const result = results.find((/** @type {any} */ r) => r.index === i);

      if (result) {
        if (!result.success) {
          console.warn("[OfflineQueue] Permanent error for", item.barcode, result.status, result.error);
        }
        // Either successful or permanent business error: remove from queue
        await db.delete(STORE_NAME, item.id);
      } else {
        // Missing in response, keep in queue
        remainingCount++;
      }
    }

    if (remainingCount === 0) {
      toast(`✅ Alle Offline-Scans erfolgreich nachgesendet.`, "success");
      playSoundSuccess();
    } else {
      toast(`⚠️ ${remainingCount} Scan(s) konnten noch nicht übertragen werden.`, "warning");
    }
    return remainingCount;

  } catch (err) {
    console.error("[OfflineQueue] Network error while flushing batch:", err);
    toast(`⚠️ Netzwerkfehler beim Nachsenden der Scans.`, "warning");
    return q.length;
  }
}
