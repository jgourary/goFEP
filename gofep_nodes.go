package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)



// Check if individual node is free, set isFree field for that node
func updateNodeStatus(n *node, tempFilePath string, wg *sync.WaitGroup) {
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

	// Set to true by default
	n.isFree = true

	// split nvidia-smi output into lines
	lines := strings.Split(outString, "\n")

	// Set default value of line to containing the first process entry to the last line
	readFromLine := len(lines)-1
	// Try to find actual value of line containing first process entry. Ignore lines at beginning and end known to be too
	// early/late to be this line
	for i := 11; i < len(lines)-1; i++ {
		if strings.Count(lines[i],"PID") > 0 {
			readFromLine = i+2
		}
	}
	// Get GPU number of node as int to compare GPU number of process with
	cardNum, err := strconv.Atoi(n.cardNumber)
	if err != nil {
		fmt.Println("Warning: failed to convert cardNumber \"" + n.cardNumber + "\" of node " + n.name + " from string to int")
		fmt.Println("assuming node " + n.name + " is unavailable and continuing...")
		n.isFree = false
	}
	// Look for processes starting at line
	for i := readFromLine; i < len(lines)-1; i++ {
		tokens := strings.Fields(lines[i])
		// if line has a token in the position signifying GPU number...
		if len(tokens) > 1 {
			// convert that token to int
			processGPU, err := strconv.Atoi(tokens[1])
			// if no error occurred
			if err == nil {
				// check process gpu num against card num
				if processGPU == cardNum {
					// if equal node is busy
					n.isFree = false
				}
			}
		}
	}

	wg.Done()
}

type nodeGroup struct {
	freeNodeIndices []int
	nodes []node
}

// Write a shell script that checks node status and deposit it in chosen directory
func createTempNodeCheckScript(tempDir string) string {
	// mkdir if not exists
	err := os.MkdirAll(tempDir, octalPermissions)
	if err != nil {
		err = errors.New("failed to create temp directory:" + tempDir)
		log.Fatal(err)
	}
	// Write temp bash script in temp dir to check node status
	tempFilePath := filepath.Join(tempDir,nodeCheckScriptName)
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		fmt.Println("failed to create temporary file: " + tempFilePath)
		fmt.Println(err)
		log.Fatal(err)
	}
	// Set file permissions jic
	err = os.Chmod(tempFilePath, octalPermissions)
	if err != nil {
		err = errors.New("failed to change temp file permissions for BAR 2 bash script")
		log.Fatal(err)
	}
	// Write file contents
	_, err = tempFile.WriteString("#!/bin/bash\n")
	_, err = tempFile.WriteString("node=$1\n")
	_, err= tempFile.WriteString("ssh -o \"StrictHostKeyChecking no\" $node nvidia-smi\n")
	if err != nil {
		fmt.Println("failed to write to temporary file: " + tempFilePath)
		log.Fatal(err)
	}
	tempFile.Close()

	return tempFilePath
}

// Update isFree field for all nodes in array, return number of free nodes
func updateStatus(ng *nodeGroup, directory string) {

	tempDir := filepath.Join(directory, "temp")
	tempFilePath := createTempNodeCheckScript(tempDir)


	// Create new wait group to determine when all goroutines have finished
	wg := sync.WaitGroup{}

	// iterate through all nodes
	for i := 0; i < len(ng.nodes); i++ {
		// Add one to wait group
		wg.Add(1)
		// check if node is free, send node with updated "isFree" param to ch, subtracting 1 from wg in the process
		go updateNodeStatus(&ng.nodes[i],tempFilePath, &wg)
	}
	// Wait for all goroutines to finish, then close channel
	wg.Wait()

	// Update free node indices
	// Make int array to store free indices
	freeIndices := make([]int, len(ng.nodes))
	numFreeNodesFound := 0
	// iterate through all nodes in group
	for i := 0; i < len(ng.nodes); i++ {
		// if node is free
		if ng.nodes[i].isFree {
			// add node index to next available spot in freeIndices
			freeIndices[numFreeNodesFound] = i
			// iterate counter of free nodes found
			numFreeNodesFound++
		}
	}
	if numFreeNodesFound > 0 {
		ng.freeNodeIndices = freeIndices[0:numFreeNodesFound]
	} else {ng.freeNodeIndices = make([]int,0)}

}

func getNodeGroup(genPrm *generalParameters) nodeGroup {

	// If source prm path is not absolute already, redefine from target directory
	nodeIniPath,err := filepath.Abs(genPrm.nodeIniPath)
	if err != nil {
		fmt.Println("Could not compute absolute path to node INI file location \"" + nodeIniPath +"\" specified in general block of INI")
		log.Fatal(err)
	}

	// Open INI file
	file, err := os.Open(nodeIniPath)
	if err != nil {
		fmt.Println("failed to open node INI at: " + nodeIniPath)
		log.Fatal(err)
	}

	// Read file line by line and save nodes
	var ng nodeGroup
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// get next line
		line := scanner.Text()
		// check that line isn't empty before cleaning
		if len(line) > 0 {
			// clean line of comments
			cleanedLine := cleanLine(line)
			// Check that line isn't empty after cleaning (it will often be if entire line was a comment)
			if len(cleanedLine) > 0 {
				// split line into tokens by comma
				tokens := strings.Split(cleanedLine,",")

				// save parameters to a new node
				thisNode := node{}
				thisNode.name = tokens[0]
				thisNode.cardNumber = tokens[1]
				thisNode.cardManufacturer = tokens[2]
				thisNode.cardGeneration = tokens[3]
				thisNode.cardModel = tokens[4]
				thisNode.memory, err = strconv.Atoi(tokens[5])
				if err != nil {
					fmt.Println("Failed to convert memory entry " + tokens[5] + " from string to int while reading node INI at " + nodeIniPath)
					log.Fatal(err)
				}
				thisNode.performanceIndex, err = strconv.Atoi(tokens[6])
				if err != nil {
					fmt.Println("Failed to convert performance index entry " + tokens[6] + " from string to int while reading node INI at " + nodeIniPath)
					log.Fatal(err)
				}

				// append node to nodeGroup
				ng.nodes = append(ng.nodes, thisNode)
			}
		}
	}

	// Sort nodes before returning by desired criteria
	switch genPrm.nodePreference {
	case "fastest":
		// sort by highest to lowest performance
		sort.Slice(ng.nodes, func(i, j int) bool { return ng.nodes[i].performanceIndex > ng.nodes[j].performanceIndex })
	case "slowest":
		// sort by lowest to highest performance
		sort.Slice(ng.nodes, func(i, j int) bool { return ng.nodes[i].performanceIndex < ng.nodes[j].performanceIndex })
	case "memory":
		// sort by highest to lowest memory
		sort.Slice(ng.nodes, func(i, j int) bool { return ng.nodes[i].memory > ng.nodes[j].memory })
	case "random":
		// shuffle node order
		rand.Shuffle(len(ng.nodes), func(i, j int) { ng.nodes[i], ng.nodes[j] = ng.nodes[j], ng.nodes[i] })
	default:
		// do nothing
	}
	return ng
}

type node struct {
	name string
	cardNumber string
	cardManufacturer string
	cardGeneration string
	cardModel string
	memory int
	performanceIndex int

	isFree bool
}




