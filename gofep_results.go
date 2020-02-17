package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)



func returnResults(genPrm *generalParameters) {
	// Get all subdirectories in bar folder
	subDirs, err := getBARSubDirs(genPrm.targetDirectory)
	if err != nil {
		fmt.Println("Failed to find BAR subdirectories in " + genPrm.targetDirectory)
	}
	forwardFEPSlice := make([][]float64, len(subDirs))
	backwardFEPSlice := make([][]float64, len(subDirs))

	// iterate
	for i := 0; i < len(subDirs); i++ {
		forwardFEPSlice[i] = make([]float64,2)
		backwardFEPSlice[i] = make([]float64,2)

		forwardFEP, backwardFEP := getStepResult(subDirs[i])
		forwardFEPSlice[i] = forwardFEP
		backwardFEPSlice[i] = backwardFEP
	}

	// Store total energy [0] and root square error [1]
	totalForwardValues := make([]float64,2)
	totalBackwardValues := make([]float64,2)

	// Compute total energy and square error
	for i := 0; i < len(subDirs); i++ {

		totalForwardValues[0] += forwardFEPSlice[i][0]
		totalForwardValues[1] += math.Pow(forwardFEPSlice[i][1],2)
		totalBackwardValues[0] += backwardFEPSlice[i][0]
		totalBackwardValues[1] += math.Pow(backwardFEPSlice[i][1],2)
	}

	// Get sqrt of sum of square errors
	totalForwardValues[1] = math.Sqrt(totalForwardValues[1])
	totalBackwardValues[1] = math.Sqrt(totalBackwardValues[1])

	outputFile := filepath.Join(genPrm.targetDirectory, finalResultFileName)

	// Create file to store final results
	out, err := os.Create(outputFile)
	if err != nil {
		fmt.Println("Failed to create output file: " + outputFile)
		log.Fatal(err)
	}

	// Write Forward Results
	_, err= out.WriteString("Forward FEP Results \n")
	for i := 0; i < len(subDirs); i++ {
		params := strings.Split(filepath.Base(subDirs[i]), "_")
		_, err= out.WriteString( params[0] + " to " + params[1] + "\t" + fmt.Sprintf("%e",forwardFEPSlice[i][0]) + " +/- " + fmt.Sprintf("%e", forwardFEPSlice[i][1]) + " kcal/mol \n")
	}
	_, err= out.WriteString("\nTotal: " + fmt.Sprintf("%e",totalForwardValues[0]) + " +/- " + fmt.Sprintf("%e",totalForwardValues[1]) + " kcal/mol \n")
	if err != nil {
		fmt.Println("Failed to write Forward FEP values to output file: " + outputFile)
		log.Fatal(err)
	}

	// Write Reverse Results
	_, err= out.WriteString("\nBackward FEP Results \n")
	for i := 0; i < len(subDirs); i++ {
		params := strings.Split(filepath.Base(subDirs[i]), "_")
		_, err= out.WriteString( params[0] + " to " + params[1] + "\t" + fmt.Sprintf("%e",backwardFEPSlice[i][0]) + " +/- " + fmt.Sprintf("%e", backwardFEPSlice[i][1]) + " kcal/mol \n")
	}
	_, err= out.WriteString("\nTotal: " + fmt.Sprintf("%e",totalBackwardValues[0]) + " +/- " + fmt.Sprintf("%e",totalBackwardValues[1]) + " kcal/mol \n")
	if err != nil {
		fmt.Println("Failed to write Backward FEP values to output file: " + outputFile)
		log.Fatal(err)
	}
}

func getStepResult(subDir string) ([]float64, []float64) {
	fileInfo, err := ioutil.ReadDir(subDir)
	if err != nil {
		fmt.Println("Failed to read directory " + subDir)
		log.Fatal(err)
	}

	var forwardFEPResults []float64
	var backwardFEPResults []float64

	foundResultFile := false

	for i := 0; i < len(fileInfo); i++ {
		if fileInfo[i].Name() == resultFileName {
			foundResultFile = true

			// open file
			resultFilePath := filepath.Join(subDir, resultFileName)
			resultFile, err := os.Open(resultFilePath)
			if err != nil {
				fmt.Println("Failed to open log file: " + resultFilePath)
				log.Fatal(err)
			}

			// read line by line
			scanner := bufio.NewScanner(resultFile)
			for scanner.Scan() {
				// get next line
				line := scanner.Text()

				// Search for keywords
				if strings.Count(line, "Free Energy via Forward FEP") > 0 {
					tokens := strings.Fields(line)

					// Get forward FEP values
					energy, warning := strconv.ParseFloat(tokens[5], 64)
					if warning != nil {
						energy = math.NaN()
						fmt.Println("Warning: Failed to parse Forward FEP energy to float in file \"" + resultFilePath + "\". Setting energy to NaN")
					}
					plusMinus, warning := strconv.ParseFloat(tokens[7], 64)
					if warning != nil {
						energy = math.NaN()
						fmt.Println("Warning: Failed to parse Forward FEP energy +/- to float in file \"" + resultFilePath + "\". Setting energy to NaN")
					}

					forwardFEPResults = []float64{energy, plusMinus}

				} else if strings.Count(line, "Free Energy via Backward FEP") > 0 {
					tokens := strings.Fields(line)

					// Get forward FEP values
					energy, warning := strconv.ParseFloat(tokens[5], 64)
					if warning != nil {
						energy = math.NaN()
						fmt.Println("Warning: Failed to parse Backward FEP energy to float in file \"" + resultFilePath + "\". Setting energy to NaN")
					}
					plusMinus, warning := strconv.ParseFloat(tokens[7], 64)
					if warning != nil {
						energy = math.NaN()
						fmt.Println("Warning: Failed to parse Backward FEP energy +/- to float in file \"" + resultFilePath + "\". Setting energy to NaN")
					}

					backwardFEPResults = []float64{energy, plusMinus}


				}
			}

		}
	}
	if foundResultFile == false {
		err = errors.New("Failed to find a file containing BAR2 results in directory " + subDir + ". Such files " +
			"should be named " + resultFileName)
		log.Fatal(err)
	}

	return forwardFEPResults, backwardFEPResults
}