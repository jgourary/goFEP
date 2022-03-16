package main

import (
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

// Called from main, manages overall process of running dynamic on multiple files with multiple parameter sets for multiple iterations
func (ng nodeGroup) DynamicManager(genPrm *generalParameters, dynPrm []dynamicParameters, maxNodes int) {

	start := time.Now()

	// Get subdirectories to run dynamic inside
	dynDirectory := filepath.Join(genPrm.targetDirectory, "dynamic")

	// Run autoDynamic using all dynamic parameter sets in order
	for i := 0; i < len(dynPrm); i++ {
		thisDynPrm := dynPrm[i]
		fmt.Println("\nPreparing to run AutoDynamic with parameter set " + dynPrm[i].name + " for " + strconv.Itoa(thisDynPrm.repetitions) + " repetition(s)...")
		for repNum := 0; repNum < thisDynPrm.repetitions; repNum++ {

			subDirs := getValidDynDirs(dynDirectory, &thisDynPrm, repNum)
			if len(subDirs) > 0 {
				fmt.Println("\n\nBeginning AutoDynamic repetition #" + strconv.Itoa(repNum+1) + " with parameter set \"" + dynPrm[i].name + "\"...\n")
				t1 := time.Now()
				ng.autoDynamic(genPrm, &dynPrm[i], subDirs, maxNodes, repNum) // j is repetition number w/ this param set
				t2 := time.Now()
				fmt.Println("\nAutoDynamic repetition #" + strconv.Itoa(repNum+1) + " with parameter set " + dynPrm[i].name + " finished in " + t2.Sub(t1).String())
			} else {
				fmt.Println("\nSkipping AutoDynamic repetition #" + strconv.Itoa(repNum+1) + " with parameter set \"" + dynPrm[i].name + "\": this repetition is already complete for all subdirectories")
			}

		}
	}

	end := time.Now()
	fmt.Println("\nAll AutoDynamic runs complete in " + end.Sub(start).String())
	fmt.Println()

}

// Called by dynamicManager, runs dynamic on multiple files for multiple iterations with ONE parameter set
func (ng nodeGroup) autoDynamic(genPrm *generalParameters, dynPrm *dynamicParameters, subDirs []string, maxNodes int, repetitionNum int) {

	// Get nodes to run dynamic on
	fmt.Print("\nLooking for " + strconv.Itoa(maxNodes) + " available nodes...")
	t1 := time.Now()
	updateStatus(&ng, genPrm.targetDirectory)
	t2 := time.Now()
	fmt.Print("found " + strconv.Itoa(len(ng.freeNodeIndices)) + " available nodes in " + t2.Sub(t1).String())
	if len(ng.freeNodeIndices) == 0 {
		err := errors.New("did not find enough free nodes to run on - exiting")
		log.Fatal(err)
	}

	fmt.Println("\nBeginning AutoDynamic run on " + strconv.Itoa(len(subDirs)) + " files...\n")

	// Create new wait group to determine when all goroutines have finished
	wg := sync.WaitGroup{}

	// iterate through all subDirs
	for i := 0; i < len(subDirs); i++ {
		// get index of node to run dynamic on by iterating through freeNodeIndices w/ constraint of not exceeding maxNodes
		nodeIndex := ng.freeNodeIndices[i%maxNodes]
		// Add one to wait group
		wg.Add(1)
		// Create go routine to run dynamic in that subDir using that node
		go ng.nodes[nodeIndex].dynamic(genPrm, dynPrm, subDirs[i], repetitionNum, &wg)

	}

	wg.Wait()

}

func (n node) dynamic(genPrm *generalParameters, dynPrm *dynamicParameters, subDir string, repetitionNum int, wg *sync.WaitGroup) {

	// Get name of xyz and key in the directory and delete previous log/arc/dyn if unneeded
	xyzPath, keyPath := getDynamicFilePaths(subDir)

	// Create bash script to run dynamic
	repetitionNumStr := strconv.Itoa(repetitionNum)
	tempDynScriptPath := createTempDynamicScript(subDir, xyzPath, keyPath, genPrm, dynPrm, &n, repetitionNumStr)

	// run newly created shell script
	out, err := exec.Command("sh", tempDynScriptPath).CombinedOutput()
	if err != nil {
		fmt.Print("Error encountered on file in subdirectory " + filepath.Dir(xyzPath) + " using node " + n.name + ": ")
		fmt.Println(err)

		// get filepath to write error to
		outFilePath := filepath.Join(subDir, dynPrm.name+repetitionNumStr+".err")
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
		fmt.Println("Dynamic finished on file in subdirectory " + filepath.Dir(xyzPath) + " using node " + n.name)
	}

	// subtract one from wg count
	wg.Done()

}

// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Helper functions
// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func getValidDynDirs(dynDirectory string, dynPrm *dynamicParameters, repNum int) []string {

	// Read in all files in dir
	fileInfo, err := ioutil.ReadDir(dynDirectory)
	if err != nil {
		fmt.Println("failed to read directory: " + dynDirectory)
		log.Fatal(err)
	}
	// Check that all files are directories and save to an array
	numValidSubDirs := 0
	validSubDirs := make([]string, len(fileInfo))

	// Iterate through all items in directory
	for i := 0; i < len(fileInfo); i++ {
		// if item is a Dir (as it should be unless the end user tampered with the directory manually...)
		if fileInfo[i].IsDir() {
			// Calculate path to log file that would exist if this combination of directory / dynamic param set / iteration num had been run
			logPath := filepath.Join(dynDirectory, fileInfo[i].Name(), dynPrm.name+"_"+strconv.Itoa(repNum)+".log")
			// see if log file exists
			_, err = os.Stat(logPath)
			// if said log file doesn't exist (error is non-nil)
			if err != nil {
				// add directory to list of approved directories
				validSubDirs[numValidSubDirs] = filepath.Join(dynDirectory, fileInfo[i].Name())
				numValidSubDirs++

			}
		}

	}
	return validSubDirs[0:numValidSubDirs]
}

