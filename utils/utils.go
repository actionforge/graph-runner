package utils

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/joho/godotenv"
)

var envFileOnce sync.Once

// Inline-if alternative in Go. Example:
// e ? a : b becomes If(e, a, b)
func If[E bool, T any](exp E, a T, b T) T {
	if exp {
		return a
	} else {
		return b
	}
}

func LoadEnvOnce() {
	envFileOnce.Do(func() {
		loadEnvFile := os.Getenv("LOAD_ENV_FILE")
		if loadEnvFile != "" {
			_ = godotenv.Load()
		}
	})
}

func NormalizeLineEndings(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

func AnyToString(value any) string {
	switch v := value.(type) {
	case int, int8, int16, int32, int64:
		return strconv.FormatInt(reflect.ValueOf(v).Int(), 10)
	case uint, uint8, uint16, uint32, uint64:
		return strconv.FormatUint(reflect.ValueOf(v).Uint(), 10)
	case float32, float64:
		return strconv.FormatFloat(reflect.ValueOf(v).Float(), 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func InitMapAndSliceInStructRecursively(v reflect.Value) {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)

		if !field.CanSet() {
			continue
		}

		k := field.Kind()

		if k == reflect.Struct {
			InitMapAndSliceInStructRecursively(field)
			continue
		}

		if k == reflect.Map && field.IsNil() {
			fieldType := field.Type()
			newMap := reflect.MakeMap(fieldType)

			field.Set(newMap)
		} else if k == reflect.Slice && field.IsNil() {
			fieldType := field.Type()
			newSlice := reflect.MakeSlice(fieldType, 0, 0)

			field.Set(newSlice)
		}
	}
}

// GetItem retrieves an item or subitem from a map.
// Especially used to retrieve items from yaml or json interface maps.
func GetItem[T any](i map[any]any, attrs ...string) (T, error) {
	var (
		exists bool
		v      T
	)

	attr := attrs[0]

	if len(attrs) > 1 {
		// If there are more than one attribute requested,
		// traverse the map until the last map is reached.
		// 'foo', 'bar', 'bas' -> i['foo']['bar']
		for _, attr := range attrs[:len(attrs)-1] {
			i, exists = i[attr].(map[any]any)
			if !exists {
				return v, fmt.Errorf("executions is not a map")
			}
		}

		// 'bas' is the last attribute retrieved afterwards
		attr = attrs[len(attrs)-1]
	}

	// Retrieve the last attribute
	// 'bas' -> i['bas']
	c, exists := i[attr]
	if !exists {
		return v, fmt.Errorf("%v is not a map", attrs[0])
	}

	// The item exists, but is undefined. E.g:
	// connections:\n
	// In this case, connections has no type, so it is undefined.
	if c == nil {
		return v, nil
	}

	v, exists = c.(T)
	if !exists {
		return v, fmt.Errorf(" is not of type %T", v)
	}

	return v, nil
}

type GetVariableOpts struct {
	Flag         bool
	Env          bool
	DockerSecret bool
	Optional     bool
}

func getEnvValue(parameterName, defaultValue string) string {
	value := os.Getenv(parameterName)
	if value == "" {
		return defaultValue
	}
	return value
}

func GetVariable(name, desc string, opts GetVariableOpts) string {

	value := ""

	if opts.Flag {
		flagValPtr := flag.String(name, "", desc)
		value = *flagValPtr
	}

	if value == "" && opts.Env {
		value = getEnvValue(strings.ToUpper(name), "")
	}

	if value == "" && opts.DockerSecret {
		value, _ = readDockerSecret(name)
	}

	if value == "" && !opts.Optional {
		log.Fatalf("%s not provided", name)
	}

	return value
}

func readDockerSecret(secretName string) (string, error) {
	secretBytes, err := os.ReadFile(fmt.Sprintf("/run/secrets/%s", secretName))
	if err != nil {
		return "", err
	}

	secret := strings.TrimSpace(string(secretBytes))
	return secret, nil
}

func FindProjectRoot() string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}
	return strings.Trim(string(output), " \r\n")
}
