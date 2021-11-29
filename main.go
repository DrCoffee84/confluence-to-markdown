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

func main() {
	ui, err := lorca.New("", "", 1024, 768)
	if err != nil {
		log.Fatal(err)
	}

	////// define javascript funcs

	ui.Bind("refresh", func() {
		CheckPathExists(true, sourceFolder)
		ReadFolderAndSetTextArea(sourceFolder, "listOfFiles", ui)
	})

	ui.Bind("convert", func() {

		CheckPathExists(true, destinyFolder, tempUnzipFolder)

		UnzipAll(sourceFolder, tempUnzipFolder)

		convertHTMLPageToMD(tempUnzipFolder)

		ReadFolderAndSetTextArea(destinyFolder, "listOfFilesDone", ui)
	})

	////// end of javascript funcs

	defer ui.Close()

	////// define html

	ui.Load("data:text/html," + url.PathEscape(`
	<html>
		<head><title>Confluence to Markdown - Taglatam (alpha v0.0.1)</title></head>
		<body onload="refresh()"><h5>Zips to convert</h5>
		<div>Put the zip files in ./html-zip-downloaded</div>
		<button onclick="refresh()">refresh</button>
		<br>
		<textarea style="resize: none;height: 200px;width: 90%;" readonly id="listOfFiles"></textarea>
		<br>
		<br>
		<br>
		<br>
		<div>The MD files have will create in ./processed-markdown</div>
		<button onclick="convert()">Convert</button>
		<br>
		<textarea style="resize: none;height: 200px;width: 90%;" readonly id="listOfFilesDone"></textarea>
		<br>
		<br>
		<br>
		<div style=" position: fixed;bottom: 4px;right: 4px;">Daniel Boullon - Taglatam </div>
		</body>
	</html>
	`))

	<-ui.Done()

}

//TODO: using listFiles function with function params :)
//TODO: same with listFolders

func convertHTMLPageToMD(tempUnzipFolder string) {
	files, err := ioutil.ReadDir(tempUnzipFolder)
	if err != nil {
		log.Fatal(err)
	}
	for _, folder := range files {
		if folder.IsDir() {
			unzipFolder := tempUnzipFolder + "/" + folder.Name()
			pageHTMLfile, err := SearchPageHTMLFile(unzipFolder)
			pageHTMLpath := tempUnzipFolder + "/" + folder.Name() + "/" + pageHTMLfile
			dirMDpath := destinyFolder + "/" + folder.Name()
			fileMDpath := dirMDpath + "/" + folder.Name() + ".md"

			if err != nil {
				log.Fatal(err)
			}

			CheckPathExists(true, dirMDpath)

			ConvertHtmlToMD(pageHTMLpath, fileMDpath)

			CopyImages(unzipFolder, dirMDpath)
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

	// write file :)
	mydata := []byte(markdown)

	errr := ioutil.WriteFile(fileMDpath, mydata, 0777)
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
		//	pathDoneZipFolder := doneZipFolder + "/" + filename
		//	MoveFile(pathFile, pathDoneZipFolder)
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
