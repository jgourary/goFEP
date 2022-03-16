package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// BAR: contains functions relevant to running Tinker's BAR 1 & BAR 2 programs
// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// BARManager is the "API" to this file. It manages functions AutoBAR1 & AutoBAR2 and sees that they run in order
func (ng nodeGroup) BARManager(genPrm *generalParameters, barPrm *barParameters, maxNodes int) {

	// Find subdirectories to run BAR inside
	fmt.Println("\nVerifying bar subdirectories...")
	subDirs, err := getBARSubDirs(genPrm.targetDirectory)
	if err != nil {
		fmt.Println("Failed to validate subdirectories for bar")
		log.Fatal(err)
	}

	// Run AutoBAR1
	fmt.Println("\nPreparing to run AutoBAR 1...")
	t1 := time.Now()
	ng.autoBAR1(subDirs, genPrm, barPrm, maxNodes)
	t2 := time.Now()
	fmt.Println("\nAutoBAR 1 finished in " + t2.Sub(t1).String())

	// Run AutoBAR2
	fmt.Println("\nPreparing to run AutoBAR 2...")
	t1 = time.Now()
	ng.autoBAR2(subDirs, genPrm, barPrm, maxNodes)
	t2 = time.Now()
	fmt.Println("\nAutoBAR 2 finished in " + t2.Sub(t1).String())
}

// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// BAR 1
// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// AutoBAR1, managed by BARManager, runs BAR1 in all subdirectories provided in parallel on different cluster nodes
func (ng nodeGroup) autoBAR1(subDirs []string, genPrm *generalParameters, barPrm *barParameters, maxNodes int) {

	// Get nodes to run BAR1 on
	fmt.Print("\nLooking for " + strconv.Itoa(maxNodes) + " available nodes...")
	t1 := time.Now()
	updateStatus(&ng, genPrm.targetDirectory)
	t2 := time.Now()
	fmt.Print("found " + strconv.Itoa(len(ng.freeNodeIndices)) + " available nodes in " + t2.Sub(t1).String())
	// Check that the number of nodes found isn't 0
	if len(ng.freeNodeIndices) < 1 {
		err := errors.New("did not find enough free nodes to run on - exiting")
		log.Fatal(err)
	}

	// Create new wait group to determine when all goroutines have finished
	wg := sync.WaitGroup{}

	// iterate through all subDirs
	fmt.Println("\nBeginning AutoBAR1 run on " + strconv.Itoa(len(subDirs)) + " files...\n")
	for i := 0; i < len(subDirs); i++ {
		// Determine which node to run BAR1 on by iterating through the list of free nodes
		nodeIndex := ng.freeNodeIndices[i%len(ng.freeNodeIndices)]
		// Add one to wait group
		wg.Add(1)
		// Launch go routine (parallel processing) to run BAR1 on selected node in subDir[i]
		go ng.nodes[nodeIndex].BAR1(subDirs[i], genPrm, barPrm, &wg)
	}

	// Wait here until all goroutines have finished
	wg.Wait()

}

// BAR1, managed by AutoBAR1, runs BAR1 in all subdirectory specified on node specified
func (n node) BAR1(subBarDir string, genPrm *generalParameters, barPrm *barParameters, wg *sync.WaitGroup) {

	// Get paths to ARC files to run BAR 1 on
	arcFilePaths, err := getBAR1FilePaths(genPrm.targetDirectory, subBarDir)
	if err != nil {
		fmt.Println("Failed to find file paths of ARC files to run BAR 1 with")
		log.Fatal(err)
	}
	arc1Path := arcFilePaths[0]
	arc2Path := arcFilePaths[1]

	// By default, Tinker writes BAR1 output to a file in the same directory as the first ARC file with the same name
	defOutputPath := strings.TrimSuffix(arc1Path, "arc") + "bar"
	// We would like to move output from there (targetDirectory/dynamic/subDynDir)
	// to the directory we are running bar in (targetDirectory/bar/subBarDir) for organizational purposes
	intendedBaseFileName := strings.TrimSuffix(filepath.Base(arc1Path), "arc")
	intendedOutputPath := filepath.Join(subBarDir, intendedBaseFileName+"bar")

	// Write script to run BAR 1 and save to BAR subdirectory
	bar1Script := createTempBAR1Script(subBarDir, arc1Path, arc2Path, genPrm, barPrm, &n)

	// Run bar 1 script we just wrote
	out, err := exec.Command("sh", bar1Script).CombinedOutput()
	// Report results to user
	if err != nil {
		fmt.Print("Error encountered on files in subdirectory " + filepath.Dir(arc1Path) + " and " + filepath.Dir(arc2Path) + " using node " + n.name)
		fmt.Println(err)

		// get filepath to write error to
		outFilePath := filepath.Join(subBarDir, "bar1.err")
		// create file to store err
		outFile, err := os.Create(outFilePath)
		if err != nil {
			fmt.Println("failed to create error log: " + outFilePath)
			log.Fatal(err)
		}
		// Set file permissions jic
		err = os.Chmod(outFilePath, octalPermissions)
		if err != nil {
			err = errors.New("failed to change temp file permissions for bar bash script")
			log.Fatal(err)
		}
		// Write output to file
		_, err = outFile.WriteString(string(out))
		fmt.Println()

	} else {
		fmt.Println("BAR1 finished successfully on files in subdirectories " + filepath.Dir(arc1Path) + " and " + filepath.Dir(arc2Path) + " using node " + n.name)
	}

	// move output file from default location to new one
	err = copyFile(intendedOutputPath, defOutputPath)
	if err != nil {
		fmt.Println("Failed to copy BAR1 output from " + defOutputPath + " to " + intendedOutputPath)
	}
	err = os.Remove(defOutputPath)
	if err != nil {
		fmt.Println("Failed to remove initial BAR1 output file from " + defOutputPath)
	}

	// subtract one from wg count
	wg.Done()
}

