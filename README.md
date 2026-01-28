# OwlDB
noSQL database built from the ground up to store databases, collections, and documents.
Note that this project is part of the class COMP 318: Concurrent Program Design at Rice
University. In order to properly test all actions, it requires an SSH tunnel to Rice
University's secure computers, however to manually test and understand, you may build
and run the test files separately.

## Getting started

Be sure to initialize the project using:

```go mod init github.com/april2546/OwlDB```

You may then install the JSON schema package:

```go get github.com/santhosh-tekuri/jsonschema/v5```

Note the "v5" at the end.  This is very important.  You may only
use and import version 5 of this package.

Remember that you must **not** install any other external packages.

Be sure to commit the resulting `go.mod` and `go.sum` files.

## Provided Code

### main

The provided `main.go` file is a simple skeleton for you to start
with. It handles gracefully closing the HTTP server when Ctrl-C is
pressed in the terminal that is running your program.  It does little
else.

### jsondata

The provided `jsondata` package provides a `JSONValue` type, a
`Visitor` interface and a few basic functions to work with JSON data.
You **must** use this package whenever you access the contents of a
JSON document in your program.

### logger

The provided `logger` package provides a structured logger based on
the standard `log/slog` package that allows you to set the log level
and colorize the output.

## Build

Build the program as follows:

```go build -o owldb```

Assuming you have a file "document.json" that holds your desired
document schema and a file "tokens.json" that holds a set of tokens
for authorization, then you can run your program like so:

```./owldb -s document.json -t tokens.json -p 3318```

Note that you can always run your program without building it first as
follows:

```go run main.go -s document.json -t tokens.json -p 3318```