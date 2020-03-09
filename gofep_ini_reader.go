package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Read INI file and return FEP params
func getParams(iniPath string) (generalParameters, setupParameters, []dynamicParameters, barParameters) {

	// Get lines of INI with comment blocks and empty lines removed
	lines, err := getSanitizedINIData(iniPath)
	if err != nil {
		fmt.Println("failed to clean comments from ini file: " + iniPath)
		log.Fatal(err)
	}

	// Identify blocks (brace enclosed sections) in lines
	blocks := getBlocks(lines)

	// Initialize param variables
	var setupPrm setupParameters
	var barPrm barParameters
	var genPrm generalParameters
	dynPrm := make([]dynamicParameters, len(blocks)-3)
	dynPrmCounter := 0

	// Set counters to make sure duplicate blocks are not defined (duplicate dynamic blocks are allowed)
	genPrmCounter := 0
	setupPrmCounter := 0
	barPrmCounter := 0

	// iterate over all blocks
	for _, b := range blocks {

		// Identify parameters in block
		paramsMap := b.generateParamsMap()
		// Based on blocktype, turn parameters map into the relevant parameter struct
		if b.blockType == "general" {
			genPrm = generateGenParams(paramsMap)
			genPrmCounter++
		} else if b.blockType == "setup" {
			setupPrm = generateSetupParams(paramsMap)
			setupPrmCounter++
		} else if b.blockType == "dynamic" {
			dynPrm[dynPrmCounter] = generateDynamicParams(paramsMap)
			dynPrmCounter++
		} else if b.blockType == "bar" {
			barPrm = generateBARParams(paramsMap)
			barPrmCounter++
		} else {
			err := errors.New("unrecognized keyword in INI file: " + b.blockType)
			log.Fatal(err)
		}

	}

	// Make sure correct number of blocks are defined
	if genPrmCounter != 1 {
		err = errors.New("missing \"bar\" block or multiple \"general\" blocks defined in INI file")
		log.Fatal(err)
	} else if setupPrmCounter != 1 {
		err = errors.New("missing \"setup\" block or multiple \"setup\" blocks defined in INI file")
		log.Fatal(err)
	} else if barPrmCounter != 1 {
		err = errors.New("missing \"bar\" block or multiple \"bar\" blocks defined in INI file")
		log.Fatal(err)
	}

	// Before returning, sort dynamic parameter sets by order field
	less := func(i, j int) bool {
		return dynPrm[i].order < dynPrm[j].order
	}
	sort.Slice(dynPrm, less)

	// Check that no 2 dynamic blocks have the same order or name
	for i, prm1 := range dynPrm {
		for j, prm2 := range dynPrm {
			if i != j {
				if prm1.order == prm2.order {
					fmt.Println("Dynamic blocks " + prm1.name + " and " + prm2.name + " have the same value for \"order\"")
					err = errors.New("no two \"dynamic\" blocks can have the same \"order\" parameter")
					log.Fatal(err)
				} else if prm1.name == prm2.name {
					fmt.Println("Dynamic blocks " + prm1.name + " and " + prm2.name + " have the same value for \"name\"")
					err = errors.New("no two \"dynamic\" blocks can have the same \"name\" parameter")
					log.Fatal(err)
				}
			}
		}
	}

	// return parameter structs
	return genPrm, setupPrm, dynPrm, barPrm
}

// Removes comment blocks and empty lines from a file - only suitable for short files!
func getSanitizedINIData(path string) ([]string, error) {
	// open file
	file, err := os.Open(path)
	if err != nil {
		fmt.Println("failed to open INI file: " + path)
		log.Fatal(err)
	}
	defer file.Close()

	// read file line by line
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// remove comment blocks from line
		line := cleanLine(scanner.Text())
		// if line has content after cleaning, append to lines to return
		if len(line) > 0 { lines = append(lines, line) }
	}
	return lines, scanner.Err()
}

// removes comment blocks from a line (string)
func cleanLine(line string) string {
	// Look out for following byte signifying a comment
	commentByte := []byte("#")[0]
	// iterate through all bytes in line
	for i := 0; i < len(line); i++ {
		// if byte = comment byte, return line up until that byte
		if line[i] == commentByte {
			if i > 1 { return line[0:i] } else { return "" }
		}
	}
	// if no comments found, return whole line
	return line
}

