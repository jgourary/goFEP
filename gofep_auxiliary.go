package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Auxiliary: contains utility functions with usage in multiple parts of the program
// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func removeContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

// Copy file copies a file from one location to another
// This functionality is not supported by a built in function in go
func copyFile(destPath string, sourcePath string) error {
	// Open source file
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		fmt.Println("failed to open file: " + sourcePath)
		log.Fatal(err)
	}

	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		fmt.Println("failed to create file: " + destPath)
		log.Fatal(err)
	}

	// Set permissions of destination file
	err = os.Chmod(destPath,octalPermissions)
	if err != nil {
		fmt.Println("failed to change permissions of file: " + destPath)
		log.Fatal(err)
	}

	// Copy contents of source file to destination file
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		fmt.Println("failed to copy contents of file: " + sourcePath + " to file " + destPath)
		log.Fatal(err)
	}

	// Close destination file
	err = destFile.Close()
	if err != nil {
		fmt.Println("failed to close file: " + destPath)
		log.Fatal(err)
	}

	// Close source file
	err = sourceFile.Close()
	if err != nil {
		fmt.Println("failed to close file: " + sourcePath)
		log.Fatal(err)
	}
	// return error, if any
	return err
}

