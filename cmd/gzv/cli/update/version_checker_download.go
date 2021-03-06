//   Copyright (C) 2019 ZVChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package update

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/log"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

func (vc *VersionChecker) download() error {
	var (
		durl       = vc.fileUpdateLists.PackageUrl
		err        error
		res        *http.Response
		targetFile = updatePath + "/" + vc.version + "/"
		targetPath = updatePath + "/" + vc.version
	)

	clent := new(http.Client)
	clent.Timeout = timeout

	uri, err := url.ParseRequestURI(durl)
	if err != nil {
		return err
	}
	filename := path.Base(uri.Path)
	vc.downloadFilename = filename

	res, err = clent.Get(durl)
	if res.Status != "200 OK" {
		return fmt.Errorf("URL response err")
	}
	if err != nil {
		return err
	}

	if isFileExist(targetFile + filename) {
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

	err = CheckMD5(targetFile+filename, vc.fileUpdateLists.PackageMd5)
	if err != nil {
		os.Remove(targetFile + filename)
		return fmt.Errorf("Failed to checkMD5, downloaded file has been removed ,err : %v \n", err)
	}

	fmt.Println("The latest version of GzV has been downloaded successfully")
	log.DefaultLogger.Infoln("The latest version of GzV has been downloaded successfully ")

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

func isFileExist(filepath string) bool {
	info, err := os.Stat(filepath)
	if os.IsNotExist(err) {
		return false
	}
	log.DefaultLogger.Infof("Installation package already exists : %v , %v , %v \n", info.Name(), info.Size(), info.ModTime())
	return true
}

func CheckMD5(path, targetHash string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	md5Hash := md5.New()
	if _, err := io.Copy(md5Hash, f); err != nil {
		return err
	}
	hash := md5Hash.Sum(nil)

	if strings.HasSuffix(targetHash, "0x") || strings.HasSuffix(targetHash, "0X") {
		targetHash = targetHash[2:]
	}

	hashBin, err := hex.DecodeString(targetHash)
	if err != nil {
		return err
	}

	if bytes.Equal(hash, hashBin) {
		return nil
	}

	return fmt.Errorf("hash inconsistency")
}

func VerifySign(md5Hex, signHex, pk string) error {
	pubKey := common.HexToPubKey(pk)
	if pubKey == nil {
		return fmt.Errorf("invalid public key")
	}

	hash := common.HexToHash(md5Hex)
	sign := common.HexToSign(signHex)
	if sign == nil {
		return fmt.Errorf("invalid signature")
	}

	result := pubKey.Verify(hash.Bytes(), sign)
	if result {
		fmt.Println("Verify package signature  success !")
		return nil
	}
	return fmt.Errorf("Verify package signature  failed")
}
