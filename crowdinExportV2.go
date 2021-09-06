package main

//	F.Demurger 2020-02
//
//	Export a Crowdin project languages and files (translated and approved strings only) - uses Crowdin API V2.
//
//	crowdinExport [options] <access token> <project Id> <zip file path/name>
//
//  3 mandatory args:
//		<access token>
//		<project Id>
//  	<zip file path/name>
//
//	Options:
//      Option -v version
//      Option -b to build the project
//            Optionnaly build the project and download the zip with all languages.
//		Option -u to specify the api url. If option not used the default api url used will be "https://crowdin.api.V2"
//      Option -p <proxy url> to use a proxy.
//      Option -t <timeout in second>. Defines a timeout for each communication with the server. This doesn't represent an overall timeout. Default timeout set in lib: 40s.
//		Option -n no spinning thingy while we wait for the file (for unattended usage).
//		Option -d <path/file> log debug info in file
//      Option -c if used, exports completly translated files only.
//
//      Returns 1 if there was an error
//
//      If option built is used, returns "built" or "skipped" if the command is successful and depending if the build was actually done.
//
//	Prior to compil, install lib: go get github.com/fabdem/go-crowdinv2
//	Linux: Cross compilation AMD64:  env GOOS=windows GOARCH=amd64 go build crowdinExportV2.go

import (
	"flag"
	"fmt"
	"github.com/fabdem/go-crowdinv2"
	"os"
	"strconv"
	"strings"
	"time"
)

var idx int = 0
var finishChan chan struct{}

func animation(c *crowdin.Crowdin) {
	sequence := [...]string{"|", "/", "-", "\\"}

	for {
		select {
		default:
			str := fmt.Sprintf("%s %d%%", sequence[idx], c.GetPercentBuildProgress())
			fmt.Printf("%s%s", str, strings.Repeat("\b", len(str)))
			// fmt.Printf("%s %d", sequence[idx], c.GetProjectId2())
			idx = (idx + 1) % len(sequence)
			amt := time.Duration(100)
			time.Sleep(time.Millisecond * amt)

		case <-finishChan:
			return
		}
	}
}

func main() {

	var versionFlg bool
	var buildFlg bool
	var proxy string
	var timeoutsec int
	var nospinFlg bool
	var uRL string
	var debug string
	var completedFilesFlg bool

	const usageVersion = "Display Version"
	const usageBuild = "Request a build"
	const usageProxy = "Use a proxy - followed with url"
	const usageTimeout = "Set the build timeout in seconds (default 50s)."
	const usageNospin = "No spinning |"
	const usageUrl = "Specify the API URL"
	const usageDebug = "Store Debug info in a file followed with path and filename"
	const usageCompletedFiles = "Exports completely translated files only (to be used along with -b option)"

	// Have to create a specific set, the default one is poluted by some test stuff from another lib (?!)
	checkFlags := flag.NewFlagSet("check", flag.ExitOnError)

	checkFlags.BoolVar(&versionFlg, "version", false, usageVersion)
	checkFlags.BoolVar(&versionFlg, "v", false, usageVersion+" (shorthand)")
	checkFlags.IntVar(&timeoutsec, "timeout", 50, usageTimeout)
	checkFlags.IntVar(&timeoutsec, "t", 50, usageTimeout+" (shorthand)")
	checkFlags.BoolVar(&buildFlg, "build", false, usageBuild)
	checkFlags.BoolVar(&buildFlg, "b", false, usageBuild+" (shorthand)")
	checkFlags.StringVar(&proxy, "proxy", "", usageProxy)
	checkFlags.StringVar(&proxy, "p", "", usageProxy+" (shorthand)")
	checkFlags.BoolVar(&nospinFlg, "nospin", false, usageNospin)
	checkFlags.BoolVar(&nospinFlg, "n", false, usageNospin+" (shorthand)")
	checkFlags.StringVar(&uRL, "url", "", usageUrl)
	checkFlags.StringVar(&uRL, "u", "", usageUrl+" (shorthand)")
	checkFlags.StringVar(&debug, "debug", "", usageDebug)
	checkFlags.StringVar(&debug, "d", "", usageDebug+" (shorthand)")
	checkFlags.BoolVar(&completedFilesFlg, "completedFiles", false, usageCompletedFiles)
	checkFlags.BoolVar(&completedFilesFlg, "c", false, usageCompletedFiles+" (shorthand)")
	checkFlags.Usage = func() {
		fmt.Printf("Usage: %s [opt] <key> <project ID> <path and name of zip>\n", os.Args[0])
		checkFlags.PrintDefaults()
	}

	// Check parameters
	checkFlags.Parse(os.Args[1:])

	if versionFlg {
		fmt.Printf("Version %s\n", "2021-09  v2.3.0")
		os.Exit(0)
	}

	// Parse the command parameters
	index := len(os.Args)
	key := os.Args[index-3]
	projectId, err := strconv.Atoi(os.Args[index-2])
	if err != nil {
		fmt.Printf("\nProjectId needs to be a number %s", err)
		os.Exit(0)
	}
	zipfilename := os.Args[index-1]

	// Create a connection
	api, err := crowdin.New(key, projectId, uRL, proxy)
	if err != nil {
		fmt.Printf("\ncrowdinExport() - connection problem %s\n", err)
		os.Exit(1)
	}

	if !nospinFlg { // Check if we need to spin the '|'
		finishChan = make(chan struct{})
		go animation(api)
	}

	if len(debug) > 0 {
		logFile, err := os.Create(debug)
		if err != nil {
			fmt.Printf("\ncrowdinExportV2() - Can't create debug file %s %v", debug, err)
			os.Exit(1)
		}
		api.SetDebug(true, logFile)
	}

	var buildId int

	if buildFlg {
		// Request a build
		if !completedFilesFlg {
			buildId, err = api.BuildAllLg(timeoutsec, true, true, false) // Export translated and approved strings only
		} else {
			buildId, err = api.BuildAllLg(timeoutsec, false, true, true) // Export approved strings only and fully translated files only
		}
		if err != nil {
			fmt.Printf("\ncrowdinExportV2() build request error\n%s\n%s\n\n", buildId, err)
			os.Exit(1)
		}
	} else {
		// Get most recent build for the project
		buildId, err = api.GetBuildId()
		if err != nil {
			fmt.Printf("\ncrowdinExportV2() can't find a build for projectId:\n%s", err)
			os.Exit(1)
		}
	}

	// Download zip file
	err = api.DownloadBuild(zipfilename, buildId)
	if err != nil {
		fmt.Printf("\ncrowdinExportV2() DownloadBuild error %s\n\n", err)
		os.Exit(1)
	}

	if !nospinFlg {
		close(finishChan) // Stop animation
	}
}
