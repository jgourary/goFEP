package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func DynamicSetup(genPrm generalParameters, setupPrm setupParameters, dynPrm []dynamicParameters) {

	// get sys time
	start := time.Now()

	fmt.Println("\nBeginning setup in directory: " + genPrm.targetDirectory)

	// Copy Prm File into directory in its own folder
	fmt.Println("\nSetting up Parameters Folder...")
	absPrmPath := copyPrmFile(genPrm.prmPath, genPrm.targetDirectory)

	// Get folder names to store xyz and key files
	fmt.Println("\nCalculating Folder Names...")
	dynamicFolders := getDynamicFolderNames(setupPrm)

	// Create folders to store xyz and key files
	fmt.Println("\nCreating Folders...")
	createDynamicFolders(genPrm.targetDirectory, dynamicFolders, dynPrm, genPrm)

	// Populate folders with xyz files
	fmt.Println("\nPopulating Folders with XYZ files...")
	createXYZFiles(genPrm.targetDirectory, genPrm.xyzPath, dynamicFolders)

	// Populate folders with key files
	fmt.Println("\nPopulating Folders with KEY files...")
	createKeyFiles(genPrm.targetDirectory, genPrm.keyPath, dynamicFolders, setupPrm, absPrmPath)

	end := time.Now()
	fmt.Println("\nSetup finished in " + end.Sub(start).String())
	fmt.Println()
}

// Calculate folder names for dynamic files based on vdw/ele parameters
func getDynamicFolderNames(prm setupParameters) []string {

	// Create array to save coeff combinations for usage in file and directory name
	numDirs := len(prm.vdw)
	dynamicFolders := make([]string, numDirs)

	// Iterate through all vdw/ele params and write combination to dynamicFolder array
	for i := 0; i < numDirs; i++ {

		// convert string (e.g. 0.50) to float, then to integer percent (e.g. 50)
		vdwFloat, err := strconv.ParseFloat(prm.vdw[i], 64)
		if err != nil {
			fmt.Println("Failed to convert vdwlambdas string to float")
			log.Fatal(err)
		}
		vdwPerc := int(100 * vdwFloat)

		eleFloat, err := strconv.ParseFloat(prm.ele[i], 64)
		if err != nil {
			fmt.Println("Failed to convert elelambdas string to float")
			log.Fatal(err)
		}
		elePerc := int(100 * eleFloat)

		// pad it with zeros to normalize lengths
		thisVdw := fmt.Sprintf("%03d", vdwPerc)
		thisEle := fmt.Sprintf("%03d", elePerc)

		// combine with separation underscore
		dynamicFolders[i] = "vdw" + thisVdw + "ele" + thisEle
	}

	return dynamicFolders
}

// Copy parameters file from current location to target directory
func copyPrmFile(sourcePrmPath string, targetDirectory string) string {

	// If source prm path is not absolute, redefine from target directory
	sourcePrmPath, err := filepath.Abs(sourcePrmPath)
	if err != nil {
		fmt.Println("Could not compute absolute path to parameters file location \"" + sourcePrmPath + "\" specified in general section of INI")
		log.Fatal(err)
	}

	// Get parameter name from sourcePrmPath
	prmName := filepath.Base(sourcePrmPath)

	// Create folder
	folderPath := filepath.Join(targetDirectory, "parameters")
	err = os.MkdirAll(folderPath, octalPermissions)
	if err != nil {
		fmt.Println("Failed to create directory to store parameters file: " + folderPath)
		log.Fatal(err)
	}

	// Create new prm file
	newPrmPath := filepath.Join(folderPath, prmName)
	err = copyFile(newPrmPath, sourcePrmPath)
	if err != nil {
		fmt.Println("Failed to copy parameters file to: " + newPrmPath + " from: " + sourcePrmPath)
		log.Fatal(err)
	}

	return newPrmPath
}

// Create subdirectories in dynamic folder based on folder names
func createDynamicFolders(directory string, dynamicFolders []string, dynPrm []dynamicParameters, genPrm generalParameters) {

	// Iterate through all folders to be created in ~/dynamic/
	for _, folderName := range dynamicFolders {

		// Create folder
		folderPath := filepath.Join(directory, "dynamic", folderName)
		err := os.MkdirAll(folderPath, octalPermissions)
		if err != nil {
			fmt.Println("Failed to create dynamic folder: " + folderPath)
			log.Fatal(err)
		}

		for i, dyn := range dynPrm {
			thisFile, _ := os.Create(filepath.Join(folderPath, "dyn_"+strconv.Itoa(i)+".sh"))
			path1 := filepath.Base(genPrm.xyzPath)
			path2 := strings.ReplaceAll(filepath.Base(genPrm.xyzPath), ".xyz", ".key")
			path3 := "dyn_" + strconv.Itoa(i) + ".log"
			thisFile.WriteString("dynamic_gpu " + path1 + " -k " + path2 + " " + dyn.numSteps +
				" " + dyn.stepInterval + " " + dyn.saveInterval + " " + dyn.ensemble + " " + dyn.temp + " " + dyn.pressure + " N > " + path3 + " \n")
		}
	}
}