// Write a bash script to perform BAR1 and save to directory specified
func createTempBAR1Script(subBarDir string, arc1Path string, arc2Path string, genPrm *generalParameters, barPrm *barParameters, n *node) string {

	// Write temp bash script in dir
	filePath := filepath.Join(subBarDir, "bar1.sh")
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("failed to create temporary BAR 1 bash script: " + filePath)
		log.Fatal(err)
	}

	// Set file permissions jic
	err = os.Chmod(filePath, octalPermissions)
	if err != nil {
		err = errors.New("failed to change temp file permissions for BAR 1 bash script")
		log.Fatal(err)
	}

	// get log path
	logPath := filepath.Join(subBarDir, "bar1.log")

	// Start with header
	_, err = file.WriteString("#!/bin/bash\n")
	// Begin here document (all following command will be performed inside node)
	_, err = file.WriteString("ssh -o \"StrictHostKeyChecking no\" " + n.name + " << END\n")
	// Source universally needed files
	_, err = file.WriteString("\tsource " + genPrm.intelSource + "\n")
	// Source gpu dependant files
	var openMMHome string
	// Source CUDA files and get openMMHome variable
	if n.cardGeneration == "Ampere" {
		_, err = file.WriteString("\tsource " + genPrm.cuda11Source + "\n")
		openMMHome = genPrm.cuda11Home
	} else if n.cardGeneration == "Turing" {
		_, err = file.WriteString("\tsource " + genPrm.cuda10Source + "\n")
		openMMHome = genPrm.cuda10Home
	} else {
		err = errors.New("card generation unrecognized - unsure which files to source. Recognized generations are " +
			"\"Maxwell\", \"Pascal\", \"Turing\". Check entry of node \"" + n.name + "\" in node INI file")
		log.Fatal(err)
	}
	// Get card number
	_, err = file.WriteString("\texport CUDA_VISIBLE_DEVICES=" + n.cardNumber + "\n")
	// Write command to launch bar1
	_, err = file.WriteString("\t" + filepath.Join(openMMHome, "bar_omm.x") + " 1 " + arc1Path + " " + barPrm.temp + " " + arc2Path + " " + barPrm.temp + " > " + logPath + " \n")
	// end here document
	_, err = file.WriteString("END")

	// Check error for all above writes
	if err != nil {
		fmt.Println("failed to write to temporary file " + filePath)
		log.Fatal(err)
	}

	file.Close()

	return filePath
}

// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Bar 2
// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// AutoBAR2, managed by BARManager, runs BAR2 in all subdirectories provided in parallel on different cluster nodes
func (ng nodeGroup) autoBAR2(subDirs []string, genPrm *generalParameters, barPrm *barParameters, maxNodes int) {

	// Get nodes to run BAR on
	fmt.Print("\nLooking for " + strconv.Itoa(maxNodes) + " available nodes...")
	t1 := time.Now()
	updateStatus(&ng, genPrm.targetDirectory)
	t2 := time.Now()
	fmt.Print("found " + strconv.Itoa(len(ng.freeNodeIndices)) + " available nodes in " + t2.Sub(t1).String())
	if len(ng.freeNodeIndices) == 0 {
		err := errors.New("did not find enough free nodes to run on - exiting")
		log.Fatal(err)
	}

	// Create new wait group to determine when all goroutines have finished
	wg := sync.WaitGroup{}

	// iterate through all subDirs
	fmt.Println("\nBeginning AutoBAR2 run on " + strconv.Itoa(len(subDirs)) + " files...\n")
	for i := 0; i < len(subDirs); i++ {
		// Determine which node to run BAR1 on by iterating through the list of free nodes
		nodeIndex := ng.freeNodeIndices[i%maxNodes]
		// Add one to wait group
		wg.Add(1)
		// run BAR2 in parallel on node selected using go routines
		go ng.nodes[nodeIndex].BAR2(subDirs[i], genPrm, barPrm, &wg)

	}
	// Wait here for all goroutines to finish
	wg.Wait()
}