// get all brace enclosed blocks in cleaned ini
func getBlocks(lines []string) []block {
	// valid block literals
	blockLiterals := []string {"general", "setup", "dynamic", "bar"}

	// Find all instances of these block literals at the beginning of lines and save the line numbers they were seen at
	var dividers []int
	for i := 0; i < len(lines); i++ {
		tokens := strings.Fields(lines[i])

		for _, literal := range blockLiterals {
			if tokens[0] == literal {
				dividers = append(dividers, i)
				break
			}
		}
	}

	// make a block struct for each divider
	blocks := make([]block, len(dividers))
	// for each block, copy data between dividers into the struct
	for i := 0 ; i < len(dividers); i++ {
		blocks[i].blockType = strings.Fields(lines[dividers[i]])[0]
		startSlice := dividers[i]+1
		if i < len(dividers) - 1 {
			endSlice := dividers[i+1] - 1
			blocks[i].lines = lines[startSlice:endSlice]
		} else {blocks[i].lines = lines[startSlice:] }

	}
	return blocks
}

// Get map of parameters from lines bounded by braces
func (b block) generateParamsMap() map[string][]string {
	paramsMap := map[string][]string{}
	for _, line := range b.lines {
		// split line into tokens by whitespace
		tokens := strings.Fields(line)
		// get number of tokens
		numTokens := len(tokens)
		// Create string slice to store value of key-value pair
		key := tokens[0]
		value := tokens[1:numTokens]
		paramsMap[key] = value
	}
	return paramsMap
}

// Generate parameters struct from parameters map
func generateSetupParams(paramsMap map[string][]string) setupParameters {
	var err error
	prm := setupParameters{}

	prm.vdw = paramsMap["vdwLambdas"]
	prm.ele = paramsMap["eleLambdas"]
	prm.rst = paramsMap["restraints"]

	if len(prm.rst) > 0 {
		if len(prm.vdw) != len(prm.ele) || len(prm.ele) != len(prm.rst) {
			fmt.Println("length of \"vdwLambdas\" = " + strconv.Itoa(len(prm.vdw)))
			fmt.Println("length of \"eleLambdas\" = " + strconv.Itoa(len(prm.ele)))
			fmt.Println("length of \"restraints\" = " + strconv.Itoa(len(prm.rst)))
			err = errors.New("error: vdwLambdas, eleLambdas, restraints specified in setup block of INI file are of unequal lengths. Please fix this and run setup again")
			log.Fatal(err)
		}
	} else {
		if len(prm.vdw) != len(prm.ele) {
			fmt.Println("length of \"vdwLambdas\" = " + strconv.Itoa(len(prm.vdw)))
			fmt.Println("length of \"eleLambdas\" = " + strconv.Itoa(len(prm.ele)))
			err = errors.New("error: vdwLambdas, eleLambdas specified in setup block of INI file are of unequal lengths. Please fix this and run setup again")
			log.Fatal(err)
		}
	}

	return prm
}

