package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const goFEPversion string = "1.1"
const goFEPPlatform string = "Linux"
const goFEPArch string = "amd64"
const goFEPcompileDate string = "February 26, 2020"
const octalPermissions os.FileMode = 0777

// gofep_results.go constants
const resultFileName string = "bar2.log"
const finalResultFileName string = "results.txt"
const nodeCheckScriptName string = "run_nvidia_smi.sh"

// Main function - entry point for command line interface
func main() {
	var err error

	// Get cmd line arguments without program name
	args := os.Args
	argsLen := len(args)

	switch argsLen {
	case 0:
		help()
	case 1:
		help()
	case 2:
		fmt.Println("Too few arguments have been provided to run goFEP - launching built-in help function...")
		help()
	default:

		// args[1] == ini file
		iniPath := args[1]
		iniPath, err = filepath.Abs(iniPath)
		if err != nil {
			fmt.Println("Failed to compute absolute path to INI file at " + iniPath)
			log.Fatal(err)
		}
		// Get parameter sets from ini file
		genPrm, setupPrm, dynPrm, barPrm := getParams(iniPath)
		// Set application working directory to target directory (relevant for interpreting relative paths in genPrm)
		err := os.Chdir(genPrm.targetDirectory)
		if err != nil {
			fmt.Println("Failed to move working directory to: " + genPrm.targetDirectory)
			log.Fatal(err)
		}

		// initialize vars
		var numNodes int

		// Args[2] = call
		switch args[2] {

		case "setup":
			// run dynamic setup
			DynamicSetup(genPrm, setupPrm)

		case "dynamic":

			// Check that enough arguments are provided
			if argsLen < 4 {
				err = errors.New("too few arguments have been provided to run goFEP dynamic. 4 arguments are required.\n " +
					"If more assistance is needed with this issue, launch goFEP with no arguments to access built-in help function")
				log.Fatal(err)
			}

			// Get number of nodes to run on from 4th argument
			numNodes, err = strconv.Atoi(os.Args[4])
			if err != nil {
				err = errors.New("Invalid argument \"" + os.Args[4] + "\" for number of nodes to run on")
				log.Fatal(err)
			}
			// If num nodes set to -1 (auto), set it to number of files to run
			if numNodes == -1 {
				numNodes = len(setupPrm.vdw)
			}
			// Get nodes from node INI
			ng := getNodeGroup(&genPrm)


			// args[3] = call type
			switch args[3] {
			// run dynamic
			case "all":
				ng.DynamicManager(&genPrm, dynPrm, numNodes)
			case "new":
				ng.DynamicManager(&genPrm, dynPrm, numNodes)
			default:
				err = errors.New("invalid argument \"" + args[3] + "\". Valid arguments following \"dynamic\" are: \"new\", \"all\".\n " +
					"If more assistance is needed with this issue, launch goFEP with no arguments to access built-in help function")
				log.Fatal(err)
			}

		case "bar":
			// Check that enough arguments were provided
			if argsLen < 4 {
				err = errors.New("too few arguments have been provided to run goFEP bar. 3 arguments are required.\n " +
					"If more assistance is needed with this issue, launch goFEP with no arguments to access built-in help function")
				log.Fatal(err)
			}
			numNodes, err = strconv.Atoi(os.Args[3])
			if err != nil {
				err = errors.New("Invalid argument \"" + os.Args[3] + "\" for number of nodes to run on")
				log.Fatal(err)
			}
			// If num nodes set to -1 (auto), set it to number of files to run (bar has 1 fewer file than dynamic, hence -1)
			if numNodes == -1 {
				numNodes = len(setupPrm.vdw)-1
			}

			// Setup BAR folders
			BARSetup(&genPrm)
			// Get nodes from node INI
			ng := getNodeGroup(&genPrm)
			// Run BAR
			ng.BARManager(&genPrm, &barPrm, numNodes)
			// Get results
			returnResults(&genPrm)

		case "auto":

			// Check that enough arguments were provided
			if argsLen < 4 {
				err = errors.New("too few arguments have been provided to run goFEP auto. 3 arguments are required.\n " +
					"If more assistance is needed with this issue, launch goFEP with no arguments to access built-in help function")
				log.Fatal(err)
			}

			// Get number of nodes to run on
			numNodes, err = strconv.Atoi(os.Args[3])
			if err != nil {
				err = errors.New("Invalid argument \"" + os.Args[3] + "\" for number of nodes to run on")
				log.Fatal(err)
			}

			// Get nodes from node INI
			ng := getNodeGroup(&genPrm)
			// If num nodes set to -1 (auto), set it to number of files to run (bar has 1 fewer file than dynamic, hence -1)
			if numNodes == -1 {
				numNodes = len(setupPrm.vdw)
			}
			// Setup for dynamic
			DynamicSetup(genPrm, setupPrm)
			// Run dynamic
			ng.DynamicManager(&genPrm, dynPrm, numNodes)

			// If num nodes set to -1 (auto), set it to number of files to run (bar has 1 fewer file than dynamic, hence -1)
			if numNodes == -1 {
				numNodes = len(setupPrm.vdw)-1
			}
			// Setup for BAR
			BARSetup(&genPrm)
			// Run BAR
			ng.BARManager(&genPrm, &barPrm, numNodes)
			// Get results
			returnResults(&genPrm)
		default:
			err = errors.New("invalid parameter " + args[2] + ". Valid parameters in this position are: \"setup\", \"dynamic\", \"bar\", \"auto\".\n " +
				"If more assistance is needed with this issue, launch goFEP with no arguments to access built-in help function")
			log.Fatal(err)
		}
	}
}

