package main

import (
	"archive/zip"
	"bufio"
	"io"
	"os"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	// "syscall"
)

type MalformedLinkError struct {
	Link   string
	Target string
}

func (e *MalformedLinkError) Error() string { return fmt.Sprintf("%s: broken link to %s", e.Link, e.Target) }

type FileNode struct {
	URI     string
	Path    string
	IsDir   bool
	Info    os.FileInfo
	Data    any
}

func (fileNode *FileNode) HTMLPath() template.HTML {
	var htmlpath string
	htmlpath += `<a href="/view/`  + `">` + "Home" + `</a> `
	p := strings.Split(fileNode.URI, string(os.PathSeparator))
	for i, dir := range p {
		if p[i] != "" {
			htmlpath += `> <a href="/view/` + filepath.Join(p[:i+1]...) + `">` + dir + `</a> `
		}
	}
	return template.HTML(htmlpath)
}

func (fileNode *FileNode) EvalSymlinks() (string, *FileNode, error) {
	var err error
	target, path, err := linkDeref(fileNode.Path);
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, err
		}
		return target, nil, err
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		return target, nil, err
	}
	return target, &FileNode{
		Path: path,
		URI: strings.TrimPrefix(path, homeDir),
		Info: fileInfo,
		IsDir: fileInfo.IsDir(),
	}, nil
}

func (fileNode *FileNode) IconPath() (string, error) {
	var icon string;
	switch fileNode.Info.Mode() & os.ModeType {
	default: icon = "file-earmark.svg"
	case os.ModeIrregular: icon = "question.svg"
	case os.ModeDir: icon = "folder2.svg"
	case os.ModeSymlink:
		_, fileNode, err := fileNode.EvalSymlinks()
		if err != nil {
			if !os.IsNotExist(err) {
				return "", err
			}
			icon = "link-broken-45deg.svg"
		} else {
			if fileNode.IsDir {
				icon = "folder-symlink.svg"
			} else {
				icon = "link-45deg.svg"
			}
		}
	}
	return filepath.Join("/static/icons/bs/files", icon), nil
}

func (fileNode *FileNode) Size() (string, error) {
	var err error
 	if fileNode.Mode() == "l" {
		_, fileNode, err = fileNode.EvalSymlinks()
		if err != nil {
			return "", nil
		}
	}
	if fileNode.IsDir {
		return "", nil
	}
	size := float64(fileNode.Info.Size())
	if size < 100 {
		return strconv.FormatFloat(size, 'f', 0, 64) + " B", nil
	}
	units := []string{" KB", " MB", " GB", " TB", " PB", " EB", " ZB"}
	for i := 0; i < 7; i++ {
		size /= 1024
		if size < 100 {
			return strconv.FormatFloat(size, 'f', 1, 64) + units[i], nil
		}
	}
	return strconv.FormatFloat(size, 'f', 1, 64) + " YiB", nil
}

func (fileNode *FileNode) Mode() string {
	switch fileNode.Info.Mode() & os.ModeType {
	default: return "f"
	case os.ModeDir: return "d"
	case os.ModeSymlink: return "l"
	}
}

func (fileNode *FileNode) ModDate() string {
	t := fileNode.Info.ModTime()
	return fmt.Sprintf("%.3s %d, %d\n", t.Month(), t.Day(), t.Year())
}

func (fileNode *FileNode) ModTime() string {
	t := fileNode.Info.ModTime()
	return fmt.Sprintf("%d:%d\n", t.Hour(), t.Minute())
}


func (fileNode *FileNode) Details() (string, error) {
	text := "Non-Regular File"
	switch fileNode.Info.Mode() & os.ModeType {

	default:
		if fileNode.Info.Size() == 0 {
			return "Empty File", nil
		}
		file, err := os.Open(fileNode.Path)
		if err != nil {
			return "", err
		}
		buffer := make([]byte, 512)
		_, err = file.Read(buffer)
		if err != nil {
			return "", err
		}
		contentType := http.DetectContentType(buffer)
		if contentType == "application/octet-stream" {
			text = "Text File"
		} else {
			text = "*"+contentType
		}

	case os.ModeDir:
		text = "Folder"

	case os.ModeSymlink:
		target, _, err := fileNode.EvalSymlinks()
		if err != nil {
			if !os.IsNotExist(err) {
				return "", err
			}
			if len(target) > 0 {
				text = "Broken Link to '"+target+"'"
			} else {
				text = "Inaccessible Link"
			}
		} else {
			text = "Link to " + target
		}

	case os.ModeSocket:
		text = "Unix Socket"

	case os.ModeDevice:
		text = "Device File"

	case os.ModeNamedPipe:
		text = "Named Pipe"

	case os.ModeTemporary:
		text = "Temporary File"

	case os.ModeAppend:
	case os.ModeExclusive:
	case os.ModeSetuid:
	case os.ModeSetgid:
	case os.ModeCharDevice:
	case os.ModeSticky:
	case os.ModeIrregular:
	}

	return text, nil
}


func getDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}


func getDirList(path string, sortBy string, ascending bool, dirsFirst bool) ([]*FileNode, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	files := make([]*FileNode, len(entries))
	for i, entry := range entries {
		filePath := filepath.Join(path, entry.Name())
		fileURI := strings.TrimLeft(path, homeDir)
		fileInfo, err := entry.Info()
		if err != nil {
			return nil, err
		}
		files[i] = &FileNode{
			Path: filePath,
			URI: fileURI,
			IsDir: entry.IsDir(),
			Info: fileInfo,
		}
	}

	switch sortBy {
	case "name": sort.SliceStable(files, func(i, j int) bool {
			return strings.ToLower(files[i].Info.Name()) < strings.ToLower(files[j].Info.Name())
		})
	case "size": sort.SliceStable(files, func(i, j int) bool {
			return files[i].Info.Size() < files[j].Info.Size()
		})
	case "time": sort.SliceStable(files, func(i, j int) bool {
			return files[i].Info.ModTime().Before(files[j].Info.ModTime())
		})
	}

	if !ascending {
		for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
			files[i], files[j] = files[j], files[i]
		}
	}

	if dirsFirst {
		var dirs, notDirs []*FileNode
		for _, fileNode := range files {
			info, err := os.Stat(fileNode.Path)
			if err != nil {
				if os.IsNotExist(err) {
					info, err = os.Lstat(fileNode.Path)
					if err != nil {
						return nil, err
					}
				} else {
					return nil, err
				}
			}
			if info.IsDir() {
				dirs = append(dirs, fileNode)
			} else {
				notDirs = append(notDirs, fileNode)
			}
		}
		return append(dirs, notDirs...), nil
	}

	return files, nil
}


func addToZip(source string, writer *zip.Writer) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Method = zip.Deflate
		header.Name, err = filepath.Rel(filepath.Dir(source), path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			header.Name += "/"
		}
		headerWriter, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(headerWriter, f)
		return err
	})
}


func readBuffer(path string) ([]string, error) {
	buff, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	defer buff.Close()

	var buffer []string
	scanner := bufio.NewScanner(buff)
	for scanner.Scan() {
		buffer = append(buffer, scanner.Text())
	}
	return buffer, nil
}


func fileExists(path string) (bool, error) {
	_, err := os.Lstat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}


func copyFile(src, dst string) error {
	fin, err := os.Open(src)
	if err != nil {
		return err
	}
	defer fin.Close()

	fout, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer fout.Close()

	_, err = io.Copy(fout, fin)
	if err != nil {
		return err
	}
	fin.Close()

	return nil
}


func copyTo(src, dstDir string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	dst := filepath.Join(dstDir, info.Name())

	fmt.Printf("Copying %s to %s\n", src, dstDir)
	switch info.Mode() & os.ModeType {
	case os.ModeDir:
		if err := os.MkdirAll(dst, 0755); err != nil {
			return err
		}
		if err := copyDir(src, dst); err != nil {
			return err
		}
	case os.ModeSymlink:
		if err := copySymlink(src, dst); err != nil {
			return err
		}
	default:
		if err := copyFile(src, dst); err != nil {
			return err
		}
	}
	fmt.Println("Finished Copying.\n\n")

	/*
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("failed to get raw syscall.Stat_t data for '%s'", src)
		}
		if err := os.Lchown(dst, int(stat.Uid), int(stat.Gid)); err != nil {
			return err
		}
	*/

	if info.Mode()&os.ModeSymlink == 0 {
		return os.Chmod(dst, info.Mode())
	}
	return nil
}


func linkDeref(link string) (string, string, error) {
	target, err := os.Readlink(link)
	if err != nil {
		return "", "", err
	}
	path := target
	if filepath.IsAbs(target) {
		if !strings.HasPrefix(path, homeDir) {
			return target, "", os.ErrNotExist
		}
		target = strings.TrimPrefix(target, homeDir)
	} else {
		path = filepath.Join(filepath.Dir(link), path)
		if !strings.HasPrefix(path, homeDir) {
			return target, "", os.ErrNotExist
		}
	}
	return target, path, nil
}


var configDir string

func readData(name string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(configDir, name))
	if err != nil {
		return nil, err
	}
	return data[:len(data)-1], nil
}


func writeData(name string, data string) error {
	return os.WriteFile(filepath.Join(configDir, name), []byte(data), 644)
}

func init() {
	userHome, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	configDir = filepath.Join(userHome, ".config", "cloud-maker")
	if err = os.MkdirAll(configDir, 0755); err != nil {
		panic(err)
	}
}
