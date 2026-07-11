import { openDB } from "idb";

const DB_NAME = "bibliothek-offline-db";
const STORE_NAME = "offline_actions";

async function getDB() {
  return openDB(DB_NAME, 3, {
    upgrade(db, oldVersion, newVersion, transaction) {
      if (oldVersion < 2) {
        if (db.objectStoreNames.contains("scans")) {
          db.deleteObjectStore("scans");
        }
      }
      let store;
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        store = db.createObjectStore(STORE_NAME, { keyPath: "id" });
      } else {
        store = transaction.objectStore(STORE_NAME);
      }
      if (oldVersion < 3) {
        if (!store.indexNames.contains("timestamp")) {
          store.createIndex("timestamp", "timestamp");
        }
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
 * Enqueues a new action to the offline queue in IndexedDB.
 * @param {'checkout' | 'checkin'} action_type
 * @param {string} barcode_id
 * @param {string|null} schueler_id
 * @returns {Promise<void>}
 */
export async function enqueueOfflineAction(action_type, barcode_id, schueler_id = null, idempotencyKey = null) {
  try {
    const db = await getDB();
    const id = idempotencyKey || crypto.randomUUID();
    await db.add(STORE_NAME, {
      id,
      action_type,
      barcode_id,
      schueler_id,
      timestamp: Date.now()
    });
  } catch (err) {
    console.error("Failed to enqueue offline action to IndexedDB:", err);
  }
}

/**
 * Retrieves the oldest action from the queue.
 * @returns {Promise<any | null>}
 */
export async function peekOfflineAction() {
  try {
    const db = await getDB();
    const tx = db.transaction(STORE_NAME, 'readonly');
    const store = tx.objectStore(STORE_NAME);
    const index = store.index('timestamp');
    const cursor = await index.openCursor();
    return cursor ? cursor.value : null;
  } catch (err) {
    console.error("Failed to peek offline queue:", err);
    return null;
  }
}

/**
 * Deletes an action from the queue.
 * @param {string} id
 * @returns {Promise<void>}
 */
export async function dequeueOfflineAction(id) {
  try {
    const db = await getDB();
    await db.delete(STORE_NAME, id);
  } catch (err) {
    console.error(`Failed to dequeue offline action ${id}:`, err);
  }
}
