<script>
  import { authStore } from "./stores/authStore.svelte.js";
  import { uiStore } from "./stores/uiStore.svelte.js";
  import { appState } from "../inventur/lib/store.svelte.js";

  import Omnibox from "./Omnibox.svelte";
  import BookDetails from "./BookDetails.svelte";
  import BookAkte from "./BookAkte.svelte";
  import Graduates from "./Graduates.svelte";
  import StudentIdDesigner from "./StudentIdDesigner.svelte";
  import LabelPrinter from "./LabelPrinter.svelte";
  import BestellWorkspace from "./BestellWorkspace.svelte";
  import UnifiedInventory from "./UnifiedInventory.svelte";
  import MediaCatalog from "./MediaCatalog.svelte";
  import StatsDashboard from "./StatsDashboard.svelte";
  import AuditLog from "./AuditLog.svelte";
  import StudentDirectory from "./StudentDirectory.svelte";
  import PermissionManager from "./PermissionManager.svelte";
  import LehrerPortal from "./LehrerPortal.svelte";
  import Mahnwesen from "./Mahnwesen.svelte";
  import SystemSettings from "./SystemSettings.svelte";

  function handleSelectBook(book) {
    appState.selectedBook = book;
    uiStore.activeTab = "media_catalog";
  }

  // Routing effects
  $effect(() => {
    if (authStore.isLoggedIn && authStore.currentUser) {
      const role = authStore.currentUser.rolle ? authStore.currentUser.rolle.toLowerCase() : "";
      const path = window.location.pathname;
      const isHelfer = role === "helfer";

      if (isHelfer) {
        if (uiStore.activeTab !== "kiosk" && uiStore.activeTab !== "media_catalog") {
          uiStore.activeTab = "kiosk";
        }
        if (path !== "/" && path !== "/kiosk" && path !== "/katalog") {
          window.history.replaceState(null, "", "/kiosk");
        }
      } else {
        /** @type {Record<string, string>} */
        const tabToPath = {
          settings: "/einstellungen",
          inventory: "/inventur",
          students_dir: "/schuelerdatei",
          orders: "/bestellungen",
          media_catalog: "/katalog",
          graduates: "/abgaenger",
          stats: "/statistiken",
          mahnwesen: "/mahnwesen",
          audit: "/logbuch",
          permissions: "/berechtigungen",
          student_ids: "/ausweise",
          labels: "/etiketten",
          kiosk: "/kiosk"
        };

        if (!uiStore.isInitialRouteMatched && path !== "/") {
          if (path.startsWith("/katalog/buch/")) {
            uiStore.activeTab = "book_detail";
            appState.activeBookId = path.replace("/katalog/buch/", "");
          } else {
            const matchedTab = Object.keys(tabToPath).find(key => tabToPath[key] === path);
            if (matchedTab) uiStore.activeTab = matchedTab;
          }
          uiStore.isInitialRouteMatched = true;
        } else if (!uiStore.isInitialRouteMatched) {
          uiStore.isInitialRouteMatched = true;
        }

        let targetPath = tabToPath[uiStore.activeTab];
        if (uiStore.activeTab === "book_detail" && appState.activeBookId) {
          targetPath = `/katalog/buch/${appState.activeBookId}`;
        }
        
        if (targetPath && path !== targetPath && uiStore.isInitialRouteMatched) {
          window.history.pushState(null, "", targetPath);
        }
      }
    }
  });

  $effect(() => {
    /** @param {KeyboardEvent} e */
    function handleGlobalKeyDown(e) {
      if (e.key === "Escape" && uiStore.activeTab !== "kiosk") {
        uiStore.activeTab = "kiosk";
      }
    }
    function handlePopState() {
      const path = window.location.pathname;
      if (path.startsWith("/katalog/buch/")) {
        uiStore.activeTab = "book_detail";
        appState.activeBookId = path.replace("/katalog/buch/", "");
      } else {
        /** @type {Record<string, string>} */
        const tabToPath = {
          settings: "/einstellungen", inventory: "/inventur",
          students_dir: "/schuelerdatei", orders: "/bestellungen",
          media_catalog: "/katalog", graduates: "/abgaenger",
          stats: "/statistiken", mahnwesen: "/mahnwesen",
          audit: "/logbuch", permissions: "/berechtigungen",
          student_ids: "/ausweise", labels: "/etiketten", kiosk: "/kiosk"
        };
        const matchedTab = Object.keys(tabToPath).find(key => tabToPath[key] === path);
        if (matchedTab) uiStore.activeTab = matchedTab;
      }
    }
    window.addEventListener("keydown", handleGlobalKeyDown);
    window.addEventListener("popstate", handlePopState);
    return () => {
      window.removeEventListener("keydown", handleGlobalKeyDown);
      window.removeEventListener("popstate", handlePopState);
    };
  });
</script>

<main class="flex-1 overflow-y-auto flex flex-col w-full">
  {#if uiStore.activeTab === "kiosk"}
    <div class="flex-1 flex items-center justify-center w-full max-w-4xl mx-auto animate-fade-in">
      <Omnibox onSelectBook={handleSelectBook} />
    </div>
  {:else if uiStore.activeTab === "books"}
    <div class="w-full animate-fade-in"><BookDetails title={uiStore.selectedBook || undefined} /></div>
  {:else if uiStore.activeTab === "graduates"}
    <div class="w-full animate-fade-in"><Graduates /></div>
  {:else if uiStore.activeTab === "orders"}
    <div class="w-full animate-fade-in"><BestellWorkspace /></div>
  {:else if uiStore.activeTab === "stats"}
    <div class="w-full animate-fade-in"><StatsDashboard /></div>
  {:else if uiStore.activeTab === "audit"}
    <div class="w-full animate-fade-in"><AuditLog /></div>
  {:else if uiStore.activeTab === "student_ids"}
    <div class="w-full animate-fade-in"><StudentIdDesigner /></div>
  {:else if uiStore.activeTab === "labels"}
    <div class="w-full animate-fade-in"><LabelPrinter /></div>
  {:else if uiStore.activeTab === "media_catalog"}
    <div class="w-full animate-fade-in"><MediaCatalog /></div>
  {:else if uiStore.activeTab === "inventory"}
    <div class="w-full animate-fade-in"><UnifiedInventory /></div>
  {:else if uiStore.activeTab === "students_dir"}
    <div class="w-full animate-fade-in"><StudentDirectory role={authStore.currentUser?.rolle} /></div>
  {:else if uiStore.activeTab === "permissions"}
    <div class="w-full animate-fade-in"><PermissionManager /></div>
  {:else if uiStore.activeTab === "mahnwesen"}
    <div class="w-full animate-fade-in"><Mahnwesen /></div>
  {:else if uiStore.activeTab === "lehrer_portal"}
    <div class="w-full animate-fade-in"><LehrerPortal user={authStore.currentUser} /></div>
  {:else if uiStore.activeTab === "settings"}
    <div class="w-full animate-fade-in"><SystemSettings /></div>
  {:else if uiStore.activeTab === "book_detail"}
    <div class="w-full animate-fade-in"><BookAkte bookId={appState.activeBookId} onBack={() => { uiStore.activeTab = 'media_catalog'; appState.activeBookId = null; }} /></div>
  {/if}
</main>
