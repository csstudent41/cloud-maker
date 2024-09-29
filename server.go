package main

import (
	"archive/zip"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
)

func processAction(w http.ResponseWriter, r *http.Request, action string) *ServerError {
	switch action {
		default: return &ServerError{nil, "Invalid Action", 400}
		case "cut": return addSelectionToBuffer(w, r, cutBuffer)
		case "copy": return addSelectionToBuffer(w, r, copyBuffer)
		case "cancel-cut": return deleteBuffer(w, r, cutBuffer)
		case "cancel-copy": return deleteBuffer(w, r, copyBuffer)
		case "cut-paste": return moveFilesFromBuffer(w, r, cutBuffer)
		case "copy-paste": return pasteFilesFromBuffer(w, r, copyBuffer)
		case "newdir": return createNewDirectory(w, r)
	}
}


func viewHandler(w http.ResponseWriter, r *http.Request) *ServerError {
	var err error
	var serr *ServerError
	for k, v := range r.URL.Query() {
		switch k {
			default: http.Redirect(w, r, r.URL.Path, 302)
			case "action": serr = processAction(w, r, v[0])
		}
	}
	if serr != nil {
		return serr
	}
	fileNode, serr := getFileNode(r.URL.Path)
	if serr != nil {
		return serr
	}
	if fileNode.Info.Mode() & os.ModeSymlink != 0 {
		fileURI := fileNode.URI
		target := ""
		target, fileNode, err = fileNode.EvalSymlinks()
		if err != nil {
			if !os.IsNotExist(err) {
				return &ServerError{err, "", 500}
			}
			if len(target) != 0 {
				return &ServerError{err, fileURI+": broken link to '"+target+"'", 404}
			} else {
				return &ServerError{err, fileURI+": Inaccessible link", 404}
			}
		}
	}
	if !fileNode.IsDir {
		http.ServeFile(w, r, fileNode.Path)
		return nil
	}
	dirList, err := getDirList(fileNode.Path, "name", true, true)
	if err != nil {
		return &ServerError{err, "", 404}
	}
	cutBuf, err := readBuffer(cutBuffer)
	if err != nil {
		return &ServerError{err, "", 500}
	}
	copyBuf, err := readBuffer(copyBuffer)
	if err != nil {
		return &ServerError{err, "", 500}
	}
	fileNode.Data = dirList
	err = renderTemplate(w, "viewDirList", &FSData{
		CutCount: len(cutBuf),
		CutBuffer: cutBuf,
		CopyCount: len(copyBuf),
		CopyBuffer: copyBuf,
		FileCount: len(dirList),
		File: fileNode,
	})
	if err != nil {
		return &ServerError{err, "", 500}
	}
	return nil
}


func downloadHandler(w http.ResponseWriter, r *http.Request) *ServerError {
	fmt.Printf("%s\n", r.Form)
	return nil
	fileNode, files, serr := getSelectedNodes(r)
	if serr != nil {
		return serr
	}
	if len(files) == 1 && !files[0].IsDir {
		sendFile(w, r, files[0].Path)
		return nil
	}
	zipName := fileNode.Info.Name() + ".zip"
	target := "/tmp/cloud/" + zipName

	archive, err := os.Create(target)
	if err != nil {
		return &ServerError{err, "", 500}
	}
	defer archive.Close()

	zipWriter := zip.NewWriter(archive)
	defer zipWriter.Close()

	for _, file := range files {
		err := addToZip(file.Path, zipWriter)
		if err != nil {
			return &ServerError{err, "", 500}
		}
	}
	zipWriter.Close()
	sendFile(w, r, target, zipName)
	return nil
}


func blockDelete(w http.ResponseWriter, r *http.Request) *ServerError {
	msg := ""
	_, files, serr := getSelectedNodes(r)
	if serr != nil {
		return serr
	}
	msg += "Delete is currently disabled for testing and security reasons.\n"
	msg += "You requested to delete following files :-\n\n"
	for _, file := range files {
		msg += file.Path + "\n"
	}
	fmt.Fprintf(w, msg)
	return nil
}

func deleteHandler(w http.ResponseWriter, r *http.Request) *ServerError {
	return blockDelete(w, r)
	fileNode, files, serr := getSelectedNodes(r)
	if serr != nil {
		return serr
	}
	for _, file := range files {
		err := os.RemoveAll(file.Path)
		if err != nil {
			return &ServerError{err, "", 500}
		}
	}
	isExist, err := fileExists(fileNode.Path)
	if err != nil {
		return &ServerError{err, "", 500}
	}
	if !isExist {
		http.Redirect(w, r, "/view/" + filepath.Dir(fileNode.URI), 303)
	} else {
		http.Redirect(w, r, "/view/" + fileNode.URI, 303)
	}
	return nil
}


func uploadHandler(w http.ResponseWriter, r *http.Request) *ServerError {
	return &ServerError{nil, "Not implemented.", 404}
}


func fileHandler(w http.ResponseWriter, r *http.Request) *ServerError {
	fileNode, serr := getFileNode(r.URL.Path)
	if serr != nil {
		return serr
	}
	if fileNode.IsDir {
		return &ServerError{nil, "File not Found.", 404}
	}
	http.ServeFile(w, r, fileNode.Path)
	return nil
}


func handler(w http.ResponseWriter, r *http.Request) *ServerError {
	if r.URL.Path != "/" {
		return &ServerError{nil, "Invalid URL", 404}
	}
	http.Redirect(w, r, "view", 303)
	return nil
}


type httpHandler func(http.ResponseWriter, *http.Request) *ServerError

func (fn httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if serr := fn(w, r); serr != nil {
		if serr.Err != nil {
			fmt.Println("\n\nError Type:", reflect.TypeOf(serr.Err))
			fmt.Println("Error Message:", serr.Error())
		}
		if serr.Message == "" {
			serr.Message = "Internal Server Error"
		}
		http.Error(w, serr.Message, serr.Status)
	}
}


func main() {
	fileServer := http.FileServer(http.Dir("./static"))
	http.Handle("/", httpHandler(handler))
	http.Handle("/view/", httpHandler(viewHandler))
	http.Handle("/upload/", httpHandler(uploadHandler))
	http.Handle("/download/", httpHandler(downloadHandler))
	http.Handle("/delete/", httpHandler(deleteHandler))
	http.Handle("/file/", httpHandler(fileHandler))
	http.Handle("/static/", http.StripPrefix("/static/", fileServer))
	fmt.Println("Server Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
