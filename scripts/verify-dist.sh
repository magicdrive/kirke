#!/usr/bin/env bash
# Checks that a Hephaestus dist directory has exactly the release layout kirke
# has always shipped (v1.2.8 and earlier, built by GoReleaser):
#
#   kirke_<version>_<os>_<arch>.tar.gz for the eight targets below, tar.gz on
#   Windows too, each containing exactly
#     kirke (kirke.exe on Windows)               mode 0755
#     completions/kirke-completion.sh            mode 0644
#     completions/bash/kirke-completion.bash     mode 0644
#     completions/zsh/_kirke                     mode 0644
#   (plus their parent directory entries), with the completion scripts equal to
#   misc/completions, and kirke_checksums.txt listing every archive.
#
# When an archive for the host platform exists, its binary must print
# "kirke version v<version>" (skipped with VERIFY_DIST_NO_EXEC=1, used where
# release credentials are present, so that no built code runs there).
#
# Every binary must have been built by exactly the Go toolchain pinned in
# hephaestus.yaml (builds[].env.GOTOOLCHAIN), which must be at least as new
# as go.mod's `go` line, as recorded in the binary's build information (read
# with `go version`, which does not execute the binary).
#
# Usage: scripts/verify-dist.sh <dist-dir> <version without v>
set -euo pipefail
export LC_ALL=C

