#!/usr/bin/env bash
set -euo pipefail
APP=flowmi

MUTED='\033[0;2m'
RED='\033[0;31m'
ORANGE='\033[38;5;214m'
NC='\033[0m' # No Color

GITHUB_REPO="flowmi-ai/flowmi"

usage() {
    cat <<EOF
Flowmi CLI Installer

Usage: install.sh [options]

Options:
    -h, --help              Display this help message
    -v, --version <version> Install a specific version (e.g., 0.1.0)
    -b, --binary <path>     Install from a local binary instead of downloading
        --no-modify-path    Don't modify shell config files (.zshrc, .bashrc, etc.)

Examples:
    curl -fsSL https://flowmi.ai/install | bash
    curl -fsSL https://flowmi.ai/install | bash -s -- --version 0.1.0
    ./install.sh --binary /path/to/flowmi
EOF
}

requested_version=${VERSION:-}
no_modify_path=false
binary_path=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help)
            usage
            exit 0
            ;;
        -v|--version)
            if [[ -n "${2:-}" ]]; then
                requested_version="$2"
                shift 2
            else
                echo -e "${RED}Error: --version requires a version argument${NC}"
                exit 1
            fi
            ;;
        -b|--binary)
            if [[ -n "${2:-}" ]]; then
                binary_path="$2"
                shift 2
            else
                echo -e "${RED}Error: --binary requires a path argument${NC}"
                exit 1
            fi
            ;;
        --no-modify-path)
            no_modify_path=true
            shift
            ;;
        *)
            echo -e "${ORANGE}Warning: Unknown option '$1'${NC}" >&2
            shift
            ;;
    esac
done

INSTALL_DIR=$HOME/.local/bin
mkdir -p "$INSTALL_DIR"

# If --binary is provided, skip all download/detection logic
if [ -n "$binary_path" ]; then
    if [ ! -f "$binary_path" ]; then
        echo -e "${RED}Error: Binary not found at ${binary_path}${NC}"
        exit 1
    fi
    specific_version="local"
else
    raw_os=$(uname -s)
    case "$raw_os" in
      Darwin*) os="darwin" ;;
      Linux*)  os="linux" ;;
      *)
        echo -e "${RED}Unsupported OS: $raw_os${NC}"
        exit 1
        ;;
    esac

    arch=$(uname -m)
    case "$arch" in
      x86_64)  arch="amd64" ;;
      aarch64) arch="arm64" ;;
      arm64)   arch="arm64" ;;
      *)
        echo -e "${RED}Unsupported architecture: $arch${NC}"
        exit 1
        ;;
    esac

    # Detect Rosetta on macOS — prefer native arm64
    if [ "$os" = "darwin" ] && [ "$arch" = "amd64" ]; then
        rosetta_flag=$(sysctl -n sysctl.proc_translated 2>/dev/null || echo 0)
        if [ "$rosetta_flag" = "1" ]; then
            arch="arm64"
        fi
    fi

    if ! command -v tar >/dev/null 2>&1; then
        echo -e "${RED}Error: 'tar' is required but not installed.${NC}"
        exit 1
    fi

    if [ -z "$requested_version" ]; then
        specific_version=$(curl -s "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | sed -n 's/.*"tag_name": *"v\([^"]*\)".*/\1/p')

        if [[ -z "$specific_version" ]]; then
            echo -e "${RED}Failed to fetch latest version${NC}"
            exit 1
        fi
    else
        # Strip leading 'v' if present
        requested_version="${requested_version#v}"
        specific_version=$requested_version

        # Verify the release exists
        http_status=$(curl -sI -o /dev/null -w "%{http_code}" "https://github.com/${GITHUB_REPO}/releases/tag/v${requested_version}")
        if [ "$http_status" = "404" ]; then
            echo -e "${RED}Error: Release v${requested_version} not found${NC}"
            echo -e "${MUTED}Available releases: https://github.com/${GITHUB_REPO}/releases${NC}"
            exit 1
        fi
    fi

    filename="${APP}_${specific_version}_${os}_${arch}.tar.gz"
    url="https://github.com/${GITHUB_REPO}/releases/download/v${specific_version}/${filename}"
fi

print_message() {
    local level=$1
    local message=$2
    local color=""

    case $level in
        info) color="${NC}" ;;
        warning) color="${NC}" ;;
        error) color="${RED}" ;;
    esac

    echo -e "${color}${message}${NC}"
}

check_version() {
    if [ -x "${INSTALL_DIR}/flowmi" ]; then
        installed_version=$("${INSTALL_DIR}/flowmi" version 2>/dev/null | head -1 | awk '{print $2}' || true)

        if [[ -n "$installed_version" && "$installed_version" == "$specific_version" ]]; then
            print_message info "${MUTED}Version ${NC}$specific_version${MUTED} already installed in ${NC}${INSTALL_DIR}${NC}"
            already_installed=true
        elif [[ -n "$installed_version" ]]; then
            print_message info "${MUTED}Installed version: ${NC}$installed_version"
        else
            print_message info "${MUTED}Existing binary in ${NC}${INSTALL_DIR}${MUTED} appears broken, reinstalling${NC}"
        fi
    fi
}

