package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/zserge/lorca"
)

var sourceFolder = "./html-zip-downloaded"
var doneZipFolder = "./done-zip"
var destinyFolder = "./processed-markdown"
var tempUnzipFolder = "./tempUnzip"
var checkTitle = true
var checkBold = true
var checkIndex = true
var checkTwoLines = true
var checkCollapsible = true
var checkMoveFilesAfterConvert = true

func main() {
	ui, err := lorca.New("", "", 1024, 768)
	if err != nil {
		log.Fatal(err)
	}

	////// define javascript funcs

	ui.Bind("Refresh", func() {
		Refresh(ui)

	})

	/// BUTTON CONVERT:
	ui.Bind("convert", func() {
		CheckVariablesToFormating(ui)

		CheckPathExists(true, destinyFolder, tempUnzipFolder)

		UnzipAll(sourceFolder, tempUnzipFolder)

		ConvertHTMLPageToMD(tempUnzipFolder)

		Refresh(ui)
	})

	////// end of javascript funcs

	defer ui.Close()

	////// define html

	ui.Load("data:text/html," + url.PathEscape(`
	<html> 	
		<head><title>Confluence to Markdown - Taglatam (alpha v0.2.5)</title></head>
		<body onload="Refresh()"><h5>Zips to convert</h5>
		<br>
		<h4>Put the zip files in ./html-zip-downloaded  List of done zips in ./done-zip</h4>
		<textarea style="resize: none;height: 200px;width: 45%;" readonly id="listOfFiles"></textarea>
		<textarea style="resize: none;height: 200px;width: 45%;" readonly id="listOfFilesDone"></textarea>
		<button onclick="Refresh()">refresh </button>
		<br>
		<br>
		<br>
		<br>
		<div>The MD files have will create in ./processed-markdown</div>
		<br>
		<input type="checkbox" id="checkMoveFilesAfterConvert" checked>Move files to done folder after convert
		<br>
		<input type="checkbox" id="checkTitle" checked>Change title 1 to title 2 (# -> ##)
		<br>
		<input type="checkbox" id="checkBold" checked>Eliminate bold from titles
		<br>
		<input type="checkbox" id="checkIndex" checked>Remove index
		<br>
		<input type="checkbox" id="checkTwoLines" checked>Remove 2 first lines
		<br>
		<input type="checkbox" id="checkCollapsible" checked>Add collapsible in code sections
		<br>
		<button onclick="convert()">Convert</button>
		<br>
		<textarea style="resize: none;height: 200px;width: 90%;" readonly id="listOfFilesProcess"></textarea>
		<br>
		<br>
		<br>
		<div style=" position: fixed;bottom: 4px;right: 4px;">Daniel Boullon - Taglatam </div>
		</body>
	</html>
	`))
	<-ui.Done()
}

func Refresh(ui lorca.UI) {
	CheckVariablesToFormating(ui)
	CheckPathExists(true, sourceFolder)
	ReadFolderAndSetTextArea(sourceFolder, "listOfFiles", ui)
	ReadFolderAndSetTextArea(doneZipFolder, "listOfFilesDone", ui)
	ReadFolderAndSetTextArea(destinyFolder, "listOfFilesProcess", ui)
}

func CheckVariablesToFormating(ui lorca.UI) {
	checkTitle = ui.Eval(`document.getElementById("checkTitle").checked`).Bool()
	checkBold = ui.Eval(`document.getElementById("checkBold").checked`).Bool()
	checkIndex = ui.Eval(`document.getElementById("checkIndex").checked`).Bool()
	checkTwoLines = ui.Eval(`document.getElementById("checkTwoLines").checked`).Bool()
	checkCollapsible = ui.Eval(`document.getElementById("checkCollapsible").checked`).Bool()
	checkMoveFilesAfterConvert = ui.Eval(`document.getElementById("checkMoveFilesAfterConvert").checked`).Bool()
}

func ConvertHTMLPageToMD(tempUnzipFolder string) {
	files, err := ioutil.ReadDir(tempUnzipFolder)
	if err != nil {
		log.Fatal(err)
	}
	for _, folder := range files {
		if folder.IsDir() {

			unzipFolder := tempUnzipFolder + "/" + folder.Name()
			pageHTMLfile, err := SearchPageHTMLFile(unzipFolder)
			pageHTMLpath := tempUnzipFolder + "/" + folder.Name() + "/" + pageHTMLfile
			dirMDpath := folder.Name()
			dirMDpath = strings.ReplaceAll(dirMDpath, " ", "-")
			dirMDpath = strings.ToLower(dirMDpath)

			reg := regexp.MustCompile(`-v\d.*`)
			dirMDpath = reg.ReplaceAllString(dirMDpath, "${1}")
			fmt.Print("dirMDPath: " + dirMDpath)

			fullMDPath := destinyFolder + "/" + dirMDpath

			fileMDpath := fullMDPath + "/" + dirMDpath + ".md"

			if err != nil {
				log.Fatal(err)
			}

			CheckPathExists(true, fullMDPath)

			ConvertHtmlToMD(pageHTMLpath, fileMDpath)

			CopyImages(unzipFolder, fullMDPath)
		}
	}
}

func CopyImages(folder, dirMDpath string) {
	regex := `.*\.(gif|jpe?g|tiff?|png|webp|bmp)`

	files, err := ioutil.ReadDir(folder)

	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {

		if !file.IsDir() {

			matched, err := regexp.MatchString(regex, file.Name())

			if err != nil {
				log.Fatal(err)
			}

			if matched {

				sourceImage := folder + "/" + file.Name()
				destinyImage := dirMDpath + "/" + file.Name()

				log.Printf("Coping %s to %s", sourceImage, destinyImage)

				CopyFile(sourceImage, destinyImage)
			}

		}
	}
}

