#!/bin/sh
# Dinky installer.
#
# By default installs the latest published release. Pass a PR number as the
#
# UX inspired by getmic.ro (https://getmic.ro) by Ben Weissmann.
# first argument to install the build produced by that pull request instead:
#
#   curl -fsSL https://raw.githubusercontent.com/TechnoSparks/dinky/main/install.sh | sh
#   curl -fsSL .../install.sh | sh -s -- 42
#
set -eu

REPO=${DINKY_REPO:-TechnoSparks/dinky}
API=https://api.github.com
WORKFLOW=build.yaml
INSTALL_DIR=
RUN_ID=
PR_NUMBER=
YES=0
DRY_RUN=0

info() { printf '%s\n' "$*"; }
die()  { printf 'error: %s\n' "$*" >&2; exit 1; }

usage() {
  cat <<EOF
Usage: sh install.sh [PR_NUMBER] [options]

Install the latest Dinky release, or the CI build for a given pull request.

Arguments:
  PR_NUMBER           Optional PR number; installs that PR's build instead of
                      the latest release.

Options:
  -d, --install-dir DIR   Target directory (default: current directory)
  -y, --yes               Skip the confirmation prompt
      --run-id ID         Install from a specific CI run
      --repo OWNER/REPO   Repository to install from (default: $REPO)
      --dry-run           Print the resolved download URL and stop
  -h, --help              Show this help

Environment:
  DINKY_REPO, DINKY_INSTALL_DIR, DINKY_PLATFORM (e.g. linux-amd64),
  GITHUB_TOKEN (see README)

Examples:
  curl -fsSL https://raw.githubusercontent.com/TechnoSparks/dinky/main/install.sh | sh
  curl -fsSL .../install.sh | sh -s -- 42
  sh install.sh --install-dir "\$HOME/.local/bin" -y
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    -d|--install-dir)
      [ $# -ge 2 ] || die "--install-dir needs a value"
      INSTALL_DIR=$2; shift 2 ;;
    --install-dir=*) INSTALL_DIR=${1#*=}; shift ;;
    -y|--yes) YES=1; shift ;;
    --run-id)
      [ $# -ge 2 ] || die "--run-id needs a value"
      RUN_ID=$2; shift 2 ;;
    --run-id=*) RUN_ID=${1#*=}; shift ;;
    --repo)
      [ $# -ge 2 ] || die "--repo needs a value"
      REPO=$2; shift 2 ;;
    --repo=*) REPO=${1#*=}; shift ;;
    --dry-run) DRY_RUN=1; shift ;;
    -h|--help) usage; exit 0 ;;
    --) shift; break ;;
    -* ) die "unknown option: $1 (try --help)" ;;
    *)
      if [ -z "$PR_NUMBER" ] && printf '%s' "$1" | grep -qE '^[0-9]+$'; then
        PR_NUMBER=$1; shift
      else
        die "unexpected argument: $1"
      fi ;;
  esac
done

command -v curl >/dev/null 2>&1 || die "curl is required"

WORKDIR=$(mktemp -d "${TMPDIR:-/tmp}/dinky-install.XXXXXX") || die "cannot create a temp directory"
trap 'rm -rf "$WORKDIR"' EXIT HUP INT TERM

if command -v jq >/dev/null 2>&1; then JSON_PARSER=jq
elif command -v python3 >/dev/null 2>&1; then JSON_PARSER=python3
else die "jq or python3 is required to parse GitHub API responses"; fi

json_query() {
  if [ "$JSON_PARSER" = jq ]; then jq -r "$1"
  else PY_EXPR=$2 python3 -c 'import json, os, sys
d = json.load(sys.stdin)
print(eval(os.environ["PY_EXPR"]))'; fi
}

api_get() {
  if [ -n "${GITHUB_TOKEN:-}" ]; then
    curl -fsSL -H 'Accept: application/vnd.github+json' -H 'X-GitHub-Api-Version: 2022-11-28' \
      -H "Authorization: Bearer $GITHUB_TOKEN" "$API$1"
  else
    curl -fsSL -H 'Accept: application/vnd.github+json' -H 'X-GitHub-Api-Version: 2022-11-28' "$API$1"
  fi
}

