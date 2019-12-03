package notify

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"encoding/hex"
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
		durl       = vc.fileUpdateLists.PackgeUrl
		fsize      int64
		err        error
		res        *http.Response
		targetFile = UpdatePath + "/" + vc.version + "/"
		targetPath = UpdatePath + "/" + vc.version
	)

	clent := new(http.Client)
	clent.Timeout = Timeout

	uri, err := url.ParseRequestURI(durl)
	if err != nil {
		return fmt.Errorf("URL err")
	}
	filename := path.Base(uri.Path)
	vc.downloadFilename = filename

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
		fmt.Println("Installation package already exists ！")
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
	defer f.Close()

	_, err = io.Copy(f, res.Body)
	if err != nil {
		return err
	}

	err = CheckMD5(targetFile+filename, vc.fileUpdateLists.Packgemd5)
	if err != nil {
		os.Remove(targetFile + filename)
		return fmt.Errorf("Failed to checkMD5, downloaded file has been removed ,err : %v \n", err)
	}

	err = DeCompressByPath(targetFile+filename, targetFile)
	if err != nil {
		return err
	}

	fmt.Println("The latest version of GzV has been downloaded successfully")
	log.DefaultLogger.Info("The latest version of GzV has been downloaded successfully ")

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
	fmt.Printf("[newfilesize: %v], [oldfilesize : %v] \n", filesize, info.Size())
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
		return err
	}
	defer zipFile.Close()
	for _, innerFile := range zipFile.File {
		info := innerFile.FileInfo()
		if info.IsDir() {
			err = os.MkdirAll(innerFile.Name, os.ModePerm)
			if err != nil {
				return err
			}
			continue
		}
		srcFile, err := innerFile.Open()
		if err != nil {
			return fmt.Errorf("Unzip File Error : %v\n", err)
		}
		defer srcFile.Close()
		newFile, err := os.Create(dest + innerFile.Name)
		if err != nil {
			return fmt.Errorf("Unzip File Error : %v\n", err)
		}
		defer newFile.Close()
		err = os.Chmod(dest+innerFile.Name, 0777)
		if err != nil {
			return fmt.Errorf("Unzip File Error : %v\n", err)
		}

		io.Copy(newFile, srcFile)

	}
	return nil
}

func CheckMD5(path, targethash string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	md5hash := md5.New()
	if _, err := io.Copy(md5hash, f); err != nil {
		return err
	}

	hash := md5hash.Sum(nil)
	hashbin, err := hex.DecodeString(targethash)
	if err != nil {
		return err
	}

	if bytes.Equal(hash, hashbin) {
		fmt.Println("CheckMD5 successful !!!")
		return nil
	}

	return fmt.Errorf("Hash inconsistency")
}