unbuffered_sed() {
    if echo | sed -u -e "" >/dev/null 2>&1; then
        sed -nu "$@"
    elif echo | sed -l -e "" >/dev/null 2>&1; then
        sed -nl "$@"
    else
        local pad="$(printf "\n%512s" "")"
        sed -ne "s/$/\\${pad}/" "$@"
    fi
}

print_progress() {
    local bytes="$1"
    local length="$2"
    [ "$length" -gt 0 ] || return 0

    local width=50
    local percent=$(( bytes * 100 / length ))
    [ "$percent" -gt 100 ] && percent=100
    local on=$(( percent * width / 100 ))
    local off=$(( width - on ))

    local filled=$(printf "%*s" "$on" "")
    filled=${filled// /■}
    local empty=$(printf "%*s" "$off" "")
    empty=${empty// /･}

    printf "\r${ORANGE}%s%s %3d%%${NC}" "$filled" "$empty" "$percent" >&4
}

download_with_progress() {
    local url="$1"
    local output="$2"

    if [ -t 2 ]; then
        exec 4>&2
    else
        exec 4>/dev/null
    fi

    local tmp_dir=${TMPDIR:-/tmp}
    local basename="${tmp_dir}/${APP}_install_$$"
    local tracefile="${basename}.trace"

    rm -f "$tracefile"
    mkfifo "$tracefile"

    # Hide cursor
    printf "\033[?25l" >&4

    trap "trap - RETURN; rm -f \"$tracefile\"; printf '\033[?25h' >&4; exec 4>&-" RETURN

    (
        curl --trace-ascii "$tracefile" -s -L -o "$output" "$url"
    ) &
    local curl_pid=$!

    unbuffered_sed \
        -e 'y/ACDEGHLNORTV/acdeghlnortv/' \
        -e '/^0000: content-length:/p' \
        -e '/^<= recv data/p' \
        "$tracefile" | \
    {
        local length=0
        local bytes=0

        while IFS=" " read -r -a line; do
            [ "${#line[@]}" -lt 2 ] && continue
            local tag="${line[0]} ${line[1]}"

            if [ "$tag" = "0000: content-length:" ]; then
                length="${line[2]}"
                length=$(echo "$length" | tr -d '\r')
                bytes=0
            elif [ "$tag" = "<= recv" ]; then
                local size="${line[3]}"
                bytes=$(( bytes + size ))
                if [ "$length" -gt 0 ]; then
                    print_progress "$bytes" "$length"
                fi
            fi
        done
    }

    wait $curl_pid
    local ret=$?
    echo "" >&4
    return $ret
}

download_and_install() {
    print_message info "\n${MUTED}Installing ${NC}flowmi ${MUTED}version: ${NC}$specific_version"
    local tmp_dir="${TMPDIR:-/tmp}/${APP}_install_$$"
    mkdir -p "$tmp_dir"
    trap "rm -rf \"$tmp_dir\"" EXIT

    if ! [ -t 2 ] || ! download_with_progress "$url" "$tmp_dir/$filename"; then
        # Fallback to standard curl in non-TTY environments or if custom progress fails
        curl -# -L -o "$tmp_dir/$filename" "$url"
    fi

    # Verify checksum
    local checksums_url="https://github.com/${GITHUB_REPO}/releases/download/v${specific_version}/checksums.txt"
    if curl -sL -o "$tmp_dir/checksums.txt" "$checksums_url"; then
        local expected
        expected=$(grep "$filename" "$tmp_dir/checksums.txt" | awk '{print $1}')
        if [ -n "$expected" ]; then
            local actual
            if command -v sha256sum >/dev/null 2>&1; then
                actual=$(sha256sum "$tmp_dir/$filename" | awk '{print $1}')
            elif command -v shasum >/dev/null 2>&1; then
                actual=$(shasum -a 256 "$tmp_dir/$filename" | awk '{print $1}')
            fi
            if [ -n "${actual:-}" ] && [ "$actual" != "$expected" ]; then
                echo -e "${RED}Error: Checksum verification failed${NC}"
                echo -e "${MUTED}Expected: ${NC}$expected"
                echo -e "${MUTED}Actual:   ${NC}$actual"
                exit 1
            fi
        fi
    fi

    tar -xzf "$tmp_dir/$filename" -C "$tmp_dir"

    mv "$tmp_dir/flowmi" "$INSTALL_DIR/"
    chmod 755 "${INSTALL_DIR}/flowmi"

    # Create fm symlink
    ln -sf "${INSTALL_DIR}/flowmi" "${INSTALL_DIR}/fm"

    rm -rf "$tmp_dir"
    trap - EXIT
}

install_from_binary() {
    print_message info "\n${MUTED}Installing ${NC}flowmi ${MUTED}from: ${NC}$binary_path"
    cp "$binary_path" "${INSTALL_DIR}/flowmi"
    chmod 755 "${INSTALL_DIR}/flowmi"

    # Create fm symlink
    ln -sf "${INSTALL_DIR}/flowmi" "${INSTALL_DIR}/fm"
}

already_installed=false

if [ -n "$binary_path" ]; then
    install_from_binary
else
    check_version
    if [ "$already_installed" != "true" ]; then
        download_and_install
    else
        # Ensure fm symlink exists even when skipping download
        ln -sf "${INSTALL_DIR}/flowmi" "${INSTALL_DIR}/fm"
    fi
fi

add_to_path() {
    local config_file=$1
    local command=$2

    if grep -Fxq "$command" "$config_file"; then
        print_message info "Command already exists in $config_file, skipping write."
    else
        echo -e "\n# flowmi" >> "$config_file"
        echo "$command" >> "$config_file"
        print_message info "${MUTED}Successfully added ${NC}flowmi ${MUTED}to \$PATH in ${NC}$config_file"
    fi
}

XDG_CONFIG_HOME=${XDG_CONFIG_HOME:-$HOME/.config}

current_shell=$(basename "$SHELL")

# Determine rc file candidates and the default to create if none exist.
# macOS opens login shells (reads .bash_profile, not .bashrc).
# Linux opens non-login shells (reads .bashrc, not .bash_profile).
fish_config_dir="$XDG_CONFIG_HOME/fish"

case $current_shell in
    fish)
        config_files="$fish_config_dir/config.fish"
        default_config="$fish_config_dir/config.fish"
    ;;
    zsh)
        config_files="${ZDOTDIR:-$HOME}/.zshrc ${ZDOTDIR:-$HOME}/.zshenv $XDG_CONFIG_HOME/zsh/.zshrc $XDG_CONFIG_HOME/zsh/.zshenv"
        default_config="${ZDOTDIR:-$HOME}/.zshrc"
    ;;
    bash)
        if [ "$(uname -s)" = "Darwin" ]; then
            config_files="$HOME/.bash_profile $HOME/.profile $HOME/.bashrc"
            default_config="$HOME/.bash_profile"
        else
            config_files="$HOME/.bashrc $HOME/.bash_profile $HOME/.profile"
            default_config="$HOME/.bashrc"
        fi
    ;;
    ash|sh)
        config_files="$HOME/.profile /etc/profile $HOME/.ashrc"
        default_config="$HOME/.profile"
    ;;
    *)
        config_files="$HOME/.profile $HOME/.bash_profile $HOME/.bashrc"
        default_config="$HOME/.profile"
    ;;