func CopyFile(sourceFile, destinationFile string) {
	input, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = ioutil.WriteFile(destinationFile, input, 0644)
	if err != nil {
		fmt.Println("Error coping file ", destinationFile)
		fmt.Println(err)
		return
	}
}

func ConvertHtmlToMD(pageHTMLpath, fileMDpath string) {
	log.Printf("File to convert is: %s in %s", pageHTMLpath, fileMDpath)
	converter := md.NewConverter("", true, nil)

	htmlFile, err := os.ReadFile(pageHTMLpath)
	if err != nil {
		log.Fatal(err)
	}

	html := string(htmlFile)

	markdown, err := converter.ConvertString(html)
	if err != nil {
		log.Fatal(err)
	}
	/// FORMAT markdown

	if checkTitle {
		r := regexp.MustCompile(`(?m)^# `)
		markdown = r.ReplaceAllStringFunc(markdown,
			func(m string) string {
				return strings.ReplaceAll(m, "# ", "## ")
			})
		markdown = strings.Replace(markdown, "## ", "# ", 1)
	}

	if checkIndex {
		re := regexp.MustCompile(`( *- \[.*\]\(.*\)\s*(\r\n|\r|\n)+)+`)
		firstMatch := re.FindString(markdown)

		markdown = strings.Replace(markdown, firstMatch, "", 1)
	}

	if checkBold {
		reg := regexp.MustCompile(`(?m)#.*\*\*.*\*\*`)
		markdown = reg.ReplaceAllStringFunc(markdown,
			func(m string) string {
				return strings.ReplaceAll(m, "**", "")
			})
	}

	// remove first
	if checkTwoLines {
		rege := regexp.MustCompile(`.*(\r\n|\r|\n).*(\r\n|\r|\n)`)
		firstTwoLine := rege.FindString(markdown)
		markdown = strings.Replace(markdown, firstTwoLine, "", 1)
	}

	// add collapsible

	// find ```(.*)```
	if checkCollapsible {
		rege := regexp.MustCompile("(?m)^(```(?:.*)(\\r\\n|\\r|\\n)([\\s\\S]*?)```)$")
		markdown = rege.ReplaceAllStringFunc(markdown,
			func(m string) string {
				return "<details>\n<summary>Code</summary>\n\n" + m + "\n\n</details>"
			})
	}

	// write file :)
	markdownBinary := []byte(markdown)

	errr := ioutil.WriteFile(fileMDpath, markdownBinary, 0777)
	if errr != nil {
		log.Fatal(err)
	}
}

func SearchPageHTMLFile(folder string) (string, error) {
	regex := `page\d+\.html`
	files, err := ioutil.ReadDir(folder)
	if err != nil {
		log.Fatal(err)
	}
	for _, file := range files {
		if !file.IsDir() {
			matched, err := regexp.MatchString(regex, file.Name())
			if err != nil {
				log.Fatal(err)
			}
			if matched {
				return file.Name(), nil
			}
		}
	}
	return "", errors.New("Page{id}.html dont found.")
}

// check if the params paths exists
func CheckPathExists(createFolder bool, paths ...string) {
	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Printf("%s path not exists", path)
			if createFolder {
				err = os.Mkdir(path, 0755)
				if err != nil {
					log.Fatal(err)
				} else {
					log.Printf("%s  folder was created", path)
				}
			}
		} else {
			log.Printf("%s path exists", path)
		}
	}
}

// read files in folder and put in textarea.
func ReadFolderAndSetTextArea(path string, idTextArea string, ui lorca.UI) {
	list := fmt.Sprintf(`document.getElementById('%s').value = ""`, idTextArea)
	ui.Eval(list)

	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	clean := fmt.Sprintf(`document.getElementById('%s').value = ""`, idTextArea)
	ui.Eval(clean)

	for _, file := range files {
		list := fmt.Sprintf(`document.getElementById('%s').value += "%s\n"`, idTextArea, file.Name())
		ui.Eval(list)
	}
}

// read files in folders.
func UnzipAll(zipFolder string, unzipFolder string) {
	files, err := ioutil.ReadDir(zipFolder)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s files ready to unzip", files)
	for _, file := range files {
		// variables need to unzip
		pathFile := zipFolder + "/" + file.Name()
		filename := file.Name()
		extension := filepath.Ext(filename)
		name := filename[0 : len(filename)-len(extension)]
		pathUnzip := unzipFolder + "/" + name

		err = Unzip(pathFile, pathUnzip)
		if err != nil {
			log.Fatal(err)
		}

		// to do: in windows dont work move file (for now (?))
		pathDoneZipFolder := doneZipFolder + "/" + filename

		fmt.Printf("Move FILE: %v", checkMoveFilesAfterConvert)

		if checkMoveFilesAfterConvert {
			MoveFile(pathFile, pathDoneZipFolder)
		}
	}
}

func MoveFile(pathFile, pathDoneZipFolder string) {
	CheckPathExists(true, doneZipFolder)
	CheckPathExists(false, pathFile)
	log.Printf("Moving %s to %s", pathFile, pathDoneZipFolder)

	e := os.Rename(pathFile, pathDoneZipFolder)
	if e != nil {
		log.Fatal(e)
	}
}

// unzip html files
func Unzip(src, dest string) error {

	log.Printf("unzip %s", src)
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}

	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	os.MkdirAll(dest, 0755)

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)

		// Check for ZipSlip (Directory traversal)
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}
	return nil
}