# --- platform detection -----------------------------------------------------
if [ -n "${DINKY_PLATFORM:-}" ]; then
  OS=${DINKY_PLATFORM%%-*}
  ARCH=${DINKY_PLATFORM#*-}
else
  OS=$(uname -s); ARCH=$(uname -m)
  case "$OS" in Linux) OS=linux ;; Darwin) OS=darwin ;; *) die "unsupported OS: $OS (override with DINKY_PLATFORM, e.g. DINKY_PLATFORM=linux-amd64)" ;; esac
  case "$ARCH" in
    x86_64|amd64) ARCH=amd64 ;;
    aarch64|arm64) ARCH=arm64 ;;
    armv7*|armv6*|armhf) ARCH=arm ;;
    *) die "unsupported architecture: $ARCH (override with DINKY_PLATFORM, e.g. DINKY_PLATFORM=linux-amd64)" ;;
  esac
fi
ARTIFACT="dinky-$OS-$ARCH"
TARBALL="dinky_${OS}_${ARCH}.tar.gz"

SOURCE=ci
RUN_ID_FOUND=
ART_ID=
DIGEST=
DOWNLOAD_URL=
RELEASE_TAG=
PR_RUN_URL=

# --- Resolve source --------------------------------------------------------
if [ -n "$PR_NUMBER" ] || [ -n "$RUN_ID" ]; then
  SOURCE=ci
else
  # Try latest published release first.
  if api_get "/repos/$REPO/releases/latest" >"$WORKDIR/release.json" 2>/dev/null; then
    asset_url=$(json_query \
      "[.assets[] | select(.name == \"$TARBALL\") | .browser_download_url][0] // empty" \
      "next((a['browser_download_url'] for a in d.get('assets', []) if a['name'] == '$TARBALL'), '')" \
      <"$WORKDIR/release.json")
    if [ -n "$asset_url" ]; then
      SOURCE=release
      DOWNLOAD_URL=$asset_url
      RELEASE_TAG=$(json_query '.tag_name // empty' 'd.get("tag_name") or ""' <"$WORKDIR/release.json")
    fi
  fi
fi