// BAR2, managed by AutoBAR2, runs BAR2 on files in subdirectory provided on node provided
func (n node) BAR2(subBarDir string, genPrm *generalParameters, barPrm *barParameters, wg *sync.WaitGroup) {

	// Get path to .bar file inside subBarDir
	barPath, err := getBAR2FilePath(subBarDir)
	if err != nil {
		log.Fatal(err)
	}

	// Get number of frames from .bar file
	frameCount := getNumFrames(barPath)

	// Write script to run bar2 and save to subBarDir
	tempFilePath := createTempBAR2Script(barPath, frameCount, genPrm, barPrm, &n)

	// Run that script
	out, err := exec.Command("sh", tempFilePath).CombinedOutput()
	// Report results to user
	if err != nil {
		fmt.Print("Error encountered on files in subdirectory " + subBarDir + " using node " + n.name)
		fmt.Println(err)

		// get filepath to write error to
		outFilePath := filepath.Join(subBarDir, "bar2.err")
		// create file to store err
		outFile, err := os.Create(outFilePath)
		if err != nil {
			fmt.Println("failed to create error log: " + outFilePath)
			log.Fatal(err)
		}
		// Set file permissions jic
		err = os.Chmod(outFilePath, octalPermissions)
		if err != nil {
			err = errors.New("failed to change temp file permissions for dynamic bash script")
			log.Fatal(err)
		}
		// Write output to file
		_, err = outFile.WriteString(string(out))
		fmt.Println()

	} else {
		fmt.Println("BAR2 finished successfully on files in subdirectory " + subBarDir + " using node " + n.name)
	}

	// subtract one from wg count
	wg.Done()
}

