// Download-Links für das Offline-Encoding-Tool (tools/video-encoder). Videos
// lassen sich nur noch über dieses Tool hochladen: es encodiert auf dem Rechner
// der Filmenden und nutzt dann den bestehenden tus-Upload.
//
// Der Release-Workflow hängt die Binaries an JEDES GitHub-Release; `latest/download`
// löst immer auf das neueste auf — ein neues Release braucht hier keine Änderung.
// Die Asset-Namen MÜSSEN mit dem Job `video-encoder` in
// .github/workflows/release.yml übereinstimmen.
const RELEASE_BASE = 'https://github.com/UlrichFrank/TeamWERK/releases/latest/download'

export const VIDEO_ENCODER_DOWNLOADS = [
  {
    platform: 'windows',
    label: 'Tool für Windows herunterladen',
    url: `${RELEASE_BASE}/teamwerk-video-encoder-windows-amd64.exe`,
  },
  {
    platform: 'macos',
    label: 'Tool für macOS herunterladen',
    url: `${RELEASE_BASE}/teamwerk-video-encoder-macos.dmg`,
  },
] as const
