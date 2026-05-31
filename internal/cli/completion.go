package cli

import (
	"context"
	"fmt"
)

func (a *App) runCompletion(_ context.Context, args []string) int {
	shell := "zsh"
	if len(args) > 0 {
		shell = args[0]
	}

	switch shell {
	case "zsh":
		fmt.Fprint(a.out, zshCompletion)
	case "bash":
		fmt.Fprint(a.out, bashCompletion)
	case "fish":
		fmt.Fprint(a.out, fishCompletion)
	default:
		a.printError(usageError(fmt.Sprintf("unknown shell %q — supported: zsh, bash, fish", shell)))
		return ExitUsage
	}
	return ExitSuccess
}

const zshCompletion = `#compdef imole

_imole() {
  local state

  _arguments \
    '--debug[enable verbose debug output]' \
    '--help[show help]' \
    '--version[show version]' \
    '1: :->command' \
    '*: :->args'

  case $state in
    command)
      local commands=(
        'scan:scan iPhone media and app storage'
        'backup:copy media to local path with verification'
        'clean:delete verified files from iPhone'
        'report:summarise a backup manifest'
        'doctor:check device connection and dependencies'
        'history:show operation log'
        'schema:show machine-readable command schema'
        'guide:show detailed usage guide'
        'update:update imole to latest version'
        'completion:generate shell completion script'
        'help:show help'
        'version:show version'
      )
      _describe 'command' commands
      ;;
    args)
      case $words[2] in
        scan)
          _arguments \
            '--summary[compact stats]' \
            '--top[show top N files]:N' \
            '--only[filter by kind]:kind:(all photos videos)' \
            '--older-than[filter by age]:age:(30d 90d 6m 1y)' \
            '--large-than[filter by size]:size:(100MB 500MB 1GB)' \
            '--source[local DCIM path]:path:_files' \
            '--cache[use cached scan]' \
            '--json[output JSON]' \
            '--fields[JSON fields to include]:fields'
          ;;
        backup)
          _arguments \
            '--to[backup destination]:path:_files -/' \
            '--only[filter by kind]:kind:(all photos videos)' \
            '--older-than[filter by age]:age:(30d 90d 6m 1y)' \
            '--large-than[filter by size]:size:(100MB 500MB 1GB)' \
            '--source[local DCIM path]:path:_files' \
            '--file[specific file to backup]:rel_path' \
            '--dry-run[preview without copying]' \
            '--yes[skip confirmation]' \
            '--json[output JSON]'
          ;;
        clean)
          _arguments \
            '--manifest[path to manifest.json]:path:_files' \
            '--source[DCIM mount path]:path:_files' \
            '--file[specific verified file]:rel_path' \
            '--dry-run[preview without deleting]' \
            '--yes[skip confirmation]'
          ;;
        report)
          _arguments \
            '--manifest[path to manifest.json]:path:_files' \
            '--json[output JSON]'
          ;;
        completion)
          local shells=(zsh bash fish)
          _describe 'shell' shells
          ;;
        update)
          _arguments \
            '--check[check only, do not install]' \
            '--nightly[install latest unreleased build]'
          ;;
      esac
      ;;
  esac
}

_imole

# To enable, add this to your ~/.zshrc:
#   eval "$(imole completion zsh)"
# Or place the output in a $fpath directory as _imole.
`

