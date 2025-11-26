package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	misc "github.com/nathanverrilli/nlvMisc"

	"github.com/spf13/pflag"
)

const jsonDataExample string = "\n{ \"endpoints\": [" +
	"\n\t{\t\"baseUrl\": \"https://ocpi.chargepoint.com/\"," +
	"\n\t\t\"token\": \"ORG ==NA==PROD==DEN==\"\n\t}," +
	"\n\t{\t\"baseUrl\": \"https://ocpi-ca.chargepoint.com/\"," +
	"\n\t\t\"token\": \"ORG ==CA==PROD==DEN==\"\n\t}" +
	"\n] }\n"

// wordSepNormalizeFunc all options are lowercase, so
// ... lowercase they shall be
func wordSepNormalizeFunc(_ *pflag.FlagSet, name string) pflag.NormalizedName {
	return pflag.NormalizedName(strings.ToLower(name))
}

var nFlags *pflag.FlagSet

/* secret flags */
var FlagSlow bool
var FlagDebugger bool
var FlagDestInsecure bool
var FlagMaxCalls int

/* standard flags */

var FlagHelp bool
var FlagQuiet bool
var FlagVerbose bool
var FlagDebug bool

/* program flags  */

var FlagAuthTokenFile string

// initFlags initializes the command line flags for the program.
// It sets up the flag set, defines the flags, and parses the command line arguments.
// Some flags are hidden from the user as they are meant only for testing
func initFlags() {
	var err error

	hideFlags := make(map[string]struct{}, 4)

	nFlags = pflag.NewFlagSet("default", pflag.ContinueOnError)
	nFlags.SetNormalizeFunc(wordSepNormalizeFunc)

	// secret flags

	nFlags.BoolVarP(&FlagDebugger, "debugger", "", false,
		"enable http mutex for debugging (only one http call at a time)")
	hideFlags["debugger"] = struct{}{}

	nFlags.BoolVarP(&FlagSlow, "slow", "", false,
		"Add some time between http calls (do not hammer server)")
	hideFlags["slow"] = struct{}{}

	nFlags.BoolVarP(&FlagDestInsecure, "insecure", "", false,
		"do not verify server cert (dangerous)")
	hideFlags["insecure"] = struct{}{}

	nFlags.IntVarP(&FlagMaxCalls, "maxcalls", "", 0,
		"Make only a few calls to any feed to speed up testing (0 == no limit)")
	hideFlags["maxcalls"] = struct{}{}

	// program flags

	nFlags.StringVarP(&FlagAuthTokenFile, "tokens", "", "endpoints.json",
		"JSON file containing base URL and authorization token for each feed\nIn this format:\n"+jsonDataExample)

	nFlags.BoolVarP(&FlagDebug, "debug", "d",
		true, "Enable additional informational and operational logging output for debug purposes")

	nFlags.BoolVarP(&FlagVerbose, "verbose", "v",
		true, "Supply additional run messages; use --debug for more information")

	nFlags.BoolVarP(&FlagHelp, "help", "h",
		false, "Display help message and usage information")

	nFlags.BoolVarP(&FlagQuiet, "quiet", "q",
		false, "Suppress log output to stdout and stderr (output still goes to logfile)")

	for flagName := range hideFlags {
		err = nFlags.MarkHidden(flagName)
		if nil != err {
			xLog.Printf("could not mark flag %s hidden because %s\n",
				flagName, err.Error())
			myFatal()
		}
	}

	// Fetch and load the program flags
	err = nFlags.Parse(os.Args[1:])
	if nil != err {
		_, _ = fmt.Fprintf(os.Stderr, "\n%s\n", nFlags.FlagUsagesWrapped(75))
		xLog.Fatalf("\nerror parsing flags because: %s\n%s %s\n%s\n\t%v\n",
			err.Error(),
			"  common issue: 2 hyphens for long-form arguments,",
			"  1 hyphen for short-form argument",
			"  Program arguments are: ",
			os.Args)
	}

	// do quietness setup first
	// only write to logfile not stderr
	// for debug and verbose messages
	if FlagQuiet {
		xLog.SetOutput(xLogBuffWriter)
		// messages only to logfile, not stderr
	}

	if FlagDebug && FlagVerbose {
		xLog.Println("\t\t/*** start program flags ***/\n")
		nFlags.VisitAll(logFlag)
		xLog.Println("\t\t/***   end program flags ***/")
	}

	if FlagHelp {
		var err1, err2 error
		_, thisCmd := filepath.Split(os.Args[0])
		_, err1 = fmt.Fprint(os.Stdout, "\n", "usage for ", thisCmd, ":\n")
		_, err2 = fmt.Fprintf(os.Stdout, "%s\n", nFlags.FlagUsagesWrapped(75))
		if nil != err1 || nil != err2 {
			xLog.Printf("huh? can't write to os.stdout because\n%s",
				misc.ConcatenateErrors(err1, err2).Error())
		}
		UsageMessage()
		_, _ = fmt.Fprintf(os.Stdout, "\t please see USAGE.MD for ")
		myFatal(0)
	}

	if FlagVerbose {
		errMsg := ""
		user, host, err := misc.UserHostInfo()
		if nil != err {
			errMsg = " (encountered error " + err.Error() + ")"
		}
		xLog.Printf("Verbose mode active (all debug and informative messages) for %s@%s%s",
			user, host, errMsg)
	}

	if FlagDebug && FlagVerbose {
		_, exeName := filepath.Split(os.Args[0])
		exeName = strings.TrimSuffix(exeName, filepath.Ext(exeName))
		bi, ok := debug.ReadBuildInfo()
		if !ok {
			xLog.Printf("huh? Could not read build information for %s "+
				"-- perhaps compiled without module support?", exeName)
		} else {
			xLog.Printf("\n***** %s BuildInfo: *****\n%s\n%s\n",
				exeName, bi.String(), strings.Repeat("*", 22+len(exeName)))
		}
	}

}

// logFlag -- This writes out to the logger the value of a
// particular flag. Called indirectly. `Write()` is used
// directly to prevent interactions with backslash
// in filenames
func logFlag(flag *pflag.Flag) {
	var sb strings.Builder
	sb.WriteString(" flag ")
	sb.WriteString(flag.Name)
	sb.WriteString(" has value [")
	sb.Write([]byte(flag.Value.String()))
	sb.WriteString("] with default [")
	sb.Write([]byte(flag.DefValue))
	sb.WriteString("]\n")

	_, _ = xLog.Writer().Write([]byte(sb.String()))
}

// UsageMessage prints useful information to the log
// Example usage:
//
//	UsageMessage()
func UsageMessage() {
	var sb strings.Builder
	sb.WriteString("Program Errors:")
	sb.WriteString("\n\t -3: Internal function failed (see log)")
	sb.WriteString("\n\t -2: Program interrupt from external signal (see log)")
	sb.WriteString("\n\t -1: External function failed (see log)")
	sb.WriteString("\n\t  0: success\n")
	sb.WriteString("/*******************************************/\n")
	sb.WriteString("Useful Program Information Here\n")
	xLog.Println(sb.String())
}
