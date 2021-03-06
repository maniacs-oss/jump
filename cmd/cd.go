package cmd

import (
	"os"
	"path/filepath"

	"github.com/gsamokovarov/jump/cli"
	"github.com/gsamokovarov/jump/config"
	"github.com/gsamokovarov/jump/fuzzy"
	"github.com/gsamokovarov/jump/scoring"
)

const proximity = 5
const osSeparator = string(os.PathSeparator)

func cdCmd(args cli.Args, conf config.Config) {
	term := args.CommandName()
	entries, err := conf.ReadEntries()

	if err != nil {
		cli.Exitf(1, "%s\n", err)
	}

	// If an auto-completion triggered a full path, just go there.
	if filepath.IsAbs(term) {
		cli.Outf("%s\n", term)
		return
	}

	index, search := 0, conf.ReadSearch()

	// If we happen to match the last term, e.g. j is called with no
	// arguments then jump to the previous search.
	if term == "" {
		term, index = search.Term, search.Index+1
	} else {
		// If there is a term given, first see if there is a bin for
		// it and if so, always follow it.
		if dir, found := conf.FindPin(term); found {
			// Except if we land on the current directory again. Then
			// ignore the term.
			if !fwdPathIsCwd(dir) {
				cli.Outf("%s\n", dir)
				return
			}
		}
	}

	fuzzyEntries := scoring.NewFuzzyEntries(entries, term)

	// Prefer an exact match if it's in a reasonable proximity of the best
	// match. Useful for jumping to (2...4) letter directories, which you
	// may just type in their exact form anyway.
	index = exactMatchInProximity(fuzzyEntries, term, index)

	for {
		if entry, ok := fuzzyEntries.Select(index); ok {
			// Remove the entries that no longer exists.
			if _, err := os.Stat(entry.Path); os.IsNotExist(err) {
				entries.Remove(entry.Path)
				conf.WriteEntries(entries)

				index++
				continue
			}

			// Jump to the next entry, if the jump is going to land on the
			// current directory.
			if fwdPathIsCwd(entry.Path) {
				index++
				continue
			}

			cli.Outf("%s\n", entry.Path)
			conf.WriteSearch(term, index)
		}

		break
	}
}

func exactMatchInProximity(entries *scoring.FuzzyEntries, term string, offset int) int {
	norm := fuzzy.NewNormalizer(filepath.Base(term))
	normalizedTerm := norm.NormalizeTerm()

	for index := offset; index <= offset+proximity; index++ {
		if entry, ok := entries.Select(index); ok {

			// Take only the base part, if you wanna do a deep search
			// like Dev/nes.
			basePath := filepath.Base(norm.NormalizePath(entry.Path))
			if basePath == normalizedTerm {
				return index
			}
		}
	}

	return offset
}

func fwdPathIsCwd(path string) bool {
	cwd, err := os.Getwd()
	if err != nil {
		return false
	}

	fwdPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}

	return fwdPath == cwd
}

func init() {
	cli.RegisterCommand("cd", "Fuzzy match a directory to jump to.", cdCmd)
}
