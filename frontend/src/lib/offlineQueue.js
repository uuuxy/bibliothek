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
  const remainingIds = [];

  for (const item of q) {
    try {
      const res = await apiFetch("/api/action", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          query: item.barcode,
          active_student_id: item.studentID ?? undefined,
          active_teacher_id: item.teacherID ?? undefined
        }),
        signal: AbortSignal.timeout(8000)
      });
      if (!res.ok) {
        console.warn("[OfflineQueue] Permanent error for", item.barcode, res.status);
      }
      // Successfully processed or permanent error: remove from IndexedDB
      await db.delete(STORE_NAME, item.id);
    } catch (err) {
      console.error("[OfflineQueue] Network error while flushing scan:", err);
      remainingIds.push(item.id);
    }
  }

  const remainingCount = remainingIds.length;
  if (remainingCount === 0) {
    toast(`✅ Alle Offline-Scans erfolgreich nachgesendet.`, "success");
    playSoundSuccess();
  } else {
    toast(`⚠️ ${remainingCount} Scan(s) konnten noch nicht übertragen werden.`, "warning");
  }
  return remainingCount;
}
