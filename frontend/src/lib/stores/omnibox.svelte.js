// stores/omnibox.svelte.js
// Status- und Logikverwaltung für die Omnibox (Svelte 5 Runes)

import { apiFetch, apiClient } from "../apiFetch.js";
import { playSoundSuccess, playSoundError } from "../audio.js";
import { enqueueOfflineAction } from "../offlineQueue.js";
import { offlineSync } from "./offlineSync.svelte.js";

export function createOmniboxStore() {
  let activeStudent = $state(/** @type {any} */ (null));
  let activeTeacher = $state(/** @type {any} */ (null));
  let queryVal = $state("");

  let toast = $state(/** @type {any} */ (null));
  let flashBorder = $state("");
  let screenFlash = $state(""); // "success" | "error" | ""
  let lastFremdrueckgabe = $state(/** @type {any} */ (null));
  let isShaking = $state(false);
  let scanError = $state(false);
  let errorMessage = $state("");
  let vormerkungAlert = $state(/** @type {{titel?: string, user?: string} | null} */ (null));
  let blockAlert = $state(/** @type {{message: string, query: string} | null} */ (null));
  // isOffline is now handled globally via offlineSync
  let offlineQueueCount = $state(0);

  // Kamerascanner-Status
  let showCamera = $state(false);
  let cameraScanner = $state(/** @type {any} */ (null));

  // Such-Status
  let debounceTimer = $state(/** @type {any} */ (null));
  let isDropdownOpen = $state(false);
  let unifiedSearchResults = $state({ students: [], books: [] });
  let selectedDropdownIndex = $state(-1);
  let totalDropdownItems = $derived(
    unifiedSearchResults.students.length + unifiedSearchResults.books.length,
  );

  let isActive = $derived(!!(activeStudent || activeTeacher || isDropdownOpen));

  // UI Feedback-Methoden
  function triggerScreenFlash(type) {
    screenFlash = type;
    setTimeout(() => {
      screenFlash = "";
    }, 300);
  }

  function triggerShake() {
    isShaking = true;
    setTimeout(() => {
      isShaking = false;
    }, 500);
  }

  function triggerFlash(color) {
    flashBorder = color;
    setTimeout(() => {
      flashBorder = "";
    }, 1000);
  }

  function showToast(message, type = "success") {
    toast = { message, type };
  }

  // Such-Logik
  function handleInput() {
    clearTimeout(debounceTimer);
    if (!queryVal.trim()) {
      isDropdownOpen = false;
      unifiedSearchResults = { students: [], books: [] };
      return;
    }
    debounceTimer = setTimeout(async () => {
      if (!queryVal.trim()) return;
      try {
        const res = await apiFetch(
          `/api/search?q=${encodeURIComponent(queryVal.trim())}`,
        );
        if (res.ok) {
          const results = await res.json();
          unifiedSearchResults = {
            students: results.students || [],
            books: results.books || [],
          };
          isDropdownOpen =
            unifiedSearchResults.students.length > 0 ||
            unifiedSearchResults.books.length > 0;
          selectedDropdownIndex = -1;
        }
      } catch (err) {
        console.error("Suche fehlgeschlagen:", err);
      }
    }, 300);
  }

  // Dropdown-Auswahl
  function selectDropdownItem(index, onSelectBook) {
    const { students, books } = unifiedSearchResults;
    if (index < students.length) {
      const student = students[index];
      queryVal = student.barcode_id;
      isDropdownOpen = false;
      submitAction(null, null); // Ohne Event
    } else {
      const book = books[index - students.length];
      queryVal = "";
      isDropdownOpen = false;
      if (onSelectBook) onSelectBook(book);
    }
  }

  // Haupt-Scan-Aktion
  async function submitAction(e, reloadProfileCb, overrideBlock = false) {
    if (e) e.preventDefault();
    if (isDropdownOpen && selectedDropdownIndex >= 0) {
      selectDropdownItem(selectedDropdownIndex, null);
      return;
    }

    const q = queryVal.trim();
    if (!q) return;

    queryVal = "";
    isDropdownOpen = false;
    lastFremdrueckgabe = null;
    errorMessage = "";

    // Disable input while processing
    document.getElementById("omnibox-input")?.blur();

    const idempotencyKey = crypto.randomUUID();

    try {
      const res = await apiClient.post("/api/action", {
        query: q,
        active_student_id: activeStudent?.id,
        active_teacher_id: activeTeacher?.id,
        confirmed_checklist: false,
        override_block: overrideBlock,
        idempotency_key: idempotencyKey
      });

      if (!res.ok) {
        let errStr = await res.text();
        try {
          const errData = JSON.parse(errStr);
          if (errData.error) errStr = errData.error;
        } catch (e) {}
        
        if (res.status === 403 && (errStr.includes("Sperre") || errStr.includes("Sperr-Automatik") || errStr.includes("überfällig"))) {
          blockAlert = { message: errStr, query: q };
          throw new Error("BLOCK_ALERT");
        }

        throw new Error(errStr || "Aktion fehlgeschlagen");
      }
      const data = await res.json();

      if (data.type === "student") {
        activeStudent = data.student;
        activeTeacher = null;
        triggerScreenFlash("success");
        playSoundSuccess();
        triggerFlash("green");
      } else if (data.type === "teacher") {
        activeTeacher = data.teacher;
        activeStudent = null;
        triggerScreenFlash("success");
        playSoundSuccess();
        triggerFlash("green");
        showToast(
          `📋 Handapparat-Sitzung gestartet für Lehrer/in ${data.teacher.vorname} ${data.teacher.nachname}`,
        );
      } else if (data.type === "ausleihe") {
        if (data.fremdrueckgabe) {
          triggerScreenFlash("warning");
          playSoundError();
          triggerFlash("orange");
          const prevName = data.vorbesitzer
            ? `${data.vorbesitzer.vorname} ${data.vorbesitzer.nachname}`
            : `${data.vorbesitzer_user.vorname} ${data.vorbesitzer_user.nachname}`;
          lastFremdrueckgabe = { vorbesitzerName: prevName };
          showToast(
            `⚠️ Fremdrückgabe erfolgt (Vorbesitzer: ${prevName})`,
            "warning",
          );
        } else {
          triggerScreenFlash("success");
          playSoundSuccess();
          triggerFlash("green");
          showToast(
            `📖 „${data.book.titel}" ausgeliehen an ${activeTeacher ? activeTeacher.vorname : activeStudent?.vorname}.`,
          );
        }
        if (reloadProfileCb) reloadProfileCb();
      } else if (data.type === "rueckgabe") {
        triggerScreenFlash("success");
        playSoundSuccess();
        triggerFlash("green");

        showToast(`📥 „${data.book.titel}" erfolgreich zurückgegeben.`);
        if (data.has_vormerkung) {
          vormerkungAlert = {
            titel: data.vormerkung_titel || data.book?.titel,
            user: data.vormerkung_user
          };
        }
        if (reloadProfileCb) reloadProfileCb();

        if (data.student && !activeStudent && !activeTeacher) {
          activeStudent = data.student;
        } else if (data.teacher && !activeStudent && !activeTeacher) {
          activeTeacher = data.teacher;
        }
      } else if (data.type === "info") {
        triggerScreenFlash("success");
        playSoundSuccess();
        triggerFlash("green");
        showToast(data.message, "success");
        if (reloadProfileCb) reloadProfileCb();
      } else if (data.type === "search_results") {
        triggerShake();
        showToast("Bitte wähle ein Ergebnis aus der Liste.", "warning");
      }
    } catch (e) {
      if (e instanceof Error && e.message === "BLOCK_ALERT") {
        triggerScreenFlash("error");
        playSoundError();
        return;
      }
      if (e instanceof TypeError || !window.navigator.onLine || offlineSync.isOffline || (e instanceof Error && e.message.includes("Timeout"))) {
         if (q.startsWith("B-")) {
            await enqueueOfflineAction("checkin", q, activeStudent?.id ?? "", idempotencyKey);
            offlineSync.updateCount();
            triggerScreenFlash("warning");
            playSoundSuccess();
            showToast(`📴 Offline: Aktion für „${q}“ gespeichert.`, "warning");
         } else {
            showToast("⚠️ Netzwerkfehler", "error");
         }
      } else {
        errorMessage = String(e);
        showToast(`⚠️ Fehler: ${e instanceof Error ? e.message : String(e)}`, "error");
      }
    } finally {
      triggerFlash("red");
    }
  }

  return {
    get activeStudent() {
      return activeStudent;
    },
    set activeStudent(v) {
      activeStudent = v;
    },
    get activeTeacher() {
      return activeTeacher;
    },
    set activeTeacher(v) {
      activeTeacher = v;
    },
    get queryVal() {
      return queryVal;
    },
    set queryVal(v) {
      queryVal = v;
    },
    get toast() {
      return toast;
    },
    set toast(v) {
      toast = v;
    },
    get flashBorder() {
      return flashBorder;
    },
    set flashBorder(v) {
      flashBorder = v;
    },
    get screenFlash() {
      return screenFlash;
    },
    set screenFlash(v) {
      screenFlash = v;
    },
    get lastFremdrueckgabe() {
      return lastFremdrueckgabe;
    },
    set lastFremdrueckgabe(v) {
      lastFremdrueckgabe = v;
    },
    get isShaking() {
      return isShaking;
    },
    set isShaking(v) {
      isShaking = v;
    },
    get scanError() {
      return scanError;
    },
    set scanError(v) {
      scanError = v;
    },
    get errorMessage() {
      return errorMessage;
    },
    set errorMessage(v) {
      errorMessage = v;
    },
    get vormerkungAlert() {
      return vormerkungAlert;
    },
    set vormerkungAlert(v) {
      vormerkungAlert = v;
    },
    get blockAlert() {
      return blockAlert;
    },
    set blockAlert(v) {
      blockAlert = v;
    },
    // isOffline handled globally
    get offlineQueueCount() {
      return offlineQueueCount;
    },
    set offlineQueueCount(v) {
      offlineQueueCount = v;
    },
    get showCamera() {
      return showCamera;
    },
    set showCamera(v) {
      showCamera = v;
    },
    get cameraScanner() {
      return cameraScanner;
    },
    set cameraScanner(v) {
      cameraScanner = v;
    },
    get debounceTimer() {
      return debounceTimer;
    },
    set debounceTimer(v) {
      debounceTimer = v;
    },
    get isDropdownOpen() {
      return isDropdownOpen;
    },
    set isDropdownOpen(v) {
      isDropdownOpen = v;
    },
    get unifiedSearchResults() {
      return unifiedSearchResults;
    },
    set unifiedSearchResults(v) {
      unifiedSearchResults = v;
    },
    get selectedDropdownIndex() {
      return selectedDropdownIndex;
    },
    set selectedDropdownIndex(v) {
      selectedDropdownIndex = v;
    },
    get totalDropdownItems() {
      return totalDropdownItems;
    },
    get isActive() {
      return isActive;
    },

    // Exportierte Methoden
    triggerScreenFlash,
    triggerShake,
    triggerFlash,
    showToast,
    handleInput,
    selectDropdownItem,
    submitAction,
  };
}

export const omniboxStore = createOmniboxStore();
