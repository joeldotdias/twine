package repository

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

const (
	tableWidth   = 80
	leftBorder   = "│ "
	rightBorder  = " │"
	contentWidth = tableWidth - len(leftBorder) - len(rightBorder)
)

func (repo *Repository) dbg() {
	_, err := exec.LookPath("less")
	// in case less isn't available, print to stdout directly
	if err != nil {
		repo.dbgInfo(os.Stdout)
		return
	}

	reader, writer := io.Pipe()
	less := exec.Command("less", "-R")
	less.Stdin = reader
	less.Stdout = os.Stdout
	less.Stderr = os.Stderr

	err = less.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting less: %v\n", err)
		// fallback to printing directly to stdout
		repo.dbgInfo(os.Stdout)
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		defer writer.Close()
		repo.dbgInfo(writer)
	}()

	wg.Wait()

	err = less.Wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running less: %v\n", err)
	}
}

func (repo *Repository) dbgInfo(w io.Writer) {
	mainHeader(w)

	field("Worktree", repo.worktree, w)
	field("Git Directory", repo.gitDir, w)

	sectionHeader("Config", w)
	field("Username", repo.conf.username, w)
	field("Email", repo.conf.email, w)
	field("Default Branch", repo.conf.defaultBranch, w)

	sectionHeader("RefStore", w)
	lineWithoutColon("Heads:", w)
	for name, ref := range repo.refStore.heads {
		field("  "+name, ref, w)
	}
	if len(repo.refStore.tags) > 0 {
		lineWithoutColon("Tags:", w)
		for name, ref := range repo.refStore.tags {
			field("  "+name, ref, w)
		}
	}

	sectionHeader("Index", w)
	field("Signature", string(repo.index.header.Signature[:]), w)
	field("Version", fmt.Sprintf("%d", repo.index.header.Version), w)
	field("Number of Entries", fmt.Sprintf("%d", repo.index.header.NumEntries), w)

	lineWithoutColon("Entries:", w)
	for i, entry := range repo.index.entries {
		lineWithoutColon(fmt.Sprintf("Entry %d:", i+1), w)
		field("  Path", entry.path, w)
		field("  SHA", fmt.Sprintf("%x", entry.sha), w)
		field("  Mode", fmt.Sprintf("%o", entry.mode), w)
		field("  Size", fmt.Sprintf("%d bytes", entry.size), w)
		field("  Flags", showFlags(entry.flags), w)
		if i < len(repo.index.entries)-1 {
			horizontalLine("-", w)
		}
	}

	horizontalLine("=", w)
}

func mainHeader(w io.Writer) {
	horizontalLine("=", w)
	fmt.Fprintf(w, "%s%-*s    %s\n", leftBorder, contentWidth, "Twine Debug Info", rightBorder)
	horizontalLine("=", w)
}

func sectionHeader(title string, w io.Writer) {
	horizontalLine("-", w)
	fmt.Fprintf(w, "%s%-*s    %s\n", leftBorder, contentWidth, title, rightBorder)
	horizontalLine("-", w)
}

func field(name, value string, w io.Writer) {
	nameWidth := 20
	valueWidth := contentWidth - nameWidth - 2 // -2 for ": " separator
	fmt.Fprintf(w, "%s%-*s: %-*s%s\n", leftBorder, nameWidth, name, valueWidth, truncateOrPad(value, valueWidth), rightBorder)
}

func lineWithoutColon(text string, w io.Writer) {
	fmt.Fprintf(w, "%s%-*s    %s\n", leftBorder, contentWidth, text, rightBorder)
}

func horizontalLine(char string, w io.Writer) {
	fmt.Fprintf(w, "+%s+\n", strings.Repeat(char, tableWidth-2))
}

func truncateOrPad(s string, width int) string {
	if len(s) > width {
		return s[:width]
	}
	return fmt.Sprintf("%-*s", width+4, s)
}

func showFlags(flags uint16) string {
	indexFlags := IndexFlags{
		assumeValid: flags&0x8000 != 0,
		extended:    flags&0x4000 != 0,
		stage:       flags & 0x3000,
	}

	flagStrs := []string{
		fmt.Sprintf("assume-valid: %t", indexFlags.assumeValid),
		fmt.Sprintf("extended: %t", indexFlags.extended),
		fmt.Sprintf("stage: %d", indexFlags.stage),
	}

	return strings.Join(flagStrs, ", ")
}
