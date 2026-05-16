# loadout

Your installed software, as code.

`loadout` snapshots every package manager on your system into a single `loadout.toml` file. Commit it to git, then restore your exact setup on any machine — or use it as a reference when switching distros or package managers.

## Install

```sh
go install github.com/akvartz/loadout@latest
```

Or build from source:

```sh
git clone https://github.com/akvartz/loadout
cd loadout
go build -o loadout .
```

## Usage

### Snapshot your current setup

```sh
loadout snapshot
# saved to loadout.toml
```

Use `--verbose` to see each package manager as it runs:

```sh
loadout snapshot --verbose
# detecting apt...
#   apt: found 142 packages
# detecting flatpak...
#   flatpak: found 12 packages
# ...
```

### Check what has changed since your last snapshot

```sh
loadout diff
# [apt]
#   + htop
#   - cowsay
# [flatpak]
#   + org.gimp.GIMP
```

Exits with code `1` if there are differences — useful in shell scripts or cron jobs.

### Restore on a new machine

Dry-run (default) — prints the install commands without executing them:

```sh
loadout restore --target shell
```

```sh
#!/usr/bin/env sh
set -e

# apt
sudo apt-get install -y curl git htop ripgrep ...

# flatpak
flatpak install -y flathub org.signal.Signal
flatpak install -y flathub com.spotify.Client
...
```

Execute it directly with `--apply`:

```sh
loadout restore --target shell --apply
```

### Restore targets

| Target | Output | Best for |
|---|---|---|
| `shell` | Shell script using each native manager | Restoring on the same distro |
| `brewfile` | Homebrew `Brewfile` | Moving to macOS |
| `nix` | `home-manager` `home.packages` snippet | Moving to NixOS / home-manager |

For `brewfile` and `nix`, packages from other managers are listed as comments — cross-manager name translation is not automatic (see [Roadmap](#roadmap)).

### Use a custom file path

```sh
loadout snapshot --file ~/dotfiles/loadout.toml
loadout diff    --file ~/dotfiles/loadout.toml
loadout restore --file ~/dotfiles/loadout.toml --target nix
```

## Supported package managers

| Source | Detection method |
|---|---|
| `apt` | `apt-mark showmanual` |
| `flatpak` | `flatpak list --app` |
| `snap` | `snap list` |
| `pip` | `pip list --user --format=json` |
| `cargo` | `~/.cargo/.crates2.json` |
| `npm` | `npm list -g --depth=0 --json` |
| `appimage` | scans `~/Applications` and `~/.local/share/applications` |
| `brew` | `brew list --formula --casks` |
| `nix` | `nix-env --query --installed` |

Managers not present on the system are silently skipped.

## State file format

`loadout.toml` is plain TOML — commit it, diff it, edit it by hand:

```toml
[meta]
schema_version = 1
captured_at = 2026-05-15T14:32:00Z
hostname = "mybox"
os = "linux"

[sources.apt]
packages = ["curl", "git", "htop", "ripgrep"]

[sources.flatpak]
packages = ["org.signal.Signal", "com.spotify.Client"]

[sources.pip]
packages = ["httpie", "yt-dlp"]
```

## Roadmap

- **Cross-manager translation plugin** — convert `apt` package names to their `brew` or `nix` equivalents automatically

## License

GPL v3 — see [LICENSE](LICENSE).
