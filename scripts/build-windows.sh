#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "$script_dir/.." && pwd)"
dist_dir="$repo_root/dist"
release_name="S1LV3RWAREZ-WIN64"
release_dir="$dist_dir/$release_name"
zip_path="$dist_dir/$release_name.zip"
marker="$dist_dir/.$release_name.generated"
marker_value="S1LV3RWAREZ generated Windows x64 release v1"

if ! command -v go >/dev/null 2>&1; then
	printf 'error: Go is required to build the Windows release\n' >&2
	exit 1
fi
if ! command -v zip >/dev/null 2>&1; then
	printf 'error: zip is required to package the Windows release (install the host zip utility)\n' >&2
	exit 1
fi

mkdir -p -- "$dist_dir"
if [[ -e "$marker" ]]; then
	if [[ "$(cat -- "$marker")" != "$marker_value" ]]; then
		printf 'error: release ownership marker has unexpected contents: %s\n' "$marker" >&2
		exit 1
	fi
elif [[ -e "$release_dir" || -e "$zip_path" ]]; then
	printf 'error: release output already exists without this script\x27s ownership marker; move it aside before building: %s or %s\n' "$release_dir" "$zip_path" >&2
	exit 1
fi
if [[ -L "$release_dir" || ( -e "$release_dir" && ! -d "$release_dir" ) ]]; then
	printf 'error: release directory must be a normal directory: %s\n' "$release_dir" >&2
	exit 1
fi

tmp_dir="$(mktemp -d "$dist_dir/.${release_name}.tmp.XXXXXX")"
cleanup() { rm -rf -- "$tmp_dir"; }
trap cleanup EXIT

(
	cd -- "$repo_root"
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
		-trimpath -buildvcs=false -ldflags='-H=windowsgui' \
		-o "$tmp_dir/S1LV3RWAREZ.exe" .
)

cat >"$tmp_dir/README.NFO" <<'EOF'
S1LV3RWAREZ - THE LOST CRACKTRO
f0rkz + sp00nz
EST. 1998 / RECOVERED 2026

WINDOWS x64 RELEASE
Run S1LV3RWAREZ.exe to launch the cracktro.
Press SPACE (or click) to switch parts.

  PART I   .. THE LOST CRACKTRO
              150 BPM hard trance in F minor
  PART II  .. THE ZIP DISK STRIKES BACK
              140 BPM tracker chip tune in A minor
              InstallWiz 2.0 says the disk is full again
  PART III .. LAN PARTY
              132 BPM big-beat breakbeat in D minor
              HL / SC / CS / TFC, one 10BASE-T hub, zero sleep
  PART IV  .. WEB CLUB
              120 BPM industrial march in E minor
              6th grade, donated IBM PCs, homepages, Red Alert,
              one teacher nuke during an exec tour. No more RA.
  PART V   .. QUAKECON
              110 BPM dark industrial electro in C minor
              Annual trips to Dallas. Why Dallas? The surface
              of the sun was already booked.

This is a retrospective creative project, not an actual recovered
historical executable. Graphics and music are generated or embedded
by the project; no extra runtime files are needed.

THE SCENE NEVER DIED
EOF

cat >"$tmp_dir/FILE_ID.DIZ" <<'EOF'
S1LV3RWAREZ - THE LOST CRACKTRO
f0rkz + sp00nz | EST. 1998 / RECOVERED 2026
PART II: THE ZIP DISK STRIKES BACK
PART III: LAN PARTY
PART IV: WEB CLUB | PART V: QUAKECON
Windows x64 | Run S1LV3RWAREZ.exe
SPACE switches parts
Retrospective creative project, made in 2026.
THE SCENE NEVER DIED
EOF

(
	cd -- "$tmp_dir"
	zip -q "$tmp_dir/$release_name.zip" S1LV3RWAREZ.exe README.NFO FILE_ID.DIZ
)

mkdir -p -- "$release_dir"
printf '%s\n' "$marker_value" >"$marker.tmp"
mv -f -- "$marker.tmp" "$marker"
for name in S1LV3RWAREZ.exe README.NFO FILE_ID.DIZ; do
	install -m 0644 -- "$tmp_dir/$name" "$release_dir/$name.tmp"
	mv -f -- "$release_dir/$name.tmp" "$release_dir/$name"
done
install -m 0644 -- "$tmp_dir/$release_name.zip" "$zip_path.tmp"
mv -f -- "$zip_path.tmp" "$zip_path"

printf 'Release artifacts:\n'
for path in "$release_dir/S1LV3RWAREZ.exe" "$release_dir/README.NFO" "$release_dir/FILE_ID.DIZ" "$zip_path"; do
	printf '  %s (%s bytes)' "$path" "$(wc -c <"$path" | tr -d '[:space:]')"
	if command -v sha256sum >/dev/null 2>&1; then
		printf '  SHA-256 %s' "$(sha256sum "$path" | cut -d ' ' -f 1)"
	fi
	printf '\n'
done
