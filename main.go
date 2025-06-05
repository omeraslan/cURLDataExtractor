package main

import (
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"flag" // Added for command-line flag parsing
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var CoderName string

// reprBytes represents byte slices similarly to Python's b” notation.
func reprBytes(b []byte) string {
	var sb strings.Builder
	sb.WriteString("b'")
	for _, B := range b {
		if B >= 32 && B < 127 && B != '\'' && B != '\\' { // Printable ASCII, not ' or \
			sb.WriteByte(B)
		} else {
			switch B {
			case '\n':
				sb.WriteString("\\n")
			case '\r':
				sb.WriteString("\\r")
			case '\t':
				sb.WriteString("\\t")
			case '\'':
				sb.WriteString("\\'")
			case '\\':
				sb.WriteString("\\\\")
			default:
				sb.WriteString(fmt.Sprintf("\\x%02x", B))
			}
		}
	}
	sb.WriteString("'")
	return sb.String()
}

// extractDataRaw extracts the --data-raw content from a cURL command.
func extractDataRaw(curlCommand string) (string, error) {
	// Python: re.search(r"--data-raw \$'(.*)'", curl_command, re.DOTALL)
	// In Go, the (?s) flag is equivalent to re.DOTALL. \$ matches the literal $ character.
	// The single quotes in \$' are literal characters.
	re := regexp.MustCompile(`(?s)--data-raw \$'(.*)'`)
	matches := re.FindStringSubmatch(curlCommand)
	if len(matches) < 2 {
		return "", fmt.Errorf("failed to extract data-raw part")
	}
	return matches[1], nil
}

// decompressGzipData decompresses gzip-compressed byte data.
func decompressGzipData(data []byte) ([]byte, error) {
	reader := bytes.NewReader(data)
	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, fmt.Errorf("decompressGzipData: failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	decompressedData, err := io.ReadAll(gzReader)
	if err != nil {
		return nil, fmt.Errorf("decompressGzipData: failed to decompress data: %w", err)
	}
	return decompressedData, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// if they represent codepoints within that range.
// decodeRawData converts an escaped string into a byte slice, mimicking Python's
// `data.encode('latin1').decode('unicode_escape').encode('latin1')` behavior.
func decodeRawData(s string) ([]byte, error) {
	var result bytes.Buffer
	inputBytes := []byte(s) // Work with the raw bytes of the input string
	i := 0                  // Current index in inputBytes

	for i < len(inputBytes) {
		if inputBytes[i] == '\\' {
			// This is the start of an escape sequence
			i++ // Move past '\'
			if i >= len(inputBytes) {
				return nil, fmt.Errorf("decodeRawData: trailing backslash")
			}

			escapeCode := inputBytes[i] // The character determining the escape type

			switch escapeCode {
			case 'n':
				result.WriteByte('\n')
				i++
			case 'r':
				result.WriteByte('\r')
				i++
			case 't':
				result.WriteByte('\t')
				i++
			case 'b':
				result.WriteByte('\b')
				i++
			case 'f':
				result.WriteByte('\f')
				i++
			case 'v':
				result.WriteByte('\v')
				i++
			case 'a':
				result.WriteByte('\a')
				i++
			case '\\':
				result.WriteByte('\\')
				i++
			case '\'':
				result.WriteByte('\'')
				i++
			case '"':
				result.WriteByte('"')
				i++
			case 'x':
				i++                         // Move past 'x'
				if i+1 >= len(inputBytes) { // Need two hex digits (inputBytes[i] and inputBytes[i+1])
					return nil, fmt.Errorf("decodeRawData: incomplete hex escape \\x (need 2 digits, got: %q)", string(inputBytes[i:]))
				}
				val, err := hex.DecodeString(string(inputBytes[i : i+2]))
				if err != nil {
					return nil, fmt.Errorf("decodeRawData: invalid hex escape \\x%s: %w", string(inputBytes[i:i+2]), err)
				}
				result.WriteByte(val[0])
				i += 2 // Consumed two hex digits
			case 'u':
				i++                         // Move past 'u'
				if i+3 >= len(inputBytes) { // Need four hex digits
					return nil, fmt.Errorf("decodeRawData: incomplete unicode escape \\u (need 4 digits, got: %q)", string(inputBytes[i:]))
				}
				code, err := strconv.ParseInt(string(inputBytes[i:i+4]), 16, 32)
				if err != nil {
					return nil, fmt.Errorf("decodeRawData: invalid unicode escape \\u%s: %w", string(inputBytes[i:i+4]), err)
				}
				if code < 0 || code > 0xFF { // Python's .encode('latin1') constraint
					return nil, fmt.Errorf("decodeRawData: unicode escape \\u%04X (codepoint %d) is outside Latin-1 range (U+0000-U+00FF)", code, code)
				}
				result.WriteByte(byte(code))
				i += 4 // Consumed four hex digits
			case 'U':
				i++                         // Move past 'U'
				if i+7 >= len(inputBytes) { // Need eight hex digits
					return nil, fmt.Errorf("decodeRawData: incomplete unicode escape \\U (need 8 digits, got: %q)", string(inputBytes[i:]))
				}
				code, err := strconv.ParseInt(string(inputBytes[i:i+8]), 16, 32)
				if err != nil {
					return nil, fmt.Errorf("decodeRawData: invalid unicode escape \\U%s: %w", string(inputBytes[i:i+8]), err)
				}
				if code < 0 || code > 0xFF { // Python's .encode('latin1') constraint
					return nil, fmt.Errorf("decodeRawData: unicode escape \\U%08X (codepoint %d) is outside Latin-1 range (U+0000-U+00FF)", code, code)
				}
				result.WriteByte(byte(code))
				i += 8 // Consumed eight hex digits
			case '0', '1', '2', '3', '4', '5', '6', '7':
				startOctalParseIndex := i // Position of the first octal digit (after '\')

				var octalDigitsBytes []byte
				// Greedily parse up to 3 octal digits
				for len(octalDigitsBytes) < 3 && i < len(inputBytes) && inputBytes[i] >= '0' && inputBytes[i] <= '7' {
					octalDigitsBytes = append(octalDigitsBytes, inputBytes[i])
					i++ // Consume the octal digit for the next iteration or for the check below
				}
				// After this loop, 'i' points to the character *after* the consumed octal sequence.
				// octalDigitsBytes contains the sequence like ['0'], or ['7','7']

				// Python's 'unicode_escape' is strict: if an octal sequence
				// (even a single digit like \0) is followed by any other digit (0-9),
				// it's an invalid octal escape. E.g., \08 or \79 are errors.
				if i < len(inputBytes) && inputBytes[i] >= '0' && inputBytes[i] <= '9' {
					// A non-octal digit followed the consumed octal digits. This is an error.
					// Reconstruct the problematic sequence for the error message.
					// It starts from what was originally `escapeCode` up to and including the offending digit.
					problematicSequence := string(inputBytes[startOctalParseIndex-1 : i+1]) // -1 to include escapeCode itself for display
					if escapeCode >= '0' && escapeCode <= '7' {                             // ensure escapeCode was an octal digit
						problematicSequence = string(inputBytes[startOctalParseIndex : i+1])
					}

					return nil, fmt.Errorf("decodeRawData: invalid octal escape \\%s", problematicSequence)
				}

				if len(octalDigitsBytes) == 0 {
					// This should not happen if we entered based on '0'-'7'
					return nil, fmt.Errorf("decodeRawData: internal error: no octal digits found where expected")
				}

				octalString := string(octalDigitsBytes)
				val, err := strconv.ParseInt(octalString, 8, 16)
				if err != nil {
					// This should also be unlikely if the above logic correctly captures octal digits.
					return nil, fmt.Errorf("decodeRawData: failed to parse octal string \\%s: %w", octalString, err)
				}
				if val > 0xFF {
					return nil, fmt.Errorf("decodeRawData: octal escape \\%s (value %d) is too large for a byte", octalString, val)
				}
				result.WriteByte(byte(val))
				// 'i' is already advanced past the consumed octal digits.
			default: // Unrecognized escape after '\'
				result.WriteByte('\\')       // Write the backslash literally
				result.WriteByte(escapeCode) // Write the character that followed the backslash
				i++                          // Consumed the escapeCode character
			}
		} else {
			// Not a backslash. This is a literal character (or start of one).
			// Decode the rune and its size from inputBytes starting at current 'i'.
			r, size := utf8.DecodeRune(inputBytes[i:])

			if r == utf8.RuneError && size == 1 {
				return nil, fmt.Errorf("decodeRawData: invalid UTF-8 sequence for a literal character at byte index %d", i)
			}

			if r <= 0xFF { // Mimic Python's char.encode('latin1') behavior for the rune
				result.WriteByte(byte(r))
			} else {
				return nil, fmt.Errorf("decodeRawData: literal character U+%04X ('%c') is outside Latin-1 range (U+0000-U+00FF) and was not escaped", r, r)
			}
			i += size // Advance by the number of bytes in the decoded rune
		}
	}
	return result.Bytes(), nil
}

func main() {
	defaultInputFile := "curl_command.txt"
	defaultOutputFile := "decoded_curl_command.txt" // As per your request for the output filename

	// Define command-line flags
	inputFile := flag.String("input", defaultInputFile, "Path to the input cURL command file.")
	outputFile := flag.String("output", defaultOutputFile, "Path to the output file for the decoded data.")
	flag.Parse() // Parse the command-line flags

	// Log input file usage
	log.Printf("Using input file: %s", *inputFile)
	if *inputFile == defaultInputFile {
		isInputSetByUser := false
		flag.Visit(func(f *flag.Flag) {
			if f.Name == "input" {
				isInputSetByUser = true
			}
		})
		if !isInputSetByUser {
			log.Println("(This is the default input path as no -input flag was provided)")
		}
	}

	// Log output file usage
	log.Printf("Using output file: %s", *outputFile)
	if *outputFile == defaultOutputFile {
		isOutputSetByUser := false
		flag.Visit(func(f *flag.Flag) {
			if f.Name == "output" {
				isOutputSetByUser = true
			}
		})
		if !isOutputSetByUser {
			log.Println("(This is the default output path as no -output flag was provided)")
		}
	}

	// Read the cURL command from the specified input file
	curlCommandBytes, err := os.ReadFile(*inputFile)
	if err != nil {
		log.Fatalf("Error reading input file %s: %v", *inputFile, err)
	}
	curlCommand := string(curlCommandBytes)

	// Extract the data-raw part
	dataRaw, err := extractDataRaw(curlCommand)
	if err != nil {
		log.Fatalf("Error during extraction: %v", err)
	}

	// !!! ADDEDWhitespaceTrimming !!!
	// Remove leading/trailing whitespace from the extracted data-raw content
	// This handles cases like $' \u001f...' where a leading space can corrupt the gzip stream.
	originalExtractedLength := len(dataRaw)
	dataRaw = strings.TrimSpace(dataRaw)
	if len(dataRaw) != originalExtractedLength {
		log.Printf("Trimmed whitespace from extracted data-raw content. Original length: %d, New length: %d", originalExtractedLength, len(dataRaw))
	}
	// !!! End of ADDEDWhitespaceTrimming !!!

	fmt.Println("Extracted data-raw part (first 100 characters, after trim):") // Log message updated
	if len(dataRaw) > 100 {
		fmt.Printf("%q\n", dataRaw[:100])
	} else {
		fmt.Printf("%q\n", dataRaw)
	}

	// Decode the raw data
	decodedData, err := decodeRawData(dataRaw)
	if err != nil {
		log.Fatalf("Error during decoding raw data: %v", err)
	}
	fmt.Println("Decoded data (first 100 bytes):")
	if len(decodedData) > 100 {
		fmt.Println(reprBytes(decodedData[:100]))
	} else {
		fmt.Println(reprBytes(decodedData))
	}

	// *** DECOMPRESSION LOGIC MODIFICATION START ***
	// Check if the data *might* be gzip compressed by looking at the Content-Encoding header
	// This is a common way to determine if decompression is needed.
	// For this specific cURL, we know it's not gzipped, so we'll just skip the decompression.
	// In a more general solution, you'd parse headers to make this decision.
	// For now, we'll assume if it's not JSON, it's URL-encoded form data.

	var finalProcessedData []byte

	// You could add logic here to check for 'Content-Encoding: gzip' header in the curl command.
	// For this specific problem, we know it's not gzipped.

	// Try to decompress only if it seems like gzipped data
	// A simple heuristic (not foolproof) is to check for gzip magic bytes (0x1f 0x8b)
	if len(decodedData) >= 2 && decodedData[0] == 0x1f && decodedData[1] == 0x8b {
		log.Println("Detected potential gzip header. Attempting decompression.")
		decompressedData, err := decompressGzipData(decodedData)
		if err != nil {
			// Log the error but don't fatally exit, in case it's not gzip after all.
			log.Printf("Warning: Decompression failed, data might not be gzipped or is corrupted: %v", err)
			finalProcessedData = decodedData // Use original data if decompression fails
		} else {
			finalProcessedData = decompressedData

			fmt.Println("Decompressed data (first 100 bytes):")
			if len(finalProcessedData) > 100 {
				fmt.Println(reprBytes(finalProcessedData[:100]))
			} else {
				fmt.Println(reprBytes(finalProcessedData))
			}
		}
	} else {
		log.Println("Data does not appear to be gzip compressed (missing magic bytes). Skipping decompression.")
		finalProcessedData = decodedData // Use the decoded data directly
	}
	// *** DECOMPRESSION LOGIC MODIFICATION END ***

	// Convert the processed data to a string (assuming UTF-8, as in the Python script)
	// If it was gzipped, this is the decompressed string.
	// If not gzipped, this is the raw decoded string.
	processedString := string(finalProcessedData)
	fmt.Println("Processed string (first 100 characters):")
	if len(processedString) > 100 {
		fmt.Printf("%q\n", processedString[:100])
	} else {
		fmt.Printf("%q\n", processedString)
	}

	// For URL-encoded data, we typically don't parse it as JSON directly.
	// We would instead parse it using net/url.ParseQuery.
	// Since your original code assumed JSON, we'll add a check.

	// Try to parse as JSON. If it fails, treat it as plain text or URL-encoded.
	var jsonData interface{} // To accept any valid JSON structure
	err = json.Unmarshal([]byte(processedString), &jsonData)
	if err != nil {
		log.Printf("Warning: Data is not valid JSON, treating as plain text or URL-encoded: %v", err)
		// If it's not JSON, write the raw processed string to the output file.
		// Or you could add logic here to specifically parse URL-encoded data.

		fmt.Println("Saving raw processed string to output file.")
		err = os.WriteFile(*outputFile, finalProcessedData, 0644)
		if err != nil {
			log.Fatalf("Error saving processed data to file %s: %v", *outputFile, err)
		}
		fmt.Printf("Processed data (not JSON) has been saved to %s\n", *outputFile)
		return // Exit the program here if it's not JSON
	}

	// Pretty-print the JSON data (like indent=2 in Python)
	prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		log.Fatalf("Error marshalling JSON to pretty format: %v", err)
	}
	fmt.Println("Parsed JSON data:")
	fmt.Println(string(prettyJSON))

	// Save the pretty JSON data to the specified output file
	err = os.WriteFile(*outputFile, prettyJSON, 0644) // 0644 gives read/write to owner, read to others
	if err != nil {
		log.Fatalf("Error saving decoded data to file %s: %v", *outputFile, err)
	}
	fmt.Printf("Decoded data has been saved to %s\n", *outputFile)
}
