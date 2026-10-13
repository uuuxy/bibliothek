// Package pdf erzeugt die gedruckten Dokumente: Briefe, Bescheide, Listen, Berichte und
// Etiketten. Ein Erzeuger bekommt seine Eingabe mit fertigen Texten und kennt die Datenbank
// nicht; die Tür füllt sie (docs/ARCHITEKTUR.md 5.2.2).
package pdf

// Datumsformat (SonarQube go:S1192).
const dateFormatDE = "02.01.2006" // TT.MM.JJJJ
