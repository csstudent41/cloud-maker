package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type ServerError struct {
	Err error
	Message string
	Status int
}

func (e *ServerError) Error() string { return e.Err.Error() }
func (e *ServerError) Unwrap() error { return e.Err }


var (
	homeDir = "/home/master/Dev/test.test/"
	dataDir = "/tmp/cloud/"
	cutBuffer = filepath.Join(dataDir, "cut_buffer")
	copyBuffer = filepath.Join(dataDir, "copy_buffer")
)


func getFileNode(url string) (*FileNode, *ServerError) {
	p := strings.Split(url, "/")
	fileURI := strings.Trim(strings.Join(p[2:], "/"), "/")
	filePath := filepath.Join(homeDir, fileURI)

	fileInfo, err := os.Lstat(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, &ServerError{err, fileURI+" not found", 404}
		}
		return nil, &ServerError{err, "", 500}
	}

	return &FileNode{
		Path: filePath,
		URI: fileURI,
		IsDir: fileInfo.IsDir(),
		Info: fileInfo,
	}, nil
}


var filePattern = regexp.MustCompile(`^-file-entry--(.+)$`)

func getSelectedNodes(r *http.Request) (*FileNode, []*FileNode, *ServerError) {
	fileNode, e := getFileNode(r.URL.Path)
	if e != nil {
		return nil, nil, e
	}
	r.ParseForm()
	fmt.Println(r.Form)
	var fileNames []string
	for key := range r.Form {
		if match := filePattern.FindStringSubmatch(key); len(match) > 1 {
			fileNames = append(fileNames, match[1])
		}
	}
	fmt.Printf("FileNames: %s\n", fileNames)
	if len(fileNames) == 0 {
		return fileNode, []*FileNode{fileNode}, nil
	}

	files := make([]*FileNode, len(fileNames))
	for i, fileName := range fileNames {
		fileNode, e := getFileNode(filepath.Join(r.URL.Path, fileName))
		if e != nil {
			return fileNode, nil, e
		}
		files[i] = fileNode
	}
	return fileNode, files, nil
}


func sendFile(w http.ResponseWriter, r *http.Request, info ...string) {
	fmt.Printf("info: %s\n", info[:])
	if len(info) < 2 {
		info = append(info, filepath.Base(info[0]))
	}
	w.Header().Set("Content-Disposition", "attachment; filename=" + info[1])
	http.ServeFile(w, r, info[0])
}


func addSelectionToBuffer(w http.ResponseWriter, r *http.Request, bufferPath string) *ServerError {
	_, files, e := getSelectedNodes(r)
	if e != nil {
		return e
	}
	buffer, err := readBuffer(bufferPath)
	if err != nil {
		return &ServerError{err, "", 500}
	}
	buff, err := os.OpenFile(bufferPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return &ServerError{err, "", 500}
	}
	var fileURI string

	writeToBuffer:
	for _, file := range files {
		fileURI = strings.Trim(file.URI, "/")
		for _, line := range buffer {
			if strings.Trim(line, "/") == fileURI {
					continue writeToBuffer
			}
		}
		if file.IsDir {
			fileURI += "/"
		}
		buff.WriteString("/" + fileURI + "\r\n")
	}

	http.Redirect(w, r, r.URL.Path, 303)
	return nil
}


func deleteBuffer(w http.ResponseWriter, r *http.Request, bufferPath string) *ServerError {
	err := os.Remove(bufferPath)
	if err != nil {
		return &ServerError{err, "", 500}
	}
	http.Redirect(w, r, r.URL.Path, 303)
	return nil
}


func moveFilesFromBuffer(w http.ResponseWriter, r *http.Request, bufferPath string) *ServerError {
	fileNode, serr := getFileNode(r.URL.Path)
	if serr != nil {
		return serr
	}
	if !fileNode.IsDir {
		return &ServerError{nil, "Cannot move file, destination is not a directory", 400}
	}
	buffer, err := readBuffer(bufferPath)
	if err != nil {
		return &ServerError{err, "", 500}
	}
	for _, line := range buffer {
		err := copyTo(filepath.Join(homeDir, line), fileNode.Path)
		if err != nil {
			return &ServerError{err, "", 500}
		}
	}
	deleteBuffer(w, r, bufferPath)
	return nil
}


func pasteFilesFromBuffer(w http.ResponseWriter, r *http.Request, bufferPath string) *ServerError {
	fileNode, serr := getFileNode(r.URL.Path)
	if serr != nil {
		return serr
	}
	if !fileNode.IsDir {
		return &ServerError{nil, "Cannot copy files, destination is not a directory", 400}
	}
	buffer, err := readBuffer(bufferPath)
	if err != nil {
		return &ServerError{err, "", 500}
	}
	fmt.Printf("Buffer content: %s\n", buffer)
	for _, line := range buffer {
		err := copyTo(filepath.Join(homeDir, line), fileNode.Path)
		if err != nil {
			return &ServerError{err, "", 500}
		}
	}
	deleteBuffer(w, r, bufferPath)
	return nil
}
