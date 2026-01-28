package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"testing"

	"github.com/RICE-COMP318-FALL24/owldb-p1group32/jsondata"
)

func saveOutput(f func()) string {
	r, w, _ := os.Pipe()
	log.SetOutput(w)
	f()
	w.Close()
	out, _ := ioutil.ReadAll(r)
	log.SetOutput(os.Stderr)
	return string(out)
}

type mainInputs struct {
	pflag       string
	pflagInput  string
	sflag       string
	sflagInput  string
	tflag       string
	tflagInput  string
	expectError bool
}

func parsingFlags(inputs mainInputs) (port string, jsonfile string, tokenfile string) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	os.Args = []string{"cmd", inputs.pflag, inputs.pflagInput, inputs.sflag, inputs.sflagInput, inputs.tflag, inputs.tflagInput}

	portnum := flag.String("p", "3318", "Port to listen on")
	jsonFlag := flag.String("s", "", "Name of file with JSON schema")
	tokenFlag := flag.String("t", "", "JSON file with mapping of usernames to tokens")

	flag.Parse()
	return *portnum, *jsonFlag, *tokenFlag
}

func fakeMain(portnum string, jsonFlag string, tokenFlag string) error {

	// ensure a file with json schema is named
	if jsonFlag == "" {
		return fmt.Errorf("Error: Must specify the name of a file with the JSON schema using the -s flag\n")
	}
	// ensure mapping of usernames to tokens is given
	if tokenFlag == "" {
		return fmt.Errorf("Error: Must specify the JSON file with mapping of user names to tokens using the -t flag\n")
	}

	// Attempting to compile provided schema
	schem, err := jsondata.New(jsonFlag)
	if err != nil {
		return fmt.Errorf("Error: Provided schema could not be compiled\n")
	}
	_ = schem
	// Checking for a valid port number
	port, err := strconv.Atoi(portnum)
	if err != nil {
		return fmt.Errorf("Error: Port Number provided was not a valid number, provided value is %e", err)
	}
	_ = port
	return nil
}
func TestMainFlags(t *testing.T) {

	testCases := []mainInputs{
		{
			pflag:       "-p",
			pflagInput:  "1025",
			sflag:       "-s",
			sflagInput:  "schemaAny.json",
			tflag:       "-t",
			tflagInput:  "tokens.json",
			expectError: false,
		},
		// invalid port number
		{
			pflag:       "-p",
			pflagInput:  "hahaha",
			sflag:       "-s",
			sflagInput:  "schemaAny.json",
			tflag:       "-t",
			tflagInput:  "tokens.json",
			expectError: true,
		},
		// non existent json file
		{
			pflag:       "-p",
			pflagInput:  "1025",
			sflag:       "-s",
			sflagInput:  "",
			tflag:       "-t",
			tflagInput:  "tokens.json",
			expectError: true,
		},
		// non existent token file
		{
			pflag:       "-p",
			pflagInput:  "1025",
			sflag:       "-s",
			sflagInput:  "schemaAny.json",
			tflag:       "-t",
			tflagInput:  "",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		var portnum string
		var jsonFlag string
		var tokenFlag string
		portnum, jsonFlag, tokenFlag = parsingFlags(tc)

		res := fakeMain(portnum, jsonFlag, tokenFlag)
		if res != nil {
			if tc.expectError == true {
				return
			} else {
				t.Fatal("Test case failed, an error occured during flag parsing when one should not have.")
			}
		} else {
			if tc.expectError == true {
				t.Fatalf("Test case failed: an error was expected but not returned")
			}
		}
	}

}