// Writes bash script to run BAR2 on node provided on BAR2 file provided
func createTempBAR2Script(barPath string, frameCount string, genPrm *generalParameters, barPrm *barParameters, n *node) string {

	scriptPath := filepath.Join(filepath.Dir(barPath), "bar2.sh")
	// Write temp bash script in dir
	tempFile, err := os.Create(scriptPath)
	if err != nil {
		fmt.Println("failed to create temporary BAR 2 bash script: " + scriptPath)
		log.Fatal(err)
	}

	// Set file permissions jic
	err = os.Chmod(scriptPath, octalPermissions)
	if err != nil {
		err = errors.New("failed to change temp file permissions for BAR 2 bash script")
		log.Fatal(err)
	}

	// get log path
	logPath := filepath.Join(filepath.Dir(barPath), "bar2.log")

	// Start with header
	_, err = tempFile.WriteString("#!/bin/bash\n")
	// Begin here document (all following command will be performed inside node)
	_, err = tempFile.WriteString("ssh -o \"StrictHostKeyChecking no\" " + n.name + " << END\n")
	// Source universally needed files
	_, err = tempFile.WriteString("\tsource " + genPrm.intelSource + "\n")
	// Source gpu dependant files
	var openMMHome string
	// Source CUDA files and get openMMHome variable
	if n.cardGeneration == "Ampere" {
		_, err = tempFile.WriteString("\tsource " + genPrm.cuda11Source + "\n")
		openMMHome = genPrm.cuda11Home
	} else if n.cardGeneration == "Turing" {
		_, err = tempFile.WriteString("\tsource " + genPrm.cuda10Source + "\n")
		openMMHome = genPrm.cuda10Home
	} else {
		err = errors.New("card generation unrecognized - unsure which files to source. Recognized generations are " +
			"\"Maxwell\", \"Pascal\", \"Turing\". Check entry of node \"" + n.name + "\" in node INI file")
		log.Fatal(err)
	}
	// Get card number
	_, err = tempFile.WriteString("\texport CUDA_VISIBLE_DEVICES=" + n.cardNumber + "\n")
	// Write command to launch bar2
	_, err = tempFile.WriteString("\t" + filepath.Join(openMMHome, "bar_omm.x") + " 2 " + barPath + " 1 " + frameCount + " " + barPrm.frameInterval +
		" 1 " + frameCount + " " + barPrm.frameInterval + " > " + logPath + " \n")
	// end here document
	_, err = tempFile.WriteString("END")

	// Check error for all above writes
	if err != nil {
		fmt.Println("failed to write to temporary file " + scriptPath)
		log.Fatal(err)
	}

	tempFile.Close()

	return scriptPath
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Helper functions
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// Retrieves number of frames from .bar files
func getNumFrames(barPath string) string {
	file, err := os.Open(barPath)
	if err != nil {
		fmt.Println("Failed to open bar file " + barPath)
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Scan()
	firstLine := scanner.Text()
	return strings.Fields(firstLine)[0]

}

// takes BAR subdirectory path, e.g. bar/vdw000ele000_vdw010ele000 and gets corresponding dynamic subdirectory paths,
// e.g. dynamic/vdw000ele000 and dynamic/vdw010ele000. Then returns paths to ARC files in those directories
func getBAR1FilePaths(directory string, subBarDir string) ([]string, error) {
	dynDir := filepath.Join(directory, "dynamic")

	var err error = nil

	// Get name of the bar directory
	barDirName := filepath.Base(subBarDir)
	// Get the names of the 2 dynamic directories corresponding to the bar directory
	dynDirNames := strings.Split(barDirName, "_")
	dynDirs := [2]string{filepath.Join(dynDir, dynDirNames[0]), filepath.Join(dynDir, dynDirNames[1])}
	// arcPaths will hold the paths to the 2 arc files
	arcPaths := make([]string, len(dynDirs))

	// for each of the 2 directories
	for i := 0; i < len(dynDirs); i++ {

		// get all files in dir
		fileInfo, err := ioutil.ReadDir(dynDirs[i])
		if err != nil {
			fmt.Println("failed to read directory " + dynDirs[i])
			log.Fatal(err)
		}

		// Initialize variables to track name and number of arc files in directory
		var arcName string
		numARC := 0

		// Iterate through all files in directory
		for i := 0; i < len(fileInfo); i++ {

			// get file ext
			fileExt := filepath.Ext(fileInfo[i].Name())
			// if ext = arc, save name and iterate counter
			if fileExt == ".arc" {
				arcName = fileInfo[i].Name()
				numARC++
			}
		}

		// Check for deficiencies with directory
		if numARC != 1 {
			err = errors.New("missing or multiple ARC file(s) in directory " + dynDirs[i])
		}
		// Get arc paths by adding arc name to directory name
		arcPaths[i] = filepath.Join(dynDirs[i], arcName)
	}
	// return arc paths
	return arcPaths, err
}

// get path to .bar file in directory
func getBAR2FilePath(subDir string) (string, error) {
	// read directory
	fileInfo, err := ioutil.ReadDir(subDir)
	if err != nil {
		fmt.Println("Failed to read directory " + subDir)
		log.Fatal(err)
	}
	// set var to track number of bar files
	numBAR := 0
	var barName string

	// iterate through all files
	for i := 0; i < len(fileInfo); i++ {
		// get file ext
		fileExt := strings.Split(fileInfo[i].Name(), ".")[1]
		// if ext = bar, save name and iterate counter
		if fileExt == "bar" {
			barName = fileInfo[i].Name()
			numBAR++
		}
	}

	// Check for deficiencies with directory
	if numBAR != 1 {
		err = errors.New("missing or multiple ARC file(s) in directory " + subDir)
	}
	// get path from name
	barPath := filepath.Join(subDir, barName)
	// return path
	return barPath, err
}

// Get subdirectories to run BAR inside
func getBARSubDirs(directory string) ([]string, error) {
	// get name of directory that hold bar subdirectories
	barDir := filepath.Join(directory, "bar")
	// read it
	fileInfo, err := ioutil.ReadDir(barDir)
	if err != nil {
		fmt.Println("Failed to read directory " + directory)
		log.Fatal(err)
	}
	// iterate through contents
	dirCounter := 0
	for i := 0; i < len(fileInfo); i++ {
		if fileInfo[i].IsDir() {
			dirCounter++
		} else {
			// flag error if any of the contents is a loose file
			err = errors.New("loose files in bar directory")
		}
	}
	// make slice to save subdirectory paths
	subDirs := make([]string, dirCounter)
	// save paths to slice
	for i := 0; i < len(fileInfo); i++ {
		subDirs[i] = filepath.Join(barDir, fileInfo[i].Name())
	}
	// return slice of paths
	return subDirs, err
}