// Generate parameters struct from parameters map
func generateDynamicParams(paramsMap map[string][]string) dynamicParameters {
	var err error
	prm := dynamicParameters{}

	// Check if parameters were specified. If not, raise fatal error
	listOfKeys := []string {"name", "order", "repetitions", "ensemble", "stepInterval", "saveInterval", "simulationTime"}
	checkIfParamsSpecified(listOfKeys, paramsMap)

	// Set block name
	prm.name = paramsMap["name"][0]
	if len(prm.name) < 1 {
		err = errors.New("dynamic block names must be at least one character in length")
		log.Fatal(err)
	}

	// Set block order
	prm.order, err = strconv.Atoi(paramsMap["order"][0])
	if err != nil {
		fmt.Println("Failed to convert \"order\" parameter in \"dynamic\" block from string to integer")
		log.Fatal(err)
	}

	// Set block repetitions
	prm.repetitions, err = strconv.Atoi(paramsMap["repetitions"][0])
	if err != nil {
		fmt.Println("Failed to convert \"repetitions\" parameter in \"dynamic\" block from string to integer")
		log.Fatal(err)
	}

	// Set block ensemble
	prm.ensemble = paramsMap["ensemble"][0]
	if prm.ensemble != "1" && prm.ensemble != "2" && prm.ensemble != "3" && prm.ensemble != "4" {
		err = errors.New("ensemble in dynamic block \"" +prm.name + " must be set to \"1\", \"2\", \"3\", or \"4\"")
		log.Fatal(err)
	}

	// Set block temp
	if prm.ensemble == "2" || prm.ensemble == "4" {
		if len(paramsMap["temp"]) < 1 {
			err = errors.New("parameter \"temp\" in block \"dynamic\" has not been set")
			log.Fatal(err)
		}
		prm.temp = paramsMap["temp"][0]
		_, err := strconv.ParseFloat(prm.temp,64)
		if err != nil {
			fmt.Println("Temperature parameter set in dynamic block: \"" + prm.name + "\" of \"" + prm.temp + "\" could not be parsed as float and thus Tinker would likely fail")
			log.Fatal(err)
		}
	}

	// Set block pressure
	if prm.ensemble == "3" || prm.ensemble == "4" {
		if len(paramsMap["pressure"]) < 1 {
			err = errors.New("parameter \"pressure\" in block \"dynamic\" has not been set")
			log.Fatal(err)
		}
		prm.pressure = paramsMap["pressure"][0]
		_, err := strconv.ParseFloat(prm.pressure,64)
		if err != nil {
			fmt.Println("Pressure parameter set in dynamic block: \"" + prm.name + "\" of \"" + prm.pressure + "\" could not be parsed as float and thus Tinker would likely fail")
			log.Fatal(err)
		}
	}

	// Set block step and save interval
	prm.stepInterval = paramsMap["stepInterval"][0]
	prm.saveInterval = paramsMap["saveInterval"][0]
	floatParams := [...]string{prm.stepInterval, prm.saveInterval}
	for _, param := range floatParams {
		_, err := strconv.ParseFloat(param, 64)
		if err != nil {
			fmt.Println("Temperature parameter set in bar block: \"" + prm.temp + "\" could not be parsed as float and thus Tinker would likely fail")
			log.Fatal(err)
		}
	}

	// Set block number of steps
	simTime, err := strconv.ParseFloat(paramsMap["simulationTime"][0], 64)
	if err != nil {
		fmt.Println("Failed to convert \"simulationTime\" parameter in \"dynamic\" block from string to float")
		log.Fatal(err)
	}
	stepInt, err := strconv.ParseFloat(prm.stepInterval, 64)
	if err != nil {
		fmt.Println("Failed to convert \"stepInterval\" parameter in \"dynamic\" block from string to float")
		log.Fatal(err)
	}
	prm.numSteps = strconv.Itoa(int(1e6 * simTime / stepInt + 0.5))

	return prm
}

// Generate parameters struct from parameters map
func generateBARParams(paramsMap map[string][]string) barParameters {
	var err error
	prm := barParameters{}

	// Check if other parameters were specified. If not, raise fatal error
	listOfKeys := []string {"frameInterval", "temp"}
	checkIfParamsSpecified(listOfKeys, paramsMap)

	// Set frame interval
	if len(paramsMap["frameInterval"]) < 1 {
		err = errors.New("parameter \"frameInterval\" in block \"bar\" has not been set")
		log.Fatal(err)
	}
	prm.frameInterval = paramsMap["frameInterval"][0]

	// set temp
	if len(paramsMap["temp"]) < 1 {
		err = errors.New("parameter \"temp\" in block \"bar\" has not been set")
		log.Fatal(err)
	}
	prm.temp = paramsMap["temp"][0]

	_, err = strconv.ParseFloat(prm.temp,64)
	if err != nil {
		fmt.Println("Temperature parameter set in bar block: \"" + prm.temp +"\" could not be parsed as float and thus Tinker would likely fail")
		log.Fatal(err)
	}

	_, err = strconv.Atoi(prm.frameInterval)
	if err != nil {
		fmt.Println("Frame interval parameter set in bar block: \"" + prm.frameInterval +"\" could not be parsed as int and thus Tinker would likely fail")
		log.Fatal(err)
	}


	return prm
}

// Check that all necessary parameters were specified in the INI
func checkIfParamsSpecified(listOfKeys []string, paramsMap map[string][]string) {
	var err error
	for i := 0; i < len(listOfKeys); i++ {
		if len(paramsMap[listOfKeys[i]]) < 1 {
			err = errors.New("parameter " + listOfKeys[i] + " has not been set")
			log.Fatal(err)
		}
	}
}

