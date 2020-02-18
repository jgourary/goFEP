# goFEP
goFEP is a command line application written in Go for the complete automation of free energy perturbation (FEP) simulations using Tinker-OpenMM on the Ren Lab cluster
## Features
* Cross-platform: precompiled goFEP binaries are available for Linux and OS X operating systems for amd64 architecture
* No Dependancies: goFEP does not require any additional software beyond the binary
* Easy to use: Install goFEP to your PATH and run FEP from anywhere with a single command
* Configurable: goFEP enables control of all the options that Tinker-OpenMM functions `dynamic_omm` and `bar_omm` do through an easy to configure INI file
* Fast: goFEP automatically assigns your jobs to the most powerful available GPUs in the cluster
* User Friendly: goFEP automatically detects missing or faulty parameters in its configuration file, and warns the user
## Inputs
* goFEP requires the same input files as Tinker: an `xyz`, `key` and `prm` file
* In addition, goFEP needs three things to run: a binary file, settings INI file, and node INI file. These will hereafter be referred to as `gofep`, `settings.ini`, and `nodes.ini`, though you can name them whatever you like.
### Installing the Binary File
###### From Binary
1. Copy a binary file from the releases section of this repository or `/home/jtg2769/software/gofep/releases` to a folder of your choice
2. Add the folder containing the binary to your path definition: `PATH=$PATH:/home/jtg2769/exampleFolder/`
###### From Source
1. Download and install Go (https://golang.org/dl) for your system
2. Download the source files from this repository to a folder of your choice
3. Convert the downloaded source files to a binary using `go build` (http://golang.org/pkg/go/build/)
4. Add the folder containing the binary to your path definition: `PATH=$PATH:/home/jtg2769/exampleFolder/`
### Writing a Settings INI file
* `settings.ini` contains all the parameters needed to run FEP
* Commenting is allowed in this file using `#`
* A template `settings.ini` with all parameters specified and explanatory comments can be found at `/home/jtg2769/software/gofep/sampleInput/`
### Writing a Node INI file
* `node.ini` contains information on all the nodes in the cluster
* Commenting is allowed in this file using `#`
* A template `nodes.ini` with explanatory comments can be found at `/home/jtg2769/software/gofep/sampleInput/`
## Running goFEP from the command line
goFEP can run in five modes: `help`,`setup`,`dynamic`,`bar`, and `auto`
### help
You can activate the built-in help function by running goFEP with no arguments: `gofep`
### setup
* `setup` sets up the file structure for `dynamic`
* It takes the `vdwLambdas`, `eleLambdas`, and `restraints` parameters specified in the `setup` block of `settings.ini` and creates a folder for each combination with an `xyz` and `key` file inside
* setup exists mainly to give you a chance to determine whether goFEP created the files you desired before you jump into a lengthy molecular dynamics run
###### Arguments
1. the path to `settings.ini`
###### Example Usage
`gofep /path/to/settings.ini setup`
### dynamic
* `dynamic` runs Tinker-OpenMM's dynamic_omm.x executable on each directory created by `setup` in parallel on different cluster nodes
###### Arguments
1. the path to `settings.ini`
2. the calltype 
* `new`: run dynamic on every directory without a pre-existing arc file
* `all`: run dynamic on every directory, starting from a pre-existing arc file if it exists
3. the maximum number of nodes to run on. If set to `-1`, it will try to assign each job to a different node.
###### Example Usage
`gofep /path/to/settings.ini dynamic new 20`
### bar
* `bar` runs Tinker-OpenMM's bar_omm.x executable (parts 1 & 2) between each directory created by `dynamic` in parallel on different cluster nodes, then writes the forward and backward free energy and error to `results.txt` in the main directory
###### Arguments
1. the path to `settings.ini`
2. the maximum number of nodes to run on. If set to `-1`, it will try to assign each job to a different node.
###### Example Usage
`gofep /path/to/settings.ini bar 19`
### auto
* `auto` is equivalent to running `setup`, then `dynamic` (with call type `new`) then `bar`
###### Arguments
1. the path to `settings.ini`
2. the maximum number of nodes to run on. If set to `-1`, it will try to assign each job to a different node.
###### Example Usage
`gofep /path/to/settings.ini auto -1`
## Practical Usage
* When first using goFEP, it is recommended that you first run `setup`, then once you have verified that goFEP set up for FEP as you intended, run `auto`
* This is because it is tedious to kill 20+ jobs on different nodes if you realize they are not doing what you wanted
* Once you have used goFEP several times, it is anticipated that you will mostly use `auto`
### If BAR doesn't converge
#### Option 1: Add intermediate vdw/ele steps
1. Edit vdwLambdas, eleLambdas, restraints in the setup block of `settings.ini` to include intermediate step(s)
2. Run `auto`. goFEP will run `dynamic` on the intermediate steps, then run `bar` again 
#### Option 2: Run dynamic for longer
1. Run `dynamic all` to double the duration of your `arc` files (goFEP will run dynamic again with the same settings starting from the existing `arc` file) OR create a new `settings.ini` and edit the `dynamic` blocks manually for more granular control
2. Run `bar`




