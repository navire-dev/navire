#!/bin/sh

# Read changed repository paths from stdin and emit validation scopes.

all=false
go=false
deps=false
core=false
dspc=false
cli=false

while IFS= read -r path; do
  [ -z "$path" ] && continue

  case "$path" in
    go.mod|go.sum)
      go=true
      deps=true
      all=true
      ;;
    *.go)
      go=true
      ;;
  esac

  case "$path" in
    shared/*)
      all=true
      ;;
    core/*|cmd/core/*)
      core=true
      ;;
    dispatcher/*|cmd/dispatcher/*)
      dspc=true
      ;;
    navirectl/*|cmd/navirectl/*)
      cli=true
      ;;
    *.go)
      all=true
      ;;
  esac
done

if [ "$all" = true ]; then
  printf '%s\n' all
else
  [ "$core" = true ] && printf '%s\n' core
  [ "$dspc" = true ] && printf '%s\n' dspc
  [ "$cli" = true ] && printf '%s\n' cli
fi

[ "$go" = true ] && printf '%s\n' go
[ "$deps" = true ] && printf '%s\n' deps
