package videos

import (
	"path/filepath"
	"strconv"
)

// Storage-Layout (siehe design.md):
//
//	{root}/
//	  uploads/            tus-Sessions (Chunks während Upload)
//	  raw/{id}.mp4        fertiger Upload, wird nach Transcode gelöscht
//	  processed/{id}/
//	    master.m3u8       multi-variant Manifest
//	    {rendition}/      z.B. 720p/, 360p/ mit index.m3u8 + Segmenten
//
// Alle Helper sind reine Funktionen und nehmen das Wurzelverzeichnis explizit
// entgegen, damit sie ohne Handler/Config testbar bleiben.

// RawPath liefert den Pfad der hochgeladenen Originaldatei eines Videos.
func RawPath(root string, id int) string {
	return filepath.Join(root, "raw", strconv.Itoa(id)+".mp4")
}

// ProcessedDir liefert das Verzeichnis mit den transcodierten HLS-Artefakten.
func ProcessedDir(root string, id int) string {
	return filepath.Join(root, "processed", strconv.Itoa(id))
}

// MasterManifestPath liefert den Pfad der Master-Playlist (master.m3u8).
func MasterManifestPath(root string, id int) string {
	return filepath.Join(ProcessedDir(root, id), "master.m3u8")
}

// RenditionDir liefert das Verzeichnis einer einzelnen Rendition (z.B. "720p").
func RenditionDir(root string, id int, rendition string) string {
	return filepath.Join(ProcessedDir(root, id), rendition)
}
