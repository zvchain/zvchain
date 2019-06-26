package network

import "testing"
import 	"github.com/zvchain/zvchain/common"

func TestPeerAuth(t *testing.T) {
	SK,_ := common.GenerateKey("")
	PK := SK.GetPubKey()
	ID := PK.GetAddress()

	content := genPeerAuthContext(PK.Hex(),SK.Hex(),nil)

	result,verifyID := content.Verify()
	if !result || verifyID != ID.Hex() {
		t.Fatalf("PeerAuth verify failed,result:%v,PK:%v,verifyPK:%v",result,ID.Hex(),verifyID)
	}

}