// Populate dynamic folders with xyz files
func createXYZFiles(directory string, sourcePath string, dynamicFolders []string) {

	// If source path is not absolute already, redefine from CWD
	sourcePath, err := filepath.Abs(sourcePath)
	if err != nil {
		fmt.Println("Could not compute absolute path to xyz file location \"" + sourcePath + "\" specified in general block of INI")
		log.Fatal(err)
	}

	// Get name of xyz file
	xyzName := filepath.Base(sourcePath)

	// Iterate through all folders in ~/dynamic/
	for i := 0; i < len(dynamicFolders); i++ {
		// copy xyz file to dir
		destPath := filepath.Join(directory, "dynamic", dynamicFolders[i], xyzName)
		err := copyFile(destPath, sourcePath)
		if err != nil {
			fmt.Println("Failed to copy xyz file from : " + sourcePath + " to: " + destPath)
			log.Fatal(err)
		}
	}
}

// Populate dynamic folders with key files
func createKeyFiles(directory string, sourcePath string, dynamicFolders []string, prm setupParameters, absPrmPath string) {

	// If source prm path is not absolute already, redefine from target directory
	sourcePath, err := filepath.Abs(sourcePath)
	if err != nil {
		fmt.Println("Could not compute absolute path to key file location \"" + sourcePath + "\" specified in general block of INI")
		log.Fatal(err)
	}

	// Iterate through all folders in ~/dynamic/
	for i := 0; i < len(dynamicFolders); i++ {

		// Open source key file
		keyName := filepath.Base(sourcePath)
		sourceKey, err := os.Open(sourcePath)
		if err != nil {
			fmt.Println("Failed to open key file: " + sourcePath)
			log.Fatal(err)
		}

		// Get folder path
		folderPath := filepath.Join(directory, "dynamic", dynamicFolders[i])
		keyPath := filepath.Join(folderPath, keyName)

		// Create new key file
		newKey, err := os.Create(keyPath)
		if err != nil {
			fmt.Println("Failed to create new key file: " + keyPath)
			log.Fatal(err)
		}

		// Read source key file line by line
		scanner := bufio.NewScanner(sourceKey)
		for scanner.Scan() {
			// get next line
			line := scanner.Text()
			// if line specifies vdw-lambda, ele-lambda or restrain-groups, replace value and write altered line to new key file
			if strings.Count(line, "vdw-lambda") > 0 {
				_, err = newKey.WriteString("vdw-lambda " + prm.vdw[i] + "\n")
				if err != nil {
					fmt.Println("Failed to write vdw-lambda line to key: " + keyPath)
					log.Fatal(err)
				}
			} else if strings.Count(line, "ele-lambda") > 0 {
				_, err = newKey.WriteString("ele-lambda " + prm.ele[i] + "\n")
				if err != nil {
					fmt.Println("Failed to write ele-lambda line to key: " + keyPath)
					log.Fatal(err)
				}
			} else if strings.Count(line, "restrain-groups") > 0 {
				tokens := strings.Fields(line)
				tokens[3] = prm.rst[i]
				for i := 0; i < len(tokens); i++ {
					_, err = newKey.WriteString(tokens[i] + " ")
					if err != nil {
						fmt.Println("Failed to write restrain-groups line to key: " + keyPath)
						log.Fatal(err)
					}
				}
				_, err = newKey.WriteString("\n")
				if err != nil {
					log.Fatal(err)
				}
			} else if strings.Count(line, "parameters") > 0 {
				_, err = newKey.WriteString("parameters ../../" + filepath.Base(absPrmPath) + "\n")
				if err != nil {
					fmt.Println("Failed to write parameters line to key: " + keyPath)
					log.Fatal(err)
				}

			} else { // else write line to new key file unchanged
				_, err = newKey.WriteString(line + "\n")
				if err != nil {
					fmt.Println("Failed copy line: \"" + line + "\" to key: " + keyPath + " from key: " + sourcePath)
					log.Fatal(err)
				}
			}
		}
		// Close new key file
		err = newKey.Close()
		if err != nil {
			fmt.Println("Failed to close key: " + keyPath)
			log.Fatal(err)
		}
		// Close source key file
		err = sourceKey.Close()
		if err != nil {
			fmt.Println("Failed to close key: " + sourcePath)
			log.Fatal(err)
		}
	}

}
