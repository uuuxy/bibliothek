# 🎨 Flat & Edge-to-Edge Sweep — Abhakliste

> Ziel: Karten-/Kachel-Anti-Pattern auf **Layout-Blöcken** beseitigen → flach auf
> Seitenhintergrund, edge-to-edge (`max-w-6xl/7xl`/`w-full`), Trennung durch
> `border-b border-gray-200` + großzügigen Abstand, Labels `text-sm font-medium
> text-gray-600`, wichtige Werte/Felder hochskaliert (`text-lg`/`text-xl`).
> Regeln: DRY (Snippets), ≤200 Zeilen/Datei, **nur UI** (keine Logik/API/State).

## Konventionsregeln (Scope-Abgrenzung)
- **KONVERTIEREN**: Seiten-/Sektions-Karten, die Formulare/Tabellen/Infoblöcke umschließen.
- **BEWAHREN** (kein Anti-Pattern): Modals/Overlays, Toasts, Dropdowns, echte
  Cover-Galerie-Kacheln (Flachlegen einer Cover-Galerie ist eine separate, riskante
  UX-Entscheidung — vor Umbau klären).

## ✅ Erledigt (verifiziert, Build grün)
- [x] `lib/MailTemplates.svelte` — doppelte Karten (bg-white rounded-3xl shadow + bg-slate-50 rounded-2xl) entfernt; max-w-6xl; Labels text-sm/medium; Betreff text-lg, Textarea text-base; Platzhalter als Links-Akzent.
- [x] `lib/OverdueWidget.svelte` — Alert-Karte → flaches Links-Akzent-Alert; Header-Kästchen raus; Label hochskaliert.

## 🔒 Bewahren (bewusst NICHT angefasst)
Modals: `DamageReportModal`, `KioskChecklistModal`, `KioskDamageModal`, `KioskReservationModal`, `StrichcodeScannerOverlay` · Toasts/Indikatoren: `OfflineIndicator` · Cover-Galerien/Kacheln (Design-Frage offen): `BuchKarte`, `KlassenBuchKachel(Startseite)`, `KlassenUebersicht(Startseite)`, `OpacSearch`-Trefferraster.

## ✅ Bestellungen — erledigt (Batch 2)
- [x] `OrderCreationPanel` — Panel-Karte → flach, Labels uppercase→text-sm/medium, Felder bg-white, Titel text-lg, Staging-Block → Links-Akzent ohne Schatten.
- [x] `OrderRecommendations`, `IncomingShipments` — Panel-Karten → flach, Header via border-b.
- [x] `SupplierManager` — beide Karten flach, Labels + Tabellenkopf entkapitalisiert.
- [x] `BestellWorkspace` — war auf Layout-Ebene bereits flach (Tableiste/PDF-Button = Controls, bleiben).

## ⏳ Offen — Layout-Karten (nach Bereich gebündelt, für Konsistenz batchweise)
**Schüler-Listen:** `components/students/ActiveStudentList`, `DeletedStudentList`
**Buch-Akte Tabs:** `BookAkteMeta`, `BookExemplareTab`, `BookHistoryTab`, `BookVormerkungenTab`, `BookBorrowersTab`, `BookBorrowersList`
**Mahnwesen/Audit:** `components/mahnwesen/MahnwesenTable`, `AuditLog`, `AdminAuditLog`
**Dashboards/Portale:** `StatsDashboard`, `LehrerPortal`, `UnifiedInventory`, `components/kiosk/KioskIdle`, `KioskActiveSession`
**Sonstiges:** `OpacSearch` (nur Filterbar/Breite), `StudentVormerkungenCard`, `ClassPrintStation`, `MailTemplates`✓, `components/labels/LabelSettings`, `designer/PropertiesPanel`
**Inventur-Modul:** `inventur/routes/settings/+page`, `inventur/routes/scanner/+page`, `inventur/lib/components/StartseitenFilter`, `StartseitenHeaderJahrgaenge`, `StartseitenHeaderKlassen`, `admin/BookTable`, `admin/BuchFormular`, `admin/KlassenUebersicht`

## 📏 Zusätzlich: Dateien > 200 Zeilen (Split nötig)
`inventur/routes/settings/+page` (304), `Omnibox` (299), `Graduates` (297), `BookExemplareTab` (296), `inventur/routes/+page` (287), `BestellWorkspace` (275), `PermissionManager` (263), u. a. — beim jeweiligen Bereichs-Batch mit Snippets/Subkomponenten aufteilen.