const bashCompletion = `_imole_completion() {
  local cur prev commands
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  commands="scan backup clean report doctor history schema guide update completion help version"

  case "${COMP_CWORD}" in
    1)
      COMPREPLY=($(compgen -W "${commands}" -- "${cur}"))
      ;;
    *)
      case "${prev}" in
        scan)
          COMPREPLY=($(compgen -W "--summary --top --only --older-than --large-than --source --cache --json --fields" -- "${cur}"))
          ;;
        backup)
          COMPREPLY=($(compgen -W "--to --only --older-than --large-than --source --file --dry-run --yes --json" -- "${cur}"))
          ;;
        clean)
          COMPREPLY=($(compgen -W "--manifest --source --file --dry-run --yes" -- "${cur}"))
          ;;
        report)
          COMPREPLY=($(compgen -W "--manifest --json" -- "${cur}"))
          ;;
        completion)
          COMPREPLY=($(compgen -W "zsh bash fish" -- "${cur}"))
          ;;
        --to|--source|--manifest)
          COMPREPLY=($(compgen -f -- "${cur}"))
          ;;
        --only)
          COMPREPLY=($(compgen -W "all photos videos" -- "${cur}"))
          ;;
        *)
          COMPREPLY=($(compgen -W "${commands}" -- "${cur}"))
          ;;
      esac
      ;;
  esac
  return 0
}

complete -F _imole_completion imole

# To enable, add this to your ~/.bashrc or ~/.bash_profile:
#   eval "$(imole completion bash)"
`

const fishCompletion = `# imole fish completion

set -l commands scan backup clean report doctor history schema guide update completion help version

complete -c imole -f -n "not __fish_seen_subcommand_from $commands" -a scan      -d 'scan iPhone media and app storage'
complete -c imole -f -n "not __fish_seen_subcommand_from $commands" -a backup    -d 'copy media to local path with verification'
complete -c imole -f -n "not __fish_seen_subcommand_from $commands" -a clean     -d 'delete verified files from iPhone'
complete -c imole -f -n "not __fish_seen_subcommand_from $commands" -a report    -d 'summarise a backup manifest'
complete -c imole -f -n "not __fish_seen_subcommand_from $commands" -a doctor    -d 'check device connection and dependencies'
complete -c imole -f -n "not __fish_seen_subcommand_from $commands" -a history   -d 'show operation log'
complete -c imole -f -n "not __fish_seen_subcommand_from $commands" -a schema    -d 'show machine-readable command schema'
complete -c imole -f -n "not __fish_seen_subcommand_from $commands" -a guide     -d 'show detailed usage guide'
complete -c imole -f -n "not __fish_seen_subcommand_from $commands" -a update    -d 'update imole to latest version'
complete -c imole -f -n "not __fish_seen_subcommand_from $commands" -a completion -d 'generate shell completion script'

# scan flags
complete -c imole -n "__fish_seen_subcommand_from scan" -l summary    -d 'compact stats'
complete -c imole -n "__fish_seen_subcommand_from scan" -l top        -d 'show top N files'
complete -c imole -n "__fish_seen_subcommand_from scan" -l only       -d 'filter by kind' -a 'all photos videos'
complete -c imole -n "__fish_seen_subcommand_from scan" -l older-than -d 'filter by age'  -a '30d 90d 6m 1y'
complete -c imole -n "__fish_seen_subcommand_from scan" -l large-than -d 'filter by size' -a '100MB 500MB 1GB'
complete -c imole -n "__fish_seen_subcommand_from scan" -l cache      -d 'use cached result'
complete -c imole -n "__fish_seen_subcommand_from scan" -l json       -d 'output JSON'

# backup flags
complete -c imole -n "__fish_seen_subcommand_from backup" -l to         -d 'backup destination' -r
complete -c imole -n "__fish_seen_subcommand_from backup" -l only       -d 'filter by kind' -a 'all photos videos'
complete -c imole -n "__fish_seen_subcommand_from backup" -l older-than -d 'filter by age'  -a '30d 90d 6m 1y'
complete -c imole -n "__fish_seen_subcommand_from backup" -l dry-run    -d 'preview without copying'
complete -c imole -n "__fish_seen_subcommand_from backup" -l yes        -d 'skip confirmation'
complete -c imole -n "__fish_seen_subcommand_from backup" -l json       -d 'output JSON'

# clean flags
complete -c imole -n "__fish_seen_subcommand_from clean" -l manifest -d 'path to manifest.json' -r
complete -c imole -n "__fish_seen_subcommand_from clean" -l dry-run  -d 'preview without deleting'
complete -c imole -n "__fish_seen_subcommand_from clean" -l yes      -d 'skip confirmation'

# To enable, add this to your ~/.config/fish/config.fish:
#   imole completion fish | source
`
