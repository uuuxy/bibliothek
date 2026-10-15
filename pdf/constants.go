// Package pdf erzeugt die gedruckten Dokumente: Briefe, Bescheide, Listen, Berichte und
// Etiketten. Ein Erzeuger bekommt seine Eingabe mit fertigen Texten und kennt die Datenbank
// nicht; die Tür füllt sie (docs/ARCHITEKTUR.md 5.2.2).
package pdf

// datumsformat steht einmal hier statt an jedem Erzeuger (SonarQube go:S1192).
const datumsformat = "02.01.2006" // TT.MM.JJJJ