if [[ $# -ne 2 ]]; then
  echo "usage: $0 <dist-dir> <version>" >&2
  exit 2
fi
dist="$1" version="$2"
repo="$(cd "$(dirname "$0")/.." && pwd)"
targets=(darwin_amd64 darwin_arm64 linux_386 linux_amd64 linux_arm64 windows_386 windows_amd64 windows_arm64)
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

fail() { echo "verify-dist: $*" >&2; exit 1; }

# version_ge A B: whether version A (N.N[.N]) is at least B.
version_ge() {
  local -a a b
  local i
  IFS=. read -ra a <<<"$1"
  IFS=. read -ra b <<<"$2"
  for i in 0 1 2; do
    if (( ${a[i]:-0} > ${b[i]:-0} )); then return 0; fi
    if (( ${a[i]:-0} < ${b[i]:-0} )); then return 1; fi
  done
  return 0
}

# The release toolchain pinned in hephaestus.yaml: one `GOTOOLCHAIN: go1.X.Y`.
pinned_go="$(sed -n 's/^[[:space:]]*GOTOOLCHAIN:[[:space:]]*"\{0,1\}go\([0-9][0-9.]*\)"\{0,1\}[[:space:]]*$/\1/p' "$repo/hephaestus.yaml")"
[[ "$pinned_go" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || fail "no single GOTOOLCHAIN: go1.X.Y in hephaestus.yaml ('$pinned_go')"
required_go="$(awk '$1 == "go" { print $2; exit }' "$repo/go.mod")"
[[ "$required_go" =~ ^[0-9]+\.[0-9]+(\.[0-9]+)?$ ]] || fail "cannot read the required Go version from go.mod ('$required_go')"
version_ge "$pinned_go" "$required_go" || fail "hephaestus.yaml pins go$pinned_go, older than go.mod's go $required_go"
command -v go >/dev/null || fail "go is needed to read the toolchain recorded in the binaries"
toolchains=""

[[ -d "$dist" ]] || fail "$dist is not a directory"

# Exactly the expected archives, and no zip.
expected_archives="$(for t in "${targets[@]}"; do echo "kirke_${version}_${t}.tar.gz"; done | sort)"
actual_archives="$(cd "$dist" && find . -maxdepth 1 -type f \( -name '*.tar.gz' -o -name '*.zip' \) | sed 's|^\./||' | sort)"
if [[ "$actual_archives" != "$expected_archives" ]]; then
  echo "expected:"; echo "$expected_archives"; echo "actual:"; echo "$actual_archives"
  fail "unexpected set of archives"
fi

# The checksum file lists exactly these archives and matches them.
sums="$dist/kirke_checksums.txt"
[[ -f "$sums" ]] || fail "missing kirke_checksums.txt"
listed="$(awk '{print $2}' "$sums" | sort)"
[[ "$listed" == "$expected_archives" ]] || fail "kirke_checksums.txt does not list exactly the archives"
grep -Eqv '^[0-9a-f]{64}  kirke_[^ ]+\.tar\.gz$' "$sums" && fail "kirke_checksums.txt has a malformed line"
if command -v sha256sum >/dev/null; then
  (cd "$dist" && sha256sum --quiet -c kirke_checksums.txt) || fail "checksum mismatch"
else
  (cd "$dist" && shasum -a 256 -c kirke_checksums.txt >/dev/null) || fail "checksum mismatch"
fi

host_os="$(uname -s | tr '[:upper:]' '[:lower:]')"
host_arch="$(uname -m)"
case "$host_arch" in x86_64) host_arch=amd64 ;; aarch64) host_arch=arm64 ;; esac

for t in "${targets[@]}"; do
  archive="$dist/kirke_${version}_${t}.tar.gz"
  exe=kirke
  [[ "$t" == windows_* ]] && exe=kirke.exe
  # mode and path of every entry, directories marked by a trailing slash.
  listing="$(tar -tvzf "$archive" | awk '{print $1, $NF}' | sort -k2)"
  expected="$(printf '%s\n' \
    "drwxr-xr-x completions/" \
    "drwxr-xr-x completions/bash/" \
    "-rw-r--r-- completions/bash/kirke-completion.bash" \
    "-rw-r--r-- completions/kirke-completion.sh" \
    "drwxr-xr-x completions/zsh/" \
    "-rw-r--r-- completions/zsh/_kirke" \
    "-rwxr-xr-x $exe" | sort -k2)"
  if [[ "$listing" != "$expected" ]]; then
    echo "expected:"; echo "$expected"; echo "actual:"; echo "$listing"
    fail "unexpected contents in $(basename "$archive")"
  fi
  dir="$work/$t"
  mkdir -p "$dir"
  tar -xzf "$archive" -C "$dir"
  for f in kirke-completion.sh bash/kirke-completion.bash zsh/_kirke; do
    cmp -s "$dir/completions/$f" "$repo/misc/completions/$f" || fail "$t: completions/$f differs from misc/completions/$f"
  done
  # The toolchain recorded in the binary (`<file>: go1.26.10`); run outside the
  # module so that go.mod cannot switch the go command itself.
  built_with="$(cd "$work" && GOTOOLCHAIN=local go version "$dir/$exe" | awk '{ print $NF }')"
  [[ "$built_with" =~ ^go([0-9]+\.[0-9]+(\.[0-9]+)?)$ ]] || fail "$t: unexpected toolchain '$built_with' in the binary"
  [[ "${BASH_REMATCH[1]}" == "$pinned_go" ]] || fail "$t: built with $built_with, hephaestus.yaml pins go$pinned_go"
  toolchains="$(printf '%s\n%s\n' "$toolchains" "$built_with" | sed '/^$/d' | sort -u)"
  if [[ -z "${VERIFY_DIST_NO_EXEC:-}" && "$t" == "${host_os}_${host_arch}" ]]; then
    out="$("$dir/$exe" --version)"
    [[ "$out" == "kirke version v$version" ]] || fail "$t: --version printed '$out', want 'kirke version v$version'"
    echo "verify-dist: $t binary: $out"
  fi
done

[[ "$(wc -l <<<"$toolchains" | tr -d ' ')" == 1 ]] || fail "binaries built by different toolchains: ${toolchains//$'\n'/ }"
echo "verify-dist: every binary built with $toolchains (pinned in hephaestus.yaml; go.mod: go $required_go)"
echo "verify-dist: ok (${#targets[@]} archives for $version, kirke_checksums.txt)"