func createTempDynamicScript(subDir string, xyzPath string, keyPath string, genPrm *generalParameters, dynPrm *dynamicParameters, n *node, repetitionNum string) string {
	scriptName := dynPrm.name + "_" + repetitionNum + ".sh"
	// Write temp bash script in current dir to check node status
	tempFilePath := filepath.Join(subDir, scriptName)
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		fmt.Println("failed to create temporary dynamic bash script: " + tempFilePath)
		log.Fatal(err)
	}

	// Set file permissions jic
	err = os.Chmod(tempFilePath, octalPermissions)
	if err != nil {
		err = errors.New("failed to change temp file permissions for dynamic bash script")
		log.Fatal(err)
	}

	// get log path
	logPath := filepath.Join(filepath.Dir(xyzPath), dynPrm.name+"_"+repetitionNum+".log")

	// Write file contents

	// Start with header
	_, err = tempFile.WriteString("#!/bin/bash\n")
	// Begin here document (all following command will be performed inside node)
	_, err = tempFile.WriteString("ssh -o \"StrictHostKeyChecking no\" " + n.name + " << END\n")
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
	// Write command to launch dynamic
	_, err = tempFile.WriteString(getDynamicLaunchCommand(openMMHome, xyzPath, keyPath, logPath, dynPrm))
	// end here document
	_, err = tempFile.WriteString("END")

	// Check error for all above writes
	if err != nil {
		fmt.Println("failed to write to temporary file " + tempFilePath)
		log.Fatal(err)
	}

	tempFile.Close()

	return tempFilePath
}

// Get ensemble dependent command to launch tinker dynamic
func getDynamicLaunchCommand(openMMHome string, xyzPath string, keyPath string, logPath string, dynPrm *dynamicParameters) string {
	basecmd := "\t" + filepath.Join(openMMHome, "dynamic_omm.x") + " " + xyzPath + " -k " + keyPath + " " + dynPrm.numSteps +
		" " + dynPrm.stepInterval + " " + dynPrm.saveInterval + " " + dynPrm.ensemble
	var cmd string
	switch dynPrm.ensemble {
	case "1":
		cmd = basecmd + " N > " + logPath + " \n"
	case "2":
		cmd = basecmd + " " + dynPrm.temp + " N > " + logPath + " \n"
	case "3":
		cmd = basecmd + " " + dynPrm.pressure + " N > " + logPath + " \n"
	case "4":
		cmd = basecmd + " " + dynPrm.temp + " " + dynPrm.pressure + " N > " + logPath + " \n"
	}

	return cmd
}

// Get xyz and key names from subdirectory and check that there aren't any issues with them
// If arc, dyn files exist, set correct permissions
func getDynamicFilePaths(subDir string) (string, string) {
	fileInfo, err := ioutil.ReadDir(subDir)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize variables to track name and number of xyz and key files in directory
	var xyzPath string
	numXYZ := 0
	var keyPath string
	numKEY := 0
	var arcPath string
	var dynPath string

	// Iterate through all files in directory
	for i := 0; i < len(fileInfo); i++ {
		// get file ext
		fileExt := strings.Split(fileInfo[i].Name(), ".")[1]
		if fileExt == "xyz" {
			xyzPath = filepath.Join(subDir, fileInfo[i].Name())
			numXYZ++
		} else if fileExt == "key" {
			keyPath = filepath.Join(subDir, fileInfo[i].Name())
			numKEY++
		} else if fileExt == "arc" {
			arcPath = filepath.Join(subDir, fileInfo[i].Name())
			err = os.Chmod(arcPath, octalPermissions)
			if err != nil {
				fmt.Println("Failed to update file permissions for arc file: " + arcPath)
				log.Fatal(err)
			}
		} else if fileExt == "dyn" {
			dynPath = filepath.Join(subDir, fileInfo[i].Name())
			err = os.Chmod(dynPath, octalPermissions)
			if err != nil {
				fmt.Println("Failed to update permissions for dyn file: " + dynPath)
				log.Fatal(err)
			}
		}
	}

	// Check for deficiencies with directory
	if numXYZ != 1 || numKEY != 1 {
		err = errors.New("missing or multiple XYZ or KEY file(s) in directory " + subDir)
		log.Fatal(err)
	}

	return xyzPath, keyPath
}

