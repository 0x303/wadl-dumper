package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/antchfx/xmlquery"
)

// Custom flag type to handle multiple -p flags
type placeholderFlag []string

func (p *placeholderFlag) String() string {
	return fmt.Sprint(*p)
}

func (p *placeholderFlag) Set(value string) error {
	*p = append(*p, value)
	return nil
}

type options struct {
	baseURL, input, replace string
	showBase                bool
	placeholders            map[string]string
	placeholderArgs         placeholderFlag
}

var opt *options

func init() {
	opt = &options{
		placeholders: make(map[string]string),
	}

	flag.StringVar(&opt.input, "i", "", "")
	flag.StringVar(&opt.input, "input", "", "")

	flag.BoolVar(&opt.showBase, "b", false, "")
	flag.BoolVar(&opt.showBase, "show-base", false, "")

	flag.StringVar(&opt.replace, "r", "", "")
	flag.StringVar(&opt.replace, "replace", "", "")

	// Add custom flag for placeholders
	flag.Var(&opt.placeholderArgs, "p", "Specify placeholder value (format: name=value)")
	flag.Var(&opt.placeholderArgs, "placeholder", "Specify placeholder value (format: name=value)")

	flag.Usage = func() {
		h := []string{
			"Usage:",
			"  wadl-dumper -i http://domain.tld/application.wadl [options...]",
			"  wadl-dumper -i /path/to/wadl.xml --show-base -r \"-alert(1)-\"",
			"  wadl-dumper -i /path/to/wadl.xml -p slug=myslug -p projectId=test123",
			"",
			"Options:",
			"  -i, --input <URL/FILE>         URL/path to WADL file",
			"  -b, --show-base                Add base URL to paths",
			"  -r, --replace <string>         Replace all unspecified placeholders with given value",
			"  -p, --placeholder <name=value> Replace specific placeholder with given value (can be used multiple times)",
			"  -h, --help                     Show its help text",
			"",
		}

		fmt.Fprint(os.Stderr, strings.Join(h, "\n"))
	}
}

func parsePlaceholders() {
	for _, p := range opt.placeholderArgs {
		parts := strings.SplitN(p, "=", 2)
		if len(parts) == 2 {
			name, value := parts[0], parts[1]
			opt.placeholders[name] = value
		}
	}
}

func errorExit(message string) {
	err := fmt.Sprintf("Error! %s\n", message)
	fmt.Fprint(os.Stderr, err)
	os.Exit(1)
}

func replaceNth(s, old string, new string, n int) string {
	i := 0

	for m := 1; m <= n; m++ {
		x := strings.Index(s[i:], old)
		if x < 0 {
			break
		}
		i += x
		if m == n {
			return s[:i] + new + s[i+len(old):]
		}
		i += len(old)
	}

	return s
}

// replacePlaceholders replaces placeholders in the path with values from the placeholders map
// or with the default replace value if specified
func replacePlaceholders(path string) string {
	// Regex to find placeholders in the format {name}
	re := regexp.MustCompile(`\{([^{}]+)\}`)

	// Use a replacement function to handle each match
	result := re.ReplaceAllStringFunc(path, func(match string) string {
		// Extract the placeholder name without braces
		name := match[1 : len(match)-1]

		// Check if we have a specific value for this placeholder
		if value, exists := opt.placeholders[name]; exists {
			return value
		}

		// If no specific value but we have a default replace value, use that
		if opt.replace != "" {
			return opt.replace
		}

		// Otherwise, leave the placeholder as is
		return match
	})

	return result
}

func main() {
	flag.Parse()
	parsePlaceholders()

	var path string
	var wadl *xmlquery.Node

	if opt.input == "" {
		errorExit("Flag -i is required, use -h flag for help.")
	}

	if strings.HasPrefix(opt.input, "http") {
		wadl, _ = xmlquery.LoadURL(opt.input)
	} else {
		f, err := os.Open(opt.input)
		if err != nil {
			errorExit(fmt.Sprintf("Can't open '%s' file.", opt.input))
		}

		wadl, _ = xmlquery.Parse(f)
	}

	if wadl == nil {
		errorExit("Can't parse WADL file.")
	}

	xmlns := xmlquery.FindOne(wadl, "//application/@xmlns")
	if xmlns == nil || !strings.Contains(xmlns.InnerText(), "wadl.dev.java.net") {
		errorExit("Not a WADL file.")
	}

	base := xmlquery.FindOne(wadl, "//resources/@base")
	if base != nil && opt.showBase {
		opt.baseURL = base.InnerText()
	} else {
		opt.baseURL = ""
	}

	for _, paths := range xmlquery.Find(wadl, "//resource/@path") {
		path = opt.baseURL + paths.InnerText()

		// Apply placeholder replacements
		path = replacePlaceholders(path)

		if opt.baseURL != "" {
			path = replaceNth(path, "//", "/", 2)
		}

		fmt.Printf("%s\n", path)
	}
}
