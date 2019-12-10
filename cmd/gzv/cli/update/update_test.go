package update

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"testing"
)

func TestRequest(t *testing.T) {
	RequestUrl = "http://47.110.159.248:8000/request"
	vc := NewVersionChecker()
	no, err := vc.requestVersion()
	if err != nil {
		t.Error("err :", err)
	}
	fmt.Println("no ===>", no)
}

func TestVerifyMd5(t *testing.T) {
	targetFile := "updateTest" + "/"
	targetPath := "updateTest"
	targetHash := "dcaf21af4112339d33a4e700479dc89b"
	durl := "https://developer.zvchain.io/zip/gzv_mac_v1.0.5.zip"

	clent := new(http.Client)
	uri, err := url.ParseRequestURI(durl)
	if err != nil {
		t.Errorf("URL err")
	}

	filename := path.Base(uri.Path)

	res, err := clent.Get(durl)
	if err != nil {
		t.Error(err)
	}

	if isFileExist(targetFile + filename) {
		//fmt.Println("Installation package already exists ！")

		err = CheckMD5(targetFile+filename, targetHash)
		if err != nil {
			t.Error("err :", err)
		}
	} else {
		err = createFolder(targetPath)
		if err != nil {
			t.Error(err)
		}

		f, err := os.Create(targetFile + filename)
		if err != nil {
			t.Error(err)
		}
		defer f.Close()

		_, err = io.Copy(f, res.Body)
		if err != nil {
			t.Error(err)
		}

		err = CheckMD5(targetFile+filename, targetHash)
		if err != nil {
			t.Error("err :", err)
		}
	}
}
