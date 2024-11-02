Kirke
=====

Kirke is a yet another command-line tool for converting JSON strings into Golang struct definitions. It’s designed to be flexible, allowing for JSON input from both files and pipes, and it handles pointer types for nested structures when specified.

Features
--------

* Supports both inline and outline Golang structs definitions
* Receives JSON strings directly or from a file
* Outputs formatted Golang structs
* Option to define struct and array fields as pointers
* Flexible with file extensions (no need for `.json`)
* Option to disable the pager for larger outputs

Installation
------------

To install `kirke`, clone the repository and build the binary:

    git clone https://github.com/magicdrive/kirke.git
    cd kirke
    go build -o kirke

Alternatively, use `go install`:

    go install github.com/magicdrive/kirke@latest

Usage
-----

    kirke [OPTIONS] [ARGUMENTS]

### Description

Receives a JSON string and outputs it as a Golang struct definition.

### Options

Option                             | Description                                                                      | Default
--------------------------------   | -------------------------------------------------------------------------------- | ---------------
`-h`, `--help`                     | Show this help message and exit                                                  |
`-v`, `--version`                  | Show version and exit                                                            |
`-n`, `--name <root-name>`         | Specify the name of the root struct. Converts to camel case automatically.       | `AutoGenerated`
`-f`, `--file <json-file>`         | Specify the input file containing JSON. Accepts files without `.json` extension. |
`-a`, `--null-as <null-type-name>` | Specify the null-type name. Used to replace `null` type from json.               | `interface{}`
`-j`, `--json <json-string>`       | Specify the JSON string to be converted to a struct.                             |
`-p`, `--pipe`                     | Receive JSON from a pipe instead of a direct argument above all else.            | `false`
`--with-pointer`                   | Define struct and array fields as pointers.                                      | `false`
`--outline`                        | Defines struct and array fields as outline struct. (default: true).              | `true`
`--inline`                         | Defines struct and array fields as inline struct. (default: false)               | `false`
`--no-pager`                       | Prevents usage of a pager even if output exceeds terminal size.                  | `false`

### Arguments

* `<root-name>`: Used as the name of the root struct in the output. Automatically converted to camel case.
* `<json-file>`: File containing JSON data. Can be read even if the extension is not `.json`.
* `<json-string>`: Direct JSON string input.
* `<null-type-name>`: The type name used to replace `null` type from json.

### Environments

Environment                        | Description                                                                      | Default
--------------------------------   | -------------------------------------------------------------------------------- | ---------------
`KIRKE_DEFAULT_ROOT_NAME`          | Specified default used <root-name>                                               | `AutoGenerated`
`KIRKE_DEFAULT_NULL_AS`            | Specified default used <null-type-name>.                                         | `interface{}`
`KIRKE_DEFAULT_OUTPUT_MODE`        | Specified the default used output mode. Only `outline` and `inline` are valid.   | `outline`
`KIRKE_DEFAULT_NO_PAGER`           | When value "1", output doesn't use the pager.                                    | `0`
`KIRKE_DEFAULT_WITH_POINTER`       | When value "1", defines struct and array fields as pointers.                     | `0`

Examples
--------

### Example 1: Direct JSON string input

Convert a JSON string to a struct without using a pager and with a specified root name:

    kirke -j '{"key": "value"}' --no-pager --name MyStruct

### Example 2: JSON from a file

Convert JSON data from a specified file to a struct with a specified root name:

    kirke -f ./path/to/example.json --name MyExample

### Example 3: JSON input from a pipe

Read JSON data from a pipe, use pointers for nested fields, and output as a struct:

    echo '{"key": "value"}' | kirke --with-pointer

### Example 4: JSON input from a pipe force.

Read JSON data from a pipe, use pointers for nested fields, and output as a struct (-j option is ignored.):

    echo '{"key": "value"}' | kirke --pipe --with-pointer -j '{"key2": "value2"}'

### Example 5: JSON null as `*string` in GO struct.

Read JSON data from a pipe, use pointers for nested fields, and output as a struct (-j option is ignored.):

    echo '{"key": "value", "obj": {"num": 1} }' | kirke -a "*string"

LICENCE
-----

[MIT License](https://github.com/magicdrive/kirke/LICENCE)
