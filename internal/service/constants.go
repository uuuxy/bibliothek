// Package service trägt die Fachlogik mit ihrer Transaktion: Ausleihe und Rückgabe, die Suche
// der Theke, das Nachbuchen, die Geräte, das Anlegen einer Bestellung mit Wareneingang, die
// Regeln des LMF-Plans, Cover, Fotos und die Importe. Die Anweisungen an die Datenbank stehen in
// repository/, die Türen in api/.
package service

// Mehrfach genutzte Domaenen-Literale (SonarQube go:S1192).
const (
	actionReturn        = "rückgabe"       // Aktionstyp Rueckgabe
	overrideBlockAction = "override-block" // Override-Kennung bei Ausleih-Blockade
	dateFormatDE        = "02.01.2006"     // TT.MM.JJJJ
)