/*// remove log, arc, dyn files from a directory
func removeDynamicOutputFiles(dir string) {
	// read directory
	fileInfo, err := ioutil.ReadDir(dir)
	if err != nil {
		fmt.Println("Failed to read directory to remove arc/log/dyn files from")
		log.Fatal(err)
	}
	// iterate through all files
	for i := 0; i < len(fileInfo); i++ {
		// get file extension
		fileExt := strings.Split(fileInfo[i].Name(), ".")[1]
		// if ext designates it as a dynamic output file...
		if fileExt == "arc" || fileExt == "dyn" || fileExt == "log" {
			// ...delete it
			filePath := filepath.Join(dir, fileInfo[i].Name())
			err = os.Remove(filePath)
			if err != nil {
				fmt.Println("Failed to remove arc/dyn/log files from directory: " + dir)
				log.Fatal(err)
			}
		}
	}
}*/

// Currently non-functional code to automatically generate scripts to kill all dynamic jobs

/*func writeKillAllScript(genPrm *generalParameters, dynPrm *dynamicParameters, nodeIndices []int, iteration int, ng *nodeGroup) {
	scriptPath := filepath.Join(genPrm.targetDirectory,"temp",dynPrm.name + "_" + strconv.Itoa(iteration) + "_abortAll.sh")
	f, err := os.Create(scriptPath)
	if err != nil {
		fmt.Println("Failed to create kill all script")
		log.Fatal(err)
	}
	err = os.Chmod(scriptPath, octalPermissions)
	if err != nil {
		fmt.Println("Failed to change permissions of kill all script")
		log.Fatal(err)
	}


	// create struct to store set of structs associating nodes and their running gpu processes
	var proc processTrackerHolder
	proc.processTrackers = make([]processTracker, len(nodeIndices))

	wg2 := sync.WaitGroup{}
	// get pids
	for i, nodeIndex := range nodeIndices {
		wg2.Add(1)
		go getPIDs(&proc, &ng.nodes[nodeIndex], genPrm, i, &wg2)
		proc.processTrackers[i].nodeName = ng.nodes[nodeIndex].name
	}
	wg2.Wait()

	// Start with header
	_, err = f.WriteString("#!/bin/bash")

	for _, processTracker := range proc.processTrackers {
		// Begin here document (all following command will be performed inside node)
		_, err = f.WriteString("ssh -o \"StrictHostKeyChecking no\" " + processTracker.nodeName + " << END\n")

		for _, pid := range processTracker.pids {
			_, err = f.WriteString("kill " + pid + "\n")
		}
		_, err = f.WriteString("echo \"Process(es) on " + processTracker.nodeName + " killed\" \n")

		// end here document
		_, err = f.WriteString("END\n\n")
	}
}

type processTrackerHolder struct {
	processTrackers []processTracker
}

type processTracker struct {
	nodeName string
	pids []string
}

// Get tinker PID(s) on node
func getPIDs(proc *processTrackerHolder,n *node, genPrm *generalParameters, i int, wg2 *sync.WaitGroup) {
	var pids []string

	tempFilePath := filepath.Join(genPrm.targetDirectory,"temp",nodeCheckScriptName)
	// ssh into node, run "nvidia-smi", and capture output
	out, err := exec.Command("sh", tempFilePath, n.name).CombinedOutput()
	if err != nil {
		fmt.Print("error encountered on " + n.name + ": ")
		fmt.Print(err)
		fmt.Println("\nout = " + string(out))
		fmt.Println("The most likely cause of this error is that you did not ssh into bme-nova before running gofep, " +
			"but it is also possible the node is down. Check node manually to verify.")
	}
	outString := string(out)
	tokens := strings.Fields(outString)
	for i,token := range tokens {
		if strings.Count(token, "dynamic_omm") > 0 {
			pids = append(pids,tokens[i-2])
		}

	}
	proc.processTrackers[i].pids = pids

	wg2.Done()
}*/
