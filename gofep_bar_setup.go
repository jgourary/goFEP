package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

// BAR setup sets up the folders that BAR related files will be save in
func BARSetup(genPrm *generalParameters) {
	fmt.Println("\nBeginning BAR setup in directory: " + genPrm.targetDirectory)
	t1 := time.Now()

	dynDirectory := filepath.Join(genPrm.targetDirectory,"dynamic")

	// Get number of folders in dyn directory without arc files
	fmt.Println("\nChecking dynamic folders to verify output...")
	numBadFolders := len(getDynamicSubDirectories(dynDirectory, true))
	// If num > 0, flag error that BAR should not be run yet
	if numBadFolders > 0 {
		err := errors.New(strconv.Itoa(numBadFolders) + " subfolders in dynamic folder are missing arc files. This must be resolved" +
			"before BAR can be performed.")
		log.Fatal(err)
	}

	// Get pairings of dynamic folders to run BAR between
	fmt.Println("\nCalculating alphabetical pairings between dynamic folders...")
	barPairings := getBarPairings(genPrm.targetDirectory)
	// Create folders with names based on these pairings in the BAR directory
	fmt.Println("\nGenerating BAR folders accordingly...")
	createBarFolders(genPrm.targetDirectory, barPairings)

	t2 := time.Now()
	fmt.Println("\nBAR Setup finished in " + t2.Sub(t1).String())

}

// Pair up folders in dynamic directory alphabetically
func getBarPairings(directory string) [][]string {
	dynDirectory := filepath.Join(directory,"dynamic")

	// Read in all files in dir
	fileInfo, err := ioutil.ReadDir(dynDirectory)
	if err != nil {
		fmt.Println("Failed to read directory " + dynDirectory)
		log.Fatal(err)
	}

	// Get all dynamic dir names and sort alphabetically
	numDynDirs := len(fileInfo)
	dirNames := make([]string, numDynDirs)
	for i := 0; i < numDynDirs; i++ {
		dirNames[i] = fileInfo[i].Name()
	}
	// Make sure directories are sorted alphabetically
	sort.Strings(dirNames)

	// Pair dynamic dirs that are adjacent alphabetically
	barPairings := make([][]string, numDynDirs-1)
	for i := 0; i < numDynDirs-1; i++ {
		barPairings[i] = make([]string, 2)
		barPairings[i][0] = dirNames[i]
		barPairings[i][1] = dirNames[i+1]
	}
	// return paired names
	return barPairings
}

// Creates folders in BAR directory based on pairings
func createBarFolders(directory string, barPairings [][]string) {
	barDirectory := filepath.Join(directory, "bar")
	for i:=0; i<len(barPairings); i++ {
		folderPath := filepath.Join(barDirectory, barPairings[i][0] + "_" + barPairings[i][1])
		err := os.MkdirAll(folderPath, octalPermissions)
		if err != nil {
			fmt.Println("Failed to create directory " + folderPath)
			log.Fatal(err)
		}
	}
}
