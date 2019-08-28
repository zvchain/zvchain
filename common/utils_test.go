package common

import "testing"

func TestCheckWeakPassword(t *testing.T){
	weak := IsWeakPassword("sss   ...")
	if !weak{
		t.Fatalf("except weak,but got not")
	}

	weak = IsWeakPassword("123222")
	if !weak{
		t.Fatalf("except weak,but got not")
	}

	weak = IsWeakPassword("abc")
	if !weak{
		t.Fatalf("except weak,but got not")
	}


	weak = IsWeakPassword("abceer")
	if !weak{
		t.Fatalf("except weak,but got not")
	}

	weak = IsWeakPassword("abc112")
	if !weak{
		t.Fatalf("except weak,but got not")
	}

	weak = IsWeakPassword("3$#@!!")
	if !weak{
		t.Fatalf("except weak,but got not")
	}

	weak = IsWeakPassword("Reeeeed")
	if !weak{
		t.Fatalf("except weak,but got not")
	}

	weak = IsWeakPassword("123Tws")
	if weak{
		t.Fatalf("except not weak,but got weak")
	}


	weak = IsWeakPassword("123Tws!!!")
	if weak{
		t.Fatalf("except not weak,but got weak")
	}

	weak = IsWeakPassword("!!@#33TT")
	if weak{
		t.Fatalf("except not weak,but got weak")
	}
}
