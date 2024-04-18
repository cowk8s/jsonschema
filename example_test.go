package jsonschema_test

import (
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func Example_fromFiles() {
	schemaFile := "./testdata/examples/schema.json"
	instanceFile := "./testdata/examples/instance.json"

	c := jsonschema.NewCompiler()
	sch, err := c.Compile(schemaFile)
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Open(instanceFile)
	if err != nil {
		log.Fatal(err)
	}
	inst, err := jsonschema.UnmarshalJSON(f)
	if err != nil {
		log.Fatal(err)
	}

	err = sch.Validate(inst)
	fmt.Println("valid:", err == nil)
	// Output:
	// valid: true
}

func Example_fromStrings() {
	catSchema, err := jsonschema.UnmarshalJSON(strings.NewReader(`{
        "type": "object",
        "properties": {
            "speak": { "const": "meow" }
        },
        "required": ["speak"]
    }`))
	if err != nil {
		log.Fatal(err)
	}
	// note that dog.json is loaded from file ./testdata/examples/dog.json
	petSchema, err := jsonschema.UnmarshalJSON(strings.NewReader(`{
        "oneOf": [
            { "$ref": "dog.json" },
            { "$ref": "cat.json" }
        ]
    }`))
	if err != nil {
		log.Fatal(err)
	}
	inst, err := jsonschema.UnmarshalJSON(strings.NewReader(`{"speak": "bow"}`))
	if err != nil {
		log.Fatal(err)
	}

	c := jsonschema.NewCompiler()
	if err := c.AddResource("./testdata/examples/cat.json", catSchema); err != nil {
		log.Fatal(err)
	}
	if err := c.AddResource("./testdata/examples/pet.json", petSchema); err != nil {
		log.Fatal(err)
	}
	sch, err := c.Compile("./testdata/examples/pet.json")
	if err != nil {
		log.Fatal(err)
	}
	err = sch.Validate(inst)
	fmt.Println("valid:", err == nil)
	// Output:
	// valid: true
}

type HTTPURLLoader struct{}

func (l HTTPURLLoader) Load(url string) (any, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("%s returned status code %d", url, resp.StatusCode)
	}
	defer resp.Body.Close()

	return jsonschema.UnmarshalJSON(resp.Body)
}

func Example_fromHTTPS() {
	schemaURL := "https://raw.githubusercontent.com/santhosh-tekuri/boon/main/tests/examples/schema.json"
	instanceFile := "./testdata/examples/instance.json"

	loader := jsonschema.SchemeURLLoader{
		"file":  jsonschema.FileLoader{},
		"http":  HTTPURLLoader{},
		"https": HTTPURLLoader{},
	}

	c := jsonschema.NewCompiler()
	c.UseLoader(loader)
	sch, err := c.Compile(schemaURL)
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Open(instanceFile)
	if err != nil {
		log.Fatal(err)
	}
	inst, err := jsonschema.UnmarshalJSON(f)
	if err != nil {
		log.Fatal(err)
	}

	err = sch.Validate(inst)
	fmt.Println("valid:", err == nil)
	// Output:
	// valid: true
}

func Example_customFormat() {
	validatePalindrome := func(v any) error {
		s, ok := v.(string)
		if !ok {
			return nil
		}
		var runes []rune
		for _, r := range s {
			runes = append(runes, r)
		}
		for i, j := 0, len(runes)-1; i <= j; i, j = i+1, j-1 {
			if runes[i] != runes[j] {
				return fmt.Errorf("no match for rune at %d", i)
			}
		}
		return nil
	}

	schema, err := jsonschema.UnmarshalJSON(strings.NewReader(`{"type": "string", "format": "palindrome"}`))
	if err != nil {
		log.Fatal(err)
	}
	inst := "hello world"

	c := jsonschema.NewCompiler()
	c.RegisterFormat(&jsonschema.Format{
		Name:     "palindrome",
		Validate: validatePalindrome,
	})
	c.AssertFormat()
	if err := c.AddResource("schema.json", schema); err != nil {
		log.Fatal(err)
	}
	sch, err := c.Compile("schema.json")
	if err != nil {
		log.Fatal(err)
	}
	err = sch.Validate(inst)
	fmt.Println("valid:", err == nil)
	// Output:
	// valid: false
}

// Example_customContentEncoding shows how to define
// "hex" contentEncoding.
func Example_customContentEndocing() {
	schema, err := jsonschema.UnmarshalJSON(strings.NewReader(`{"type": "string", "contentEncoding": "hex"}`))
	if err != nil {
		log.Fatal(err)
	}
	inst := "abcxyz"

	c := jsonschema.NewCompiler()
	c.RegisterContentEncoding(&jsonschema.Decoder{
		Name:   "hex",
		Decode: hex.DecodeString,
	})
	c.AssertContent()
	if err := c.AddResource("schema.json", schema); err != nil {
		log.Fatal(err)
	}
	sch, err := c.Compile("schema.json")
	if err != nil {
		log.Fatal(err)
	}
	err = sch.Validate(inst)
	fmt.Println("valid:", err == nil)
	// Output:
	// valid: false
}

// Example_customContentMediaType shows how to define
// "application/xml" contentMediaType.
func Example_customContentMediaType() {
	schema, err := jsonschema.UnmarshalJSON(strings.NewReader(`{"type": "string", "contentMediaType": "application/xml"}`))
	if err != nil {
		log.Fatal(err)
	}
	inst := "<abc></def>"

	c := jsonschema.NewCompiler()
	c.RegisterContentMediaType(&jsonschema.MediaType{
		Name: "application/xml",
		Validate: func(b []byte) error {
			return xml.Unmarshal(b, new(any))
		},
		UnmarshalJSON: nil, // xml is not json-compatiable format
	})
	c.AssertContent()
	if err := c.AddResource("schema.json", schema); err != nil {
		log.Fatal(err)
	}
	sch, err := c.Compile("schema.json")
	if err != nil {
		log.Fatal(err)
	}
	err = sch.Validate(inst)
	fmt.Println("valid:", err == nil)
	// Output:
	// valid: false
}

type dclarkRegexp regexp2.Regexp

func (re *dclarkRegexp) MatchString(s string) bool {
	matched, err := (*regexp2.Regexp)(re).MatchString(s)
	return err == nil && matched
}

func (re *dclarkRegexp) String() string {
	return (*regexp2.Regexp)(re).String()
}

func dlclarkCompile(s string) (jsonschema.Regexp, error) {
	re, err := regexp2.Compile(s, regexp2.ECMAScript)
	if err != nil {
		return nil, err
	}
	return (*dclarkRegexp)(re), nil
}

func Example_customRegexEngine() {
	// golang regexp does not support escape sequence: `\c`
	schema, err := jsonschema.UnmarshalJSON(strings.NewReader(`{
		"type": "string",
		"pattern": "^\\cc$"
	}`))
	if err != nil {
		log.Fatal(err)
	}
	inst, err := jsonschema.UnmarshalJSON(strings.NewReader(`"\u0003"`))
	if err != nil {
		log.Fatal(err)
	}

	c := jsonschema.NewCompiler()
	c.UseRegexpEngine(dlclarkCompile)
	if err := c.AddResource("schema.json", schema); err != nil {
		log.Fatal(err)
	}

	sch, err := c.Compile("schema.json")
	if err != nil {
		log.Fatal(err)
	}
	err = sch.Validate(inst)
	fmt.Println("valid:", err == nil)
	// Output:
	// valid: true
}