if [ "$SOURCE" = ci ]; then
  if [ -n "$RUN_ID" ]; then
    candidate_ids=$RUN_ID
  elif [ -n "$PR_NUMBER" ]; then
    api_get "/repos/$REPO/pulls/$PR_NUMBER" >"$WORKDIR/pr.json" || die "PR #$PR_NUMBER not found"
    pr_branch=$(json_query '.head.ref // empty' 'd.get("head", {}).get("ref") or ""' <"$WORKDIR/pr.json")
    [ -n "$pr_branch" ] || die "could not resolve branch for PR #$PR_NUMBER"
    # Keep branch name out of interpolated code: pass via env to Python.
    PR_BRANCH=$pr_branch api_get "/repos/$REPO/actions/workflows/$WORKFLOW/runs?status=completed&per_page=50" >"$WORKDIR/runs.json"
    candidate_ids=$(PR_BRANCH=$pr_branch json_query \
      '[.workflow_runs[] | select(.conclusion == "success" and .head_branch == env.PR_BRANCH) | .id][0:5] | .[]' \
      '"\n".join(str(r[\"id\"]) for r in d.get(\"workflow_runs\", []) if r.get(\"conclusion\") == \"success\" and r.get(\"head_branch\") == __import__(\"os\").environ[\"PR_BRANCH\"])' \
      <"$WORKDIR/runs.json")
    [ -n "$candidate_ids" ] || die "no successful build found for PR #$PR_NUMBER (branch $pr_branch)"
  else
    # No release available: fall back to latest successful build.
    api_get "/repos/$REPO/actions/workflows/$WORKFLOW/runs?status=completed&per_page=30" >"$WORKDIR/runs.json"
    candidate_ids=$(json_query \
      '[.workflow_runs[] | select(.conclusion == "success") | .id][0:10] | .[]' \
      '"\n".join(str(r["id"]) for r in d.get("workflow_runs", []) if r.get("conclusion") == "success")' \
      <"$WORKDIR/runs.json")
  fi

  for rid in $candidate_ids; do
    if ! api_get "/repos/$REPO/actions/runs/$rid/artifacts?name=$ARTIFACT&per_page=100" >"$WORKDIR/artifacts.json" 2>/dev/null; then
      continue
    fi
    found=$(json_query \
      '[.artifacts[] | select(.expired == false) | ((.id | tostring) + " " + (.digest // ""))][0] // ""' \
      '" ".join(map(str, next(([a["id"], a.get("digest") or ""] for a in d.get("artifacts", []) if not a.get("expired")), ["", ""])))' \
      <"$WORKDIR/artifacts.json")
    ART_ID=${found%% *}
    [ -n "$ART_ID" ] || continue
    if [ "$found" = "$ART_ID" ]; then DIGEST=; else DIGEST=${found#* }; fi
    RUN_ID_FOUND=$rid
    break
  done

  [ -n "$RUN_ID_FOUND" ] || die "no successful $WORKFLOW build with artifact '$ARTIFACT' found"
  RUN_ID=$RUN_ID_FOUND
  if [ -n "${GITHUB_TOKEN:-}" ]; then
    DOWNLOAD_URL="$API/repos/$REPO/actions/artifacts/$ART_ID/zip"
  else
    DOWNLOAD_URL="https://nightly.link/$REPO/actions/runs/$RUN_ID/$ARTIFACT.zip"
  fi
fi

# --- dry run ---------------------------------------------------------------
if [ "$DRY_RUN" -eq 1 ]; then
  if [ "$SOURCE" = release ]; then info "source: release $RELEASE_TAG"
  else info "source: CI run $RUN_ID (artifact $ART_ID)"; info "sha256: $DIGEST"; fi
  info "url: $DOWNLOAD_URL"
  exit 0
fi

# --- confirm ---------------------------------------------------------------
if [ -z "$INSTALL_DIR" ]; then INSTALL_DIR=.; fi
# Expand ~
case "$INSTALL_DIR" in \~) INSTALL_DIR=$HOME ;; \~/*) INSTALL_DIR=$HOME/${INSTALL_DIR#~/} ;; esac
[ -n "$INSTALL_DIR" ] || die "install dir is empty"

if [ "$SOURCE" = release ]; then
  info "Dinky $RELEASE_TAG ($OS/$ARCH) will be installed to $INSTALL_DIR/dinky"
else
  info "Dinky CI build ($OS/$ARCH) from run $RUN_ID will be installed to $INSTALL_DIR/dinky"
  [ -n "$DIGEST" ] && info "artifact sha256: $DIGEST"
fi
if [ ! -w "$INSTALL_DIR" ] 2>/dev/null || [ ! -d "$INSTALL_DIR" ]; then
  info "Note: $INSTALL_DIR is not writable; sudo will be used."
fi

if [ "$YES" -ne 1 ]; then
  if [ -r /dev/tty ]; then
    printf 'Proceed with installation? [y/N] ' > /dev/tty
    read -r ans < /dev/tty || ans=
    case "$ans" in y|Y|yes|YES) ;; *) info "Aborted."; exit 0 ;; esac
  else
    die "no TTY for confirmation; re-run with -y/--yes"
  fi
fi

# --- download + unpack -----------------------------------------------------
info "Downloading..."
if [ "$SOURCE" = release ]; then
  curl -fsSL "$DOWNLOAD_URL" -o "$WORKDIR/$TARBALL" || die "download failed: $DOWNLOAD_URL"
  tar -xzf "$WORKDIR/$TARBALL" -C "$WORKDIR" || die "could not unpack $TARBALL"
  BIN_SRC=$(find "$WORKDIR" -type f -name dinky | head -n 1)
  [ -n "$BIN_SRC" ] || die "binary not found in release archive"
else
  ZIP="$WORKDIR/$ARTIFACT.zip"
  if [ -n "${GITHUB_TOKEN:-}" ]; then
    # Resolve the redirect without leaking the token to the (possibly foreign) host.
    curl -fsS -D "$WORKDIR/hdr" -o /dev/null --max-redirs 0 \
      -H 'Accept: application/vnd.github+json' -H 'X-GitHub-Api-Version: 2022-11-28' \
      -H "Authorization: Bearer $GITHUB_TOKEN" "$DOWNLOAD_URL" >/dev/null 2>&1 || true
    loc=$(sed -n 's/^[Ll]ocation: *//p' "$WORKDIR/hdr" 2>/dev/null | tr -d '\r' | tail -n 1)
    [ -n "$loc" ] || die "could not resolve artifact download URL"
    curl -fsSL "$loc" -o "$ZIP" || die "download failed: $DOWNLOAD_URL"
  else
    curl -fsSL "$DOWNLOAD_URL" -o "$ZIP" || die "download failed: $DOWNLOAD_URL"
  fi

  if [ -n "$DIGEST" ] && [ "$DIGEST" != "null" ]; then
    expected=${DIGEST#sha256:}
    if command -v sha256sum >/dev/null 2>&1; then actual=$(sha256sum "$ZIP" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then actual=$(shasum -a 256 "$ZIP" | awk '{print $1}')
    elif command -v python3 >/dev/null 2>&1; then actual=$(python3 -c 'import hashlib,sys; print(hashlib.sha256(open(sys.argv[1],"rb").read()).hexdigest())' "$ZIP")
    else actual=; fi
    if [ -n "$actual" ] && [ "$actual" = "$expected" ]; then info "Checksum verified"
    else die "checksum mismatch (expected $expected, got ${actual:-unavailable})"; fi
  fi

  if command -v unzip >/dev/null 2>&1; then unzip -j -qo "$ZIP" -d "$WORKDIR"
  elif command -v python3 >/dev/null 2>&1; then python3 - "$ZIP" "$WORKDIR" <<'PY'
import os, sys, zipfile
zip_path, dest = sys.argv[1], sys.argv[2]
os.makedirs(dest, exist_ok=True)
with zipfile.ZipFile(zip_path) as z:
    for m in z.infolist():
        if m.is_dir():
            continue
        name = os.path.basename(m.filename.replace("\\", "/"))
        if not name or name in (".", ".."):
            raise SystemExit("unsafe entry in archive: " + m.filename)
        out = os.path.join(dest, name)
        if os.path.abspath(out) != os.path.abspath(os.path.join(dest, name)):
            raise SystemExit("unsafe path in archive: " + m.filename)
        with z.open(m) as src, open(out, "wb") as dst:
            dst.write(src.read())
PY
  elif command -v bsdtar >/dev/null 2>&1; then bsdtar -xf "$ZIP" -C "$WORKDIR" --strip-components=99
  else die "unzip, python3, or bsdtar is required"; fi || die "could not extract the artifact"
  BIN_SRC="$WORKDIR/$ARTIFACT"
fi

[ -f "$BIN_SRC" ] || die "binary not found in downloaded package"

# --- install (sudo if needed) ----------------------------------------------
mkdir -p "$INSTALL_DIR" 2>/dev/null || true
if [ -w "$INSTALL_DIR" ] && [ -d "$INSTALL_DIR" ]; then
  install -m 755 "$BIN_SRC" "$INSTALL_DIR/dinky" 2>/dev/null || { cp "$BIN_SRC" "$INSTALL_DIR/dinky"; chmod 755 "$INSTALL_DIR/dinky"; }
else
  if ! command -v sudo >/dev/null 2>&1; then
    die "$INSTALL_DIR is not writable and sudo is not available; use --install-dir"
  fi
  sudo mkdir -p "$INSTALL_DIR"
  sudo install -m 755 "$BIN_SRC" "$INSTALL_DIR/dinky" || { sudo cp "$BIN_SRC" "$INSTALL_DIR/dinky"; sudo chmod 755 "$INSTALL_DIR/dinky"; }
fi

case ":$PATH:" in *":$INSTALL_DIR:"*) ;; *) info "Note: $INSTALL_DIR is not in your PATH." ;; esac
info "Installed dinky to $INSTALL_DIR/dinky"
if [ "$SOURCE" = release ]; then info "Source: release $RELEASE_TAG"; else info "Source: CI run $RUN_ID"; fi
info "Run it with: dinky [file...]"
