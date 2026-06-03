import { apiFetch } from "./apiFetch.js";
import { playSoundSuccess, playSoundError } from "./audio.js";

const QUEUE_KEY = "bibliothek_offline_queue";

export function loadQueue() {
  try {
    return JSON.parse(localStorage.getItem(QUEUE_KEY) || "[]");
  } catch { return []; }
}

export function saveQueue(/** @type {any[]} */ q) {
  localStorage.setItem(QUEUE_KEY, JSON.stringify(q));
  return q.length;
}

export function enqueueOfflineScan(/** @type {string} */ barcode, /** @type {string|null} */ studentID, /** @type {string|null} */ teacherID) {
  const q = loadQueue();
  const alreadyQueued = q.some(
    (/** @type {any} */ item) => item.barcode === barcode && item.studentID === studentID && item.teacherID === teacherID
  );
  if (!alreadyQueued) {
    q.push({ barcode, studentID, teacherID, queuedAt: Date.now() });
    return saveQueue(q);
  }
  return q.length;
}

export async function flushOfflineQueue(/** @type {function(string, string): void} */ showToast) {
  const q = loadQueue();
  if (q.length === 0) return 0;

  showToast(`📡 Verbindung wiederhergestellt – ${q.length} Offline-Scan(s) werden nachgesendet…`, "success");

  const remaining = [];
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
    } catch {
      remaining.push(item);
    }
  }

  saveQueue(remaining);
  if (remaining.length === 0) {
    showToast(`✅ Alle Offline-Scans erfolgreich nachgesendet.`, "success");
    playSoundSuccess();
  } else {
    showToast(`⚠️ ${remaining.length} Scan(s) konnten noch nicht übertragen werden.`, "warning");
  }
  return remaining.length;
}
