package cmd

import (
	"actionforge/graph-runner/core"
	"actionforge/graph-runner/utils"
	_ "embed"
	"encoding/json"
	"io"

	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

//go:embed frozen.yml
var frozenGraph []byte

const ghZipBaseUrl = "https://github.com/actionforge/graph-runner/archive"
const goRegistryList = "https://go.dev/dl/?mode=json"
const goRegistry = "https://go.dev/dl"

func ExecuteFrozenGraph() error {
	return core.RunGraph(frozenGraph)
}

type GoFile struct {
	Filename string `json:"filename"`
	Os       string `json:"os"`
	Arch     string `json:"arch"`
	Version  string `json:"version"`
	Sha256   string `json:"sha256"`
	Size     int64  `json:"size"`
	Kind     string `json:"kind"`
}

type GoVersion struct {
	Version string   `json:"version"`
	Stable  bool     `json:"stable"`
	Files   []GoFile `json:"files"`
}

var cmdFreeze = &cobra.Command{
	Use:   "freeze [filename]",
	Short: `Freeze a graph file`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filename := args[0]
		output, _ := cmd.Flags().GetString("output")
		if runtime.GOOS == "windows" {
			output += ".exe"
		}

		absOutput, err := filepath.Abs(output)
		if err != nil {
			log.Fatal(err)
		}

		actionHomeDir := utils.GetActionforgeDir()

		err = os.MkdirAll(actionHomeDir, os.ModePerm)
		if err != nil {
			log.Fatal("Error creating temp dir")
		}

		repoDir, err := downloadAndExtractGraphRunner(actionHomeDir)
		if err != nil {
			log.Fatal(err)
		}

		goBin, err := downloadAndExtractGo(actionHomeDir)
		if err != nil {
			log.Fatal(err)
		}

		graph, err := os.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
		}

		frozenGraph := filepath.Join(repoDir, "cmd", "frozen.yml")
		err = os.WriteFile(frozenGraph, graph, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Building binary")
		c := exec.Command(goBin,
			"build",
			"-ldflags",
			"-X actionforge/graph-runner/core.FrozenGraph=true -X actionforge/graph-runner/core.Production=true",
			"-o",
			absOutput,
			repoDir,
		)
		c.Dir = repoDir
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		err = c.Run()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("🚀 Binary written to %s\n", output)
	},
}

func downloadAndExtractGraphRunner(dstDir string) (dir string, err error) {
	var (
		ref     string
		refName string
	)

	info, ok := core.GetBuildSettings()
	if core.IsProduction() && ok {
		ref = info["vcs.revision"]
		refName = ref
	} else {
		refName = "freeze"
		ref = "refs/heads/freeze"
	}

	cachePath := filepath.Join(dstDir, "cache")
	err = os.MkdirAll(cachePath, os.ModePerm)
	if err != nil {
		return "", err
	}

	repoZip := filepath.Join(cachePath, refName+".zip")
	_, err = os.Stat(repoZip)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}

		fmt.Printf("Downloading graph-runner from %s\n", ref)
		err := utils.DownloadFile(fmt.Sprintf("%s/%s.zip", ghZipBaseUrl, ref), repoZip, func(contentLength int64) io.Writer {
			return progressbar.DefaultBytes(contentLength)
		})
		if err != nil {
			return "", errors.New("Error downloading graph-runner")
		}
	}

	dir = filepath.Join(dstDir, fmt.Sprintf("graph-runner-%s", refName))
	_, err = os.Stat(dir)
	if err == nil {
		// Always remove existing directory to get a fresh copy
		err = os.RemoveAll(dir)
		if err != nil {
			return "",
				errors.New("Error removing existing graph-runner directory")
		}
	}

	fmt.Println("Unzipping graph-runner")
	err = utils.Unzip(repoZip, dstDir)
	if err != nil {
		return "", errors.New("Error unzipping graph-runner")
	}

	return dir, nil
}

func downloadAndExtractGo(dstDir string) (dir string, err error) {
	goBin := filepath.Join(dstDir, "go", "bin", "go")
	if runtime.GOOS == "windows" {
		goBin += ".exe"
	}

	_, err = os.Stat(goBin)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}

		fmt.Println("Getting go release info")
		var releases []GoVersion
		err = getJson(goRegistryList, &releases)
		if err != nil {
			return "", errors.New("Error getting go releases")
		}
		if len(releases) == 0 {
			return "", errors.New("No go releases found")
		}

		latest := releases[0]

		var goFile GoFile
		// find the latest go release for this architecture and OS
		for _, file := range latest.Files {
			if file.Os == runtime.GOOS && file.Arch == runtime.GOARCH {
				goFile = file
				break
			}
		}

		if goFile.Filename == "" {
			return "", errors.New("No go release found for this OS and architecture")
		}

		goZip := filepath.Join(dstDir, "cache", goFile.Filename)

		_, err = os.Stat(goZip)
		if err != nil {
			if !os.IsNotExist(err) {
				return "", err
			}

			downloadUrl := fmt.Sprintf("%s/%s", goRegistry, goFile.Filename)
			fmt.Printf("Downloading %s from %s\n", goFile.Version, downloadUrl)
			err = utils.DownloadFile(downloadUrl, goZip, func(contentLength int64) io.Writer {
				return progressbar.DefaultBytes(contentLength)
			})
			if err != nil {
				return "", errors.New("Error downloading graph-runner")
			}
		}

		if strings.HasSuffix(goFile.Filename, ".tar.gz") {
			fmt.Println("Untarring go")
			err = utils.Untar(goZip, dstDir)
			if err != nil {
				return "", errors.New("Error unzipping graph-runner")
			}
		} else if strings.HasSuffix(goFile.Filename, ".zip") {
			fmt.Println("Unzipping go")
			err = utils.Unzip(goZip, dstDir)
			if err != nil {
				return "", errors.New("Error unzipping graph-runner")
			}
		} else {
			return "", errors.New("Unknown file type")
		}
	}
	return goBin, nil
}

func getJson(url string, target interface{}) error {
	r, err := http.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}

func init() {
	cmdRoot.AddCommand(cmdFreeze)

	cmdFreeze.Flags().String("output", "", "The output path for the binary")
}
