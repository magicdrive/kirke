# kirke command completion for Bash

_kirke_completions() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    opts="
        -h --help
        -v --version
        -n --name
        -f --file
        -a --null-as
        -j --json
        -p --pipe
        --pointer --pointer-struct
        --pager --auto-pager
        -o --outline
        -i --inline
    "

    # Completion logic for specific options
    case "$prev" in
        -n|--name)
            COMPREPLY=( $(compgen -W "root struct name" -- "$cur") )
            return 0
            ;;
        -f|--file)
            COMPREPLY=( $(compgen -f -- "$cur") )
            return 0
            ;;
        -a|--null-as)
            COMPREPLY=( $(compgen -W "any interface{} int string bool *string *int *bool" -- "$cur") )
            return 0
            ;;
        -j|--json)
            COMPREPLY=( $(compgen -W "JSON string" -- "$cur") )
            return 0
            ;;
        --pointer|--pointer-struct|--pager|--auto-pager)
            COMPREPLY=( $(compgen -W "on off" -- "$cur") )
            return 0
            ;;
    esac

    # Provide general options and argument suggestions
    if [[ ${cur} == -* ]]; then
        COMPREPLY=( $(compgen -W "${opts}" -- "$cur") )
        return 0
    fi

    # Provide argument suggestions
    COMPREPLY=( $(compgen -W "root-name json-file json-string null-type-name" -- "$cur") )
    return 0
}

# Register completion function for Bash
complete -F _kirke_completions kirke

