package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	err = filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Base(path) == "OWNERS" {
			relPath, err := filepath.Rel(cwd, filepath.Dir(path))
			if err != nil {
				return err
			}

			parentOwners, err := collectParentOwners(cwd, relPath)
			if err != nil {
				return err
			}

			owners, err := parseOwnersFile(path)
			if err != nil {
				return err
			}

			allOwners := append(parentOwners, owners...)
			if relPath == "." {
				relPath = ""
			} else {
				relPath = fmt.Sprintf("/%s", relPath)
			}
			fmt.Printf("%s/**/* %s\n", relPath, strings.Join(allOwners, " "))
		}

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func parseOwnersFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	owners := []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		owner := scanner.Text()
		if owner != "" {
			owners = append(owners, owner)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return owners, nil
}

func collectParentOwners(cwd, relPath string) ([]string, error) {
	parents := strings.Split(relPath, string(filepath.Separator))
	parentOwners := []string{}

	for i := range parents {
		parentPath := filepath.Join(cwd, filepath.Join(parents[:i]...))
		ownersFile := filepath.Join(parentPath, "OWNERS")

		if _, err := os.Stat(ownersFile); !os.IsNotExist(err) {
			owners, err := parseOwnersFile(ownersFile)
			if err != nil {
				return nil, err
			}
			parentOwners = append(parentOwners, owners...)
		}
	}

	return parentOwners, nil
}