func help() {
	fmt.Println()
	fmt.Println("goFEP Beta v" + goFEPversion)
	fmt.Println("Compiled " + goFEPcompileDate + " for " + goFEPPlatform + "-" + goFEPArch )
	fmt.Println("Justin Gourary | jtgourary@utexas.edu")
	fmt.Println("Univ. of Texas at Austin | Dept. of Biomedical Eng.")
	fmt.Println("Pengyu Ren Lab")
	fmt.Println()
	fmt.Println("IMPORTANT NOTE: goFEP must ALWAYS be run from bme-nova")
	fmt.Println("i.e. \"ssh bme-nova\" THEN \"gofep path/to/xxx.ini xxx xxx ##\"")
	fmt.Println()
	fmt.Println("The first argument in a call to goFEP should always be a path to a configuration file")
	fmt.Println("A sample configuration file with explanatory comments can be found at /home/jtg2769/software/gofep/sampleInput/settings.ini")
	fmt.Println()
	fmt.Println("Second argument should always be a task to perform")
	fmt.Println("Valid tasks are: \"setup\", \"dynamic\", \"bar\",\"auto\"")
	fmt.Println("Intended usage is to either run setup, dynamic, and bar in sequence, or, if you're feeling lucky today, to run auto, which does all three sequentially")
	fmt.Println()
	fmt.Println("Make a selection to learn more about these tasks and how to run them:")
	fmt.Println("(1) setup")
	fmt.Println("(2) dynamic")
	fmt.Println("(3) bar")
	fmt.Println("(4) auto")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		err = errors.New("invalid selection: could not read input")
		log.Fatal(err)
	}
	textInt, err := strconv.Atoi(strings.Fields(text)[0])
	if err != nil {
		err = errors.New("invalid selection: could not convert input to integer")
		log.Fatal(err)
	}
	switch textInt {
	case 1:
		fmt.Println()
		fmt.Println("* setup creates files to run tinker dynamic_omm on in directory specified in config file")
		fmt.Println()
		fmt.Println("* further arguments are (1) the path to a configuration ini file")
		fmt.Println()
		fmt.Println("* usage: \"gofep /path/to/config.ini setup\"")
		fmt.Println()
	case 2:
		fmt.Println()
		fmt.Println("* dynamic runs tinker dynamic_omm on files in the directory specified in the config file with the parameters specified in the config file")
		fmt.Println()
		fmt.Println("* further arguments are the (1) path to a config file (2) call type (3) max number of nodes to use (a positive integer or to have it set automatically -1)")
		fmt.Println()
		fmt.Println("* Valid selections for call type are: \"new\" and \"all\"")
		fmt.Println()
		fmt.Println("* \"new\" runs dynamic_omm on only those xyz files in the directory that don't yet have a corresponding arc file")
		fmt.Println("* this is useful if for example you would like to add an intermediate vdw/ele combination between two existing combinations")
		fmt.Println()
		fmt.Println("* \"all\" runs dynamic_omm on all xyz files in the directory")
		fmt.Println()
		fmt.Println("* usage: \"gofep /path/to/config.ini dynamic new 20\"")
		fmt.Println()
	case 3:
		fmt.Println()
		fmt.Println("* bar runs tinker bar_omm on files in the directory specified in the config file with the parameters specified in the config file")
		fmt.Println()
		fmt.Println("* further arguments are the (1) path to a config file (2) max number of nodes to use (a positive integer or to have it set automatically -1)")
		fmt.Println()
		fmt.Println("* usage: \"gofep /path/to/config.ini bar 10\"")
		fmt.Println()
	case 4:
		fmt.Println()
		fmt.Println("* auto is equivalent to running the following tasks in sequence:")
		fmt.Println("  setup -> dynamic (with call type \"new\") -> bar")
		fmt.Println()
		fmt.Println("* further arguments are the (1) path to a config file (2) max number of nodes to use (a positive integer or to have it set automatically -1)")
		fmt.Println()
		fmt.Println("* usage: \"gofep /path/to/config.ini auto -1\"")
		fmt.Println()
		fmt.Println("* Restart help to learn more about each of the above tasks")
		fmt.Println()
	default:
		fmt.Println()
		fmt.Println("* Invalid selection")
		fmt.Println()
	}
}