// Generate params data type from params map
func generateGenParams(paramsMap map[string][]string) generalParameters {
	var err error
	prm := generalParameters{}
	// Check if targetDirectory was specified
	_, ok := paramsMap["targetDirectory"]
	if ok {
		// Check if other parameters were specified. If not, raise fatal error
		listOfKeys := []string {"xyz", "key", "prm", "nodeINI"}
		checkIfParamsSpecified(listOfKeys, paramsMap)

		// prm.targetDirectory is specified - xyz/key/prm paths are assumed to be absolute paths
		prm.targetDirectory = paramsMap["targetDirectory"][0]
		prm.xyzPath = paramsMap["xyz"][0]
		prm.keyPath = paramsMap["key"][0]
		prm.prmPath = paramsMap["prm"][0]
		prm.nodeIniPath = paramsMap["nodeINI"][0]
	} else {
		// Check if other parameters were specified. If not, raise fatal error
		listOfKeys := []string {"xyz", "key", "prm", "nodeINI"}
		checkIfParamsSpecified(listOfKeys, paramsMap)

		// prm.targetDirectory is not specified - assumed to be current working directory - xyz/key/prm paths are
		// assumed to be relative to CWD
		prm.targetDirectory, err = os.Getwd()
		if err != nil {
			fmt.Println("Error while setting target directory to current working directory")
			log.Fatal(err)
		}
		prm.xyzPath = filepath.Join(prm.targetDirectory,paramsMap["xyz"][0])
		prm.keyPath = filepath.Join(prm.targetDirectory,paramsMap["key"][0])
		prm.prmPath = filepath.Join(prm.targetDirectory,paramsMap["prm"][0])
		prm.nodeIniPath = filepath.Join(prm.targetDirectory,paramsMap["nodeINI"][0])
	}

	// Check if other parameters were specified. If not, raise fatal error
	listOfKeys := []string {"nodePreference", "intelSource", "cuda8Source", "cuda10Source", "cuda8Home", "cuda10Home"}
	checkIfParamsSpecified(listOfKeys, paramsMap)

	prm.nodePreference = paramsMap["nodePreference"][0]

	prm.intelSource = paramsMap["intelSource"][0]
	prm.cuda8Source = paramsMap["cuda8Source"][0]
	prm.cuda10Source= paramsMap["cuda10Source"][0]
	prm.cuda8Home = paramsMap["cuda8Home"][0]
	prm.cuda10Home = paramsMap["cuda10Home"][0]

	// Check files specified really exist
	var files = [...]string {prm.keyPath, prm.xyzPath, prm.prmPath, prm.nodeIniPath, prm.intelSource,
		prm.cuda8Home, prm.cuda8Source, prm.cuda10Home, prm.cuda10Source}
	for _, file := range files {
		fileExists, err := pathExists(file)
		if err != nil {
			fmt.Println("Error while verifying existence of file \"" + file + "\"")
			log.Fatal(err)
		} else if fileExists == false {
			err = errors.New("file specified in INI \"" + file + "\" does not exist")
			log.Fatal(err)
		}
	}

	return prm
}

// exists returns whether the given file or directory exists
func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	// file exists and no error
	if err == nil { return true, nil }
	// file does not exist and no error
	if os.IsNotExist(err) { return false, nil }
	// file exists and error
	return true, err
}

// Contains fields for parameters relevant to multiple steps
type generalParameters struct {
	targetDirectory string
	xyzPath string
	keyPath string
	prmPath string
	nodeIniPath string
	nodePreference string
	intelSource string
	cuda8Source string
	cuda8Home string
	cuda10Source string
	cuda10Home string
}
// Contains fields for parameters relevant to gofep_dynamic_setup
type setupParameters struct {
	vdw []string
	ele []string
	rst []string
}

// Contains fields for parameters relevant to gofep_dynamic
type dynamicParameters struct {
	name string
	order int
	repetitions int
	stepInterval string
	saveInterval string
	numSteps string
	ensemble string
	temp string
	pressure string
}

// Contains fields for parameters relevant to gofep_bar
type barParameters struct {
	temp string
	frameInterval string
}

// Derived from a brace enclosed section of the ini file
type block struct {
	// type: setup, bar, or dynamic
	blockType string
	// lines of ini inside braces
	lines []string
}