esac

if [[ "$no_modify_path" != "true" ]]; then
    config_file=""
    for file in $config_files; do
        if [[ -f $file && -w $file ]]; then
            config_file=$file
            break
        fi
    done

    # If no existing writable rc file, create the default one
    if [[ -z $config_file ]]; then
        config_dir=$(dirname "$default_config")
        if [[ ! -d "$config_dir" ]]; then
            mkdir -p "$config_dir"
        fi
        touch "$default_config"
        config_file="$default_config"
        print_message info "${MUTED}Created ${NC}$config_file"
    fi

    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        case $current_shell in
            fish)
                add_to_path "$config_file" "fish_add_path $INSTALL_DIR"
            ;;
            *)
                add_to_path "$config_file" "export PATH=$INSTALL_DIR:\$PATH"
            ;;
        esac
    fi
fi

if [ -n "${GITHUB_ACTIONS-}" ] && [ "${GITHUB_ACTIONS}" == "true" ]; then
    echo "$INSTALL_DIR" >> "$GITHUB_PATH"
    print_message info "Added $INSTALL_DIR to \$GITHUB_PATH"
fi

echo -e ""
echo -e "${MUTED}▄▄  ▄▄${NC}"
echo -e "${MUTED}█ ▀▄▀ █  ${NC}flowmi ${MUTED}v${specific_version}${NC}"
echo -e "${MUTED}█    ▄▀${NC}"
echo -e "${MUTED}▀   ▀${NC}"
echo -e ""
echo -e "${MUTED}To get started:${NC}"
echo -e ""
echo -e "  fm auth login   ${MUTED}# Authenticate${NC}"
echo -e "  fm note list    ${MUTED}# List your notes${NC}"
echo -e "  fm --help       ${MUTED}# See all commands${NC}"
echo -e ""
echo -e "${MUTED}Docs: ${NC}https://github.com/${GITHUB_REPO}"
echo -e ""
