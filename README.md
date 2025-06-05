# cURL Data Extractor and Decoder

## Description

The primary aim of this Go utility is to decode gzipped data from cURL requests, particularly the content found within the `--data-raw $'(...)'` payload (often obtained by copying a request as cURL from browser developer tools). To achieve this, the utility extracts the raw string, processes various escape sequences (mimicking Python's `s.encode('latin1').decode('unicode_escape').encode('latin1')` behavior and applying Latin-1 encoding constraints from U+0000 to U+00FF), decompresses the Gzipped data, and then pretty-prints the resulting JSON.
## Features

* **Extracts Data**: Isolates the content from the `--data-raw $'(...)'` part of a cURL command.
* **Decodes Escapes**: Handles common escape sequences such as `\n`, `\r`, `\t`, `\\`, `\'`, `\"`, as well as hexadecimal (`\xHH`), 4-digit Unicode (`\uHHHH`), 8-digit Unicode (`\UHHHHHHHH`), and octal (`\OOO`) escapes.
* **Latin-1 Constraint**: During decoding, Unicode escapes (`\u...`, `\U...`) must represent codepoints within the Latin-1 range (U+0000 to U+00FF). Literal non-ASCII characters in the input string must also fall within this range.
* **Gzip Decompression**: Automatically attempts to decompress the decoded data if it's in Gzip format.
* **JSON Parsing & Pretty-Printing**: Parses the (potentially decompressed) data as JSON and outputs it in a human-readable, indented format.
* **File I/O**: Reads the cURL command from a specified input file and writes the processed JSON to a specified output file.
* **Command-Line Flags**: Allows customization of input and output file paths.

## Prerequisites

* **Go**: Version 1.18 or higher (due to the use of `any` for JSON unmarshaling, and general modern Go practices).

## Setup

1.  **Install Go**: Ensure Go is installed on your system. You can download it from [golang.org](https://golang.org/).
2.  **Get the Code**:
    * Save the main program code (the Go script that includes `func main()`, `func decodeRawData()`, etc.) as `main.go` in a new directory.
    * If you have the test code (the Go script that includes `func TestDecodeRawData()`, etc.), save it as `main_test.go` in the same directory.

## How to Obtain the cURL Command

This program is designed to work with cURL commands, often for replaying or analyzing network requests. You can typically obtain these commands directly from your web browser's developer tools:

1.  **Open Developer Tools**: In your browser (Chrome, Firefox, Edge, Safari), open the Developer Tools. This is usually done by pressing `F12`, or by right-clicking on a webpage element and selecting "Inspect" or "Inspect Element".
2.  **Navigate to the Network Tab**: Within the Developer Tools, find and select the "Network" tab.
3.  **Trigger the Request**: Perform the network request you're interested in. This might involve loading a webpage, submitting a form, or an action that triggers an API call. You should see the request appear in the Network tab's log.
4.  **Find and Copy the Request**:
    * Locate the specific network request in the list.
    * Right-click on the request.
    * Look for an option like:
        * "Copy" -> "Copy as cURL (bash)" (Chrome, Edge)
        * "Copy Value" -> "Copy as cURL" (Firefox - select the "bash" or "cmd" option if available, often it defaults to a suitable format)
        * "Copy as cURL" (Safari)
          The exact wording might vary slightly between browsers and their versions. Choose the option that provides a "bash" or POSIX-compatible cURL command if multiple are offered.
5.  **Paste into Input File**: Paste the copied cURL command into a plain text file (e.g., `curl_command.txt`). This file will serve as the input to this Go program.

The program expects the `--data-raw $'(...)'` format for the data payload, which is a common output for the "Copy as cURL (bash)" option when the request contains data.

## Usage

There are two main ways to use this program: by running a pre-compiled binary or by building and running from the source code.

### Using a Pre-compiled Binary

If you have downloaded a pre-compiled binary (e.g., `cURLDataExtractor` or `cURLDataExtractor.exe` from the GitHub releases page for your operating system):

1.  **Download**: Download the appropriate executable for your system.
2.  **Place it**: Put the executable in a directory of your choice. You might want to add this directory to your system's PATH for easier access from any terminal location, but this is optional.
3.  **(For Linux/macOS) Make it executable**: If you are on Linux or macOS, you may need to give the downloaded file execute permissions. Open your terminal, navigate to the directory where you saved the binary, and run:
    ```bash
    chmod +x cURLDataExtractor
    ```
    *(Replace `cURLDataExtractor` with the actual downloaded file name if it's different.)*

4.  **Run from Terminal**: Open your terminal. If the directory containing the executable is not in your system's PATH, you'll need to navigate to it first (`cd /path/to/directory`).

    * **To run with default file names** (expecting `curl_command.txt` in the same directory as the executable or the current working directory, and outputting `decoded_curl_command.txt`):

      On **Linux/macOS**:
        ```bash
        ./cURLDataExtractor
        ```
      On **Windows** (Command Prompt or PowerShell):
        ```cmd
        cURLDataExtractor.exe
        ```
      (If using PowerShell and the executable is in the current directory, you might need `.\cURLDataExtractor.exe`)

    * **To run with specified input and output files**:

      On **Linux/macOS**:
        ```bash
        ./cURLDataExtractor -input path/to/your_curl_file.txt -output path/to/processed_data.json
        ```
      On **Windows** (Command Prompt or PowerShell):
        ```cmd
        cURLDataExtractor.exe -input path\to\your_curl_file.txt -output path\to\processed_data.json
        ```

### Building and Running from Source

If you prefer to build from the Go source code (as described in the "Setup" section):

1.  **Build the Executable** (optional, you can also run directly with `go run`):
    Open your terminal, navigate to the directory where you saved `main.go`, and run:
    ```bash
    go build main.go
    ```
    This will create an executable file named `main` (or `main.exe` on Windows) in that directory.

2.  **Run the Program** (from the source directory):

    * Using default file names:
        ```bash
        ./main 
        # or, to run without building first:
        # go run main.go
        ```

    * Specifying input and output files:
        ```bash
        ./main -input your_curl_file.txt -output processed_data.json
        # or, to run without building first:
        # go run main.go -input your_curl_file.txt -output processed_data.json
        ```

---
**Command-Line Flags** (apply whether using a pre-compiled binary or running from source):

* `-input <filepath>`: Path to the input file containing the cURL command. (Default: `curl_command.txt`)
* `-output <filepath>`: Path to the output file where the decoded JSON will be saved. (Default: `decoded_curl_command.txt`)
## Input File Format

The input file (e.g., `curl_command.txt`) should be a plain text file containing a single, complete cURL command, typically copied from browser developer tools as described above. The program specifically looks for the `--data-raw $'(...)'` argument.

**Example `curl_command.txt`:**

```bash
curl '[https://api.example.com/submit](https://api.example.com/submit)' \
  -H 'Accept: application/json' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  --data-raw $'field1=value1&field2=\\u0048ello\\x20World%21%0AThis\\x20is\\x20a\\x20test\\nNextLineHere%0D%0A%5COctal\\101' \
  --compressed
```
(Note: The actual data inside $'...' would typically be more complex, potentially gzipped, and representing a JSON structure after decoding and decompression).

## Output File Format

The output file (e.g., `decoded_curl_command.txt`) will contain the final processed data, which is expected to be JSON, pretty-printed with an indent of 2 spaces.

**Example `decoded_curl_command.txt` (if the input resolved to this JSON):**

```json
{
  "key": "value",
  "message": "Hello World!",
  "details": [
    "item1",
    "item2"
  ]
} 
```
## Workflow

The program performs the following steps:

1.  **Parses Flags**: Reads command-line flags for input and output file paths.
2.  **Reads Input**: Reads the cURL command from the specified input file.
3.  **Extracts Raw Data**: Uses a regular expression to find and extract the content within `--data-raw $'(...)`.
4.  **Trims Whitespace**: Removes any leading or trailing whitespace from the extracted raw data string.
5.  **Decodes Data**:
    * Processes the extracted string, interpreting escape sequences (`\n`, `\xHH`, `\uHHHH`, octal, etc.).
    * Ensures that all decoded characters and Unicode escapes fall within the Latin-1 range (U+0000-U+00FF).
6.  **Decompresses Data**: Attempts to decompress the resulting byte slice using Gzip. If the data is not Gzipped, this step will likely fail, and an error will be logged (the program expects Gzipped data at this stage if decompression is needed).
7.  **Parses JSON**: Unmarshals the (potentially decompressed) byte slice into a generic JSON structure.
8.  **Pretty-Prints JSON**: Marshals the JSON structure back into a byte slice with indentation for readability.
9.  **Writes Output**: Saves the pretty-printed JSON to the specified output file.
10. **Logging**: Provides logs about the files being used and key steps/errors during processing.
## Testing

If you have the `main_test.go` file alongside `main.go`:

1.  Navigate to the project directory in your terminal.
2.  Run the tests using the command:
    ```bash
    go test
    ```
3.  For more detailed output, use:
    ```bash
    go test -v
    ```
4.  To check test coverage:
    ```bash
    go test -cover
    ```