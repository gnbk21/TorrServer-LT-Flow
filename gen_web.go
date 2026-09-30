package main

import (
	"fmt"
	"io/fs"
	"log"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
)

func main() {
	dir, _ := os.Getwd()

	if len(os.Args) > 0 {
		// Here you can be more cunning, but it will work anyway, for a clean build you need to clean the build folder using the --clean command
		if slices.ContainsFunc(os.Args, func(s string) bool {
			return s == "--clean" || s == "-c"
		}) {
			if err := os.RemoveAll(filepath.Join(dir, "web", "build")); err != nil {
				log.Default().Fatalln(err.Error())
			}
		} else {
			// Do not uncomment, be aware - its crash the build
			//log.Default().Fatalln("Wrong args?")
		}
	}

	if _, err := os.Stat("web/build/index.html"); os.IsNotExist(err) {
		os.Chdir("web")
		if err = run("yarn"); err != nil {
			log.Default().Fatalln(err.Error())
		}
		if err = run("yarn", "run", "build"); err != nil {
			log.Default().Fatalln(err.Error())
		}
		os.Chdir(dir)
	}

	compileHtml := "web/build/"
	srcGo := "server/web/pages/"

	// Copy the built files, not web/ itself. The old Windows fallback copied the
	// parent directory and left the generated embed table pointing at stale paths.
	if err := os.RemoveAll(srcGo + "template/pages"); err != nil {
		log.Default().Fatalln(err.Error())
	}
	if err := os.CopyFS(srcGo+"template/pages", os.DirFS(compileHtml)); err != nil {
		log.Default().Fatalln(err.Error())
	}

	files := make([]string, 0)

	err := filepath.WalkDir(srcGo+"template/pages/", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			name := strings.TrimPrefix(path, srcGo+"template/")
			if strings.Contains(name, "\\") {
				// Adding the ability to run on Windows with standard Go commands
				name = strings.TrimPrefix(strings.ReplaceAll(name, "\\", "/"), "server/web/pages/template/")
			}
			if !strings.HasPrefix(filepath.Base(name), ".") && !strings.HasSuffix(name, ".map") {
				files = append(files, name)
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	sort.Strings(files)
	fmap := writeEmbed(srcGo+"template/html.go", files)
	writeRoute(srcGo+"template/route.go", fmap)
}

func writeEmbed(fname string, files []string) map[string]string {
	ff, err := os.Create(fname)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer ff.Close()
	embedStr := `package template

import (
	_ "embed"
)
`
	ret := make(map[string]string)

	for _, f := range files {
		fname := cleanName(strings.TrimPrefix(f, "pages"))
		embedStr += "\n//go:embed " + f + "\nvar " + fname + " []byte\n"
		ret[strings.TrimPrefix(f, "pages")] = fname
	}

	ff.WriteString(embedStr)
	return ret
}

func writeRoute(fname string, fmap map[string]string) {
	ff, err := os.Create(fname)
	if err != nil {
		log.Fatal(err)
	}
	defer ff.Close()
	ff.WriteString("package template\n\nimport \"github.com/gin-gonic/gin\"\n\nfunc RouteWebPages(route gin.IRouter) {\n")
	fmt.Fprintln(ff, " route.GET(\"/\", assetHandler(Indexhtml, \"text/html; charset=utf-8\", \"no-cache\"))")
	fmt.Fprintln(ff, " route.HEAD(\"/\", assetHandler(Indexhtml, \"text/html; charset=utf-8\", \"no-cache\"))")
	mime.AddExtensionType(".webmanifest", "application/manifest+json")
	keys := make([]string, 0, len(fmap))
	for key := range fmap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, link := range keys {
		fmime := mime.TypeByExtension(filepath.Ext(link))
		if fmime == "" {
			fmime = "application/octet-stream"
		}
		if fmime == "application/javascript" || fmime == "application/xml" {
			fmime += "; charset=utf-8"
		}
		if fmime == "image/x-icon" {
			fmime = "image/vnd.microsoft.icon"
		}
		for _, method := range []string{"GET", "HEAD"} {
			fmt.Fprintf(ff, " route.%s(%q, assetHandler(%s, %q, %q))\n", method, link, fmap[link], fmime, assetCacheControl(link))
		}
	}
	fmt.Fprintln(ff, "}")
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func cleanName(fn string) string {
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return strings.Title(reg.ReplaceAllString(fn, ""))
}

func assetCacheControl(path string) string {
	name := filepath.Base(path)
	if strings.HasSuffix(name, ".html") || strings.HasSuffix(name, ".webmanifest") || name == "manifest.json" || name == "sw.js" || name == "service-worker.js" {
		return "no-cache"
	}
	if strings.HasPrefix(path, "/assets/") && regexp.MustCompile(`-[A-Za-z0-9_-]{8,}\.`).MatchString(name) {
		return "public, max-age=31536000, immutable"
	}
	return "public, max-age=3600"
}
