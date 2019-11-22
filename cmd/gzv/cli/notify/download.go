package notify

import (
	"archive/zip"
	"fmt"
	"github.com/zvchain/zvchain/log"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
)

func (vc *VersionChecker) download() error {
	var (
		durl       string
		fsize      int64
		err        error
		res        *http.Response
		targetFile = UpdatePath + "/" + vc.version + "/"
		targetPath = UpdatePath + "/" + vc.version
	)

	clent := new(http.Client)
	clent.Timeout = Timeout

	switch System {
	case "darwin":
		durl = UrlDarwin
	case "linux":
		durl = UrlLinux
	case "windows":
		durl = UrlWindows
	}

	uri, err := url.ParseRequestURI(durl)
	if err != nil {
		panic("URL err")
	}
	filename := path.Base(uri.Path)

	res, err = clent.Get(durl)
	if err != nil {
		return err
	}

	fsize, err = strconv.ParseInt(res.Header.Get("Content-Length"), 10, 32)
	if err != nil {
		return err
	}

	vc.filesize = fsize
	if isFileExist(targetFile+filename, fsize) {
		fmt.Println("Installation package already exists ！\n")
		return nil
	}

	err = createFolder(targetPath)
	if err != nil {
		return err
	}

	f, err := os.Create(targetFile + filename)
	if err != nil {
		return err
	}

	_, err = io.Copy(f, res.Body)
	if err != nil {
		return err
	}

	err = DeCompressByPath(targetFile+filename, targetFile)
	if err != nil {
		return err
	}

	fmt.Println("The latest version of GzV has been downloaded successfully\n")
	log.DefaultLogger.Info("The latest version of GzV has been downloaded successfully\n ")

	defer f.Close()
	return nil
}

func createFolder(filePath string) error {
	if !isFolderExist(filePath) {
		err := os.MkdirAll(filePath, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

func isFolderExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func isFileExist(filepath string, filesize int64) bool {
	info, err := os.Stat(filepath)
	if os.IsNotExist(err) {
		fmt.Println(info)
		return false
	}
	fmt.Printf("=====>>>filesize: %v, info.Size : %v \n", filesize, info.Size())
	if filesize == info.Size() {
		fmt.Printf("Installation package already exists : %v , %v , %v \n", info.Name(), info.Size(), info.ModTime())
		return true
	}
	err = os.RemoveAll(filepath)
	if err != nil {
		fmt.Println(err)
	}
	return false
}

func DeCompressByPath(tarFile, dest string) error {
	srcFile, err := os.Open(tarFile)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	return DeCompress(srcFile, dest)
}
func DeCompress(srcFile *os.File, dest string) error {
	zipFile, err := zip.OpenReader(srcFile.Name())
	if err != nil {
		fmt.Println("Unzip File Error：", err)
		return err
	}
	defer zipFile.Close()
	for _, innerFile := range zipFile.File {
		info := innerFile.FileInfo()
		if info.IsDir() {
			err = os.MkdirAll(innerFile.Name, os.ModePerm)
			if err != nil {
				fmt.Println("Unzip File Error : ", err)
				return err
			}
			continue
		}
		srcFile, err := innerFile.Open()
		if err != nil {
			fmt.Println("Unzip File Error : ", err)
			continue
		}
		defer srcFile.Close()
		fmt.Println("================================>>", dest+innerFile.Name)
		newFile, err := os.Create(dest + innerFile.Name)
		if err != nil {
			fmt.Println("Unzip File Error : ", err)
			continue
		}
		err = os.Chmod(dest+innerFile.Name, 0777)
		if err != nil {
			fmt.Println("Chmod File Error : ", err)
			continue
		}

		io.Copy(newFile, srcFile)
		newFile.Close()
	}
	return nil
}
