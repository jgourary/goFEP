package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Auxiliary: contains utility functions with usage in multiple parts of the program
// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

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

// Return a slice of directories without files of type "ext" from a slice of directories
func getDirsWithoutFilesOfType(dirs []string, ext string) []string {

	// Create array to store subdirs w/o xxx files
	numSubDirs := len(dirs)
	noXXXSubDirs := make([]string, numSubDirs)

	// Create counter to record number of qualifying subdirectories found
	numSubDirsWithoutXXX := 0

	// Iterate through all subdirs
	for i := 0; i < numSubDirs; i++ {

		// read all files in subdir
		fileInfo, err := ioutil.ReadDir(dirs[i])
		if err != nil {
			fmt.Println("failed to read directory: " + dirs[i])
			log.Fatal(err)
		}

		// Search through all files and record if an "xxx" file is found
		xxxFileInDir := false
		for i := 0; i < len(fileInfo); i++ {
			if strings.Split(fileInfo[i].Name(), ".")[1] == ext {
				xxxFileInDir = true
			}
		}
		// If no "xxx" files in dir, add dir to relevant array
		if !xxxFileInDir {
			noXXXSubDirs[numSubDirsWithoutXXX] = dirs[i]
			// Iterate counter of qualifying subdirectories
			numSubDirsWithoutXXX++
		}
	}

	// Return array of subdirs w/o "xxx" files excluding empty entries
	if numSubDirsWithoutXXX > 0 {
		return noXXXSubDirs[:numSubDirsWithoutXXX]
	} else {
		return []string{}
	}
}


