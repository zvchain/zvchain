package update

//
//func TestVerifyMd5(t *testing.T) {
//	targetFile := "updateTest" + "/"
//	targetPath := "updateTest"
//	targetHash := "dcaf21af4112339d33a4e700479dc89b"
//	durl := "https://developer.zvchain.io/zip/gzv_mac_v1.0.5.zip"
//	//durl = "https://developer.zvchain.io/zip/gzv_linux_v1.0.5.zip"
//	//durl = "https://developer.zvchain.io/zip/gzv_windows_v1.0.5.zip"
//
//	clent := new(http.Client)
//	uri, err := url.ParseRequestURI(durl)
//	if err != nil {
//		t.Errorf("URL err")
//	}
//
//	filename := path.Base(uri.Path)
//
//	res, err := clent.Get(durl)
//	if err != nil {
//		t.Error(err)
//	}
//
//	if isFileExist(targetFile + filename) {
//		//fmt.Println("Installation package already exists ！")
//
//		err = CheckMD5(targetFile+filename, targetHash)
//		if err != nil {
//			t.Error("err :", err)
//		}
//	} else {
//		err = createFolder(targetPath)
//		if err != nil {
//			t.Error(err)
//		}
//
//		f, err := os.Create(targetFile + filename)
//		if err != nil {
//			t.Error(err)
//		}
//		defer f.Close()
//
//		_, err = io.Copy(f, res.Body)
//		if err != nil {
//			t.Error(err)
//		}
//
//		err = CheckMD5(targetFile+filename, targetHash)
//		if err != nil {
//			t.Error("err :", err)
//		}
//	}
//}
//
//func TestDownload(t *testing.T) {
//	vc := NewVersionChecker()
//	vc.fileUpdateLists.PackageSign = "0xed591b1361d4820e04ee385eb267d7f0e5579918f238360e095fde798ca84e145dd56e0e9c46d81060538055a246094d38930f376764cd0309c946c7165876f001"
//	vc.fileUpdateLists.PackageUrl = "https://developer.zvchain.io/zip/gzv_mac_v1.0.5.zip"
//	vc.fileUpdateLists.PackageMd5 = "dcaf21af4112339d33a4e700479dc89b"
//	vc.fileUpdateLists.FileList = []string{"gzv"}
//	//err := vc.download()
//	//if err != nil {
//	//	t.Fatal(err)
//	//}
//}
//
//func TestCompare(t *testing.T) {
//	a := "v1.0.4-alpha4"
//	n := strings.Index(a, "4")
//	fmt.Println("n =====>", n)
//	s := string([]byte(a)[n])
//	fmt.Println("s ==>", s)
//
//	cv1 := string([]byte(a)[1])
//	cv2 := string([]byte(a)[3])
//	cv3 := string([]byte(a)[5])
//
//	fmt.Println("cv ==>", cv1, cv2, cv3)
//}
