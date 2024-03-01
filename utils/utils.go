package utils

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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

func DownloadFile(url string, dstFile string, cb func(contentLength int64) io.Writer) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	f, _ := os.OpenFile(dstFile, os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()

	var writer io.Writer
	if cb != nil {
		writer = cb(resp.ContentLength)
	}
	if writer != nil {
		_, err := io.Copy(io.MultiWriter(f, writer), resp.Body)
		if err != nil {
			return err
		}
	} else {
		_, err := io.Copy(f, resp.Body)
		if err != nil {
			return err
		}
	}
	return nil
}

func Unzip(zipFile, dstDir string) error {
	archive, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer archive.Close()

	for _, f := range archive.File {
		filePath := filepath.Join(dstDir, f.Name)

		if !strings.HasPrefix(filePath, filepath.Clean(dstDir)+string(os.PathSeparator)) {
			return err
		}
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
		if err != nil {
			return err
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		fileInArchive, err := f.Open()
		if err != nil {
			return err
		}

		_, err = io.Copy(dstFile, fileInArchive)
		if err != nil {
			panic(err)
		}

		dstFile.Close()
		fileInArchive.Close()
	}

	return nil
}

func Untar(tarGzFile, dstDir string) error {
	file, err := os.Open(tarGzFile)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tarReader := tar.NewReader(gzr)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		filePath := filepath.Join(dstDir, header.Name)
		if !strings.HasPrefix(filePath, filepath.Clean(dstDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path")
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
				return err
			}
			outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	return nil
}

func GetActionforgeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return filepath.Join(home, ".actionforge")
}

func GetSanitizedEnviron() []string {
	env := os.Environ()
	var sanitizedEnv []string
	for _, e := range env {
		if !strings.HasPrefix(e, "GRAPH_FILE=") &&
			!strings.HasPrefix(e, "INPUT_") {
			sanitizedEnv = append(sanitizedEnv, e)
		}
	}
	return sanitizedEnv
}
