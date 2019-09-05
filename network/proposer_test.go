package network

import (
	"math"
	"testing"
)

func TestProprosers(t *testing.T) {

	if !InitTestNetwork() {
		t.Fatalf("init network failed")
	}
	nodes := []string{
		"zv00c318574fd4756e72ab79ab7ddcd1bdd9e2ce3842253a8739f8ca227d0077b1",
		"zv00c318574fd4756e72ab79ab7ddcd1bdd9e2ce3842253a8739f8ca227d0077b8",
		"zv1e7ea67c13003e24f54fd15d5a8049c818bda43b2c4e1d0dc74cf0d6c97d8800",
		"zv1e7ea67c13003e24f54fd15d5a8049c818bda43b2c4e1d0dc74cf0d6c97d88a0",
		"zv2440ebda7e060b8aedfe401dda1f15a839ca6266e13c2185997c55dd099c1800",
		"zv2440ebda7e060b8aedfe401dda1f15a839ca6266e13c2185997c55dd099c1847",
		"zv2b8d09362515da99cbc58a1343e9abc291f65e1dd975e48a6389cb164b9b1400",
		"zv2b8d09362515da99cbc58a1343e9abc291f65e1dd975e48a6389cb164b9b1412",
		"zv2d59999a90e039ffcb513b6932704351a6af5516bb8b9f5cfe3beac0fbff0b00",
		"zv2d59999a90e039ffcb513b6932704351a6af5516bb8b9f5cfe3beac0fbff0be8",
		"zv2d7d49c3342f6ad421162b540c163284241da5060f5ec7c13a14dfc5d4463300",
		"zv2d7d49c3342f6ad421162b540c163284241da5060f5ec7c13a14dfc5d44633c6",
		"zv30476b75954edf58ca0b50f249dd4bdf82a77a50104a7660242980b26a773d00",
		"zv30476b75954edf58ca0b50f249dd4bdf82a77a50104a7660242980b26a773dc8",
		"zv36f0a907fcd26b0e4183bdc7414196f9c77ad723851e20fdec8b373135e77300",
		"zv36f0a907fcd26b0e4183bdc7414196f9c77ad723851e20fdec8b373135e773f3",
		"zv3997ec837b4410dc62a808bdb7cfd5a8160199ac01933b757e7f451b2f728000",
		"zv3997ec837b4410dc62a808bdb7cfd5a8160199ac01933b757e7f451b2f7280dd",
		"zv458f46b27e7986c6fc2634ffc46c4a900029872e01638b5e1a1d84fd4dce1800",
		"zv458f46b27e7986c6fc2634ffc46c4a900029872e01638b5e1a1d84fd4dce183e",
		"zv4ca7be3efc3c81abcdf268442a4302cae9a0127cbc5a2b9810488f4d0858c100",
		"zv4ca7be3efc3c81abcdf268442a4302cae9a0127cbc5a2b9810488f4d0858c12c",
		"zv4d152da247a90044d9e1dc38eae436131f64e35ae649e5800249c5ad988c0d00",
		"zv4d152da247a90044d9e1dc38eae436131f64e35ae649e5800249c5ad988c0d1b",
		"zv5c38dceb3fd413a9ec96d62c5e9d5f59a4aed55f2b72f88869f05b1abbefb200",
		"zv5c38dceb3fd413a9ec96d62c5e9d5f59a4aed55f2b72f88869f05b1abbefb249",
		"zv5c54d11011de08098ecf314374e3fd47c6f45b3259302788b5ba61523b8bff00",
		"zv5c54d11011de08098ecf314374e3fd47c6f45b3259302788b5ba61523b8bff20",
		"zv5ccd8c5bc888b0a207b4872490f6b090eafd614a2d86cb845aa5d225ad583d00",
		"zv5ccd8c5bc888b0a207b4872490f6b090eafd614a2d86cb845aa5d225ad583dc8",
		"zv6292e9b6e52edf22e17e774caa07c0880f8433fe02ef9f0181a85ac1ec966100",
		"zv6292e9b6e52edf22e17e774caa07c0880f8433fe02ef9f0181a85ac1ec9661e8",
		"zv6301cf53f2217e5b5bef4dfc3a786c7bcc614d31608406dac3b9589c43a7e800",
		"zv6301cf53f2217e5b5bef4dfc3a786c7bcc614d31608406dac3b9589c43a7e874",
		"zv70ff3822bca1f3e485335a5c2d808892cef07db8e80a16239bef2b71adcbea00",
		"zv70ff3822bca1f3e485335a5c2d808892cef07db8e80a16239bef2b71adcbea25",
		"zv75e8dd6b2a2397a84fab2bc7dde9e5993724eece0728d4a1e79b0db83930d900",
		"zv75e8dd6b2a2397a84fab2bc7dde9e5993724eece0728d4a1e79b0db83930d9a0",
		"zv7d0e67eaa410583ecc35fda617ae75a425f16882ae27d927d947c17516a35a00",
		"zv7d0e67eaa410583ecc35fda617ae75a425f16882ae27d927d947c17516a35a84",
		"zv83c036b9f51e707c8d437abf21a820f05699a9e469900b744567240743c4b400",
		"zv83c036b9f51e707c8d437abf21a820f05699a9e469900b744567240743c4b4d8",
		"zv85591e20b8138b9a5d17b32d6e92c1e04cc32027f47cffc94da1c90feba6f900",
		"zv85591e20b8138b9a5d17b32d6e92c1e04cc32027f47cffc94da1c90feba6f99b",
		"zv89a01709a82c63d89a9485a332a622b672d72ef6ffd1393fcce59cea723caa00",
		"zv89a01709a82c63d89a9485a332a622b672d72ef6ffd1393fcce59cea723caa4b",
		"zv8baa4f9d6d3e896a6e9bcafdfbfba20684865202b64f1482878b19cb396b4800",
		"zv8baa4f9d6d3e896a6e9bcafdfbfba20684865202b64f1482878b19cb396b4849",
		"zv9656b3de707f507161b8c9060210fa24cbb6eef1817a4177e1a97e2ce3a05600",
		"zv9656b3de707f507161b8c9060210fa24cbb6eef1817a4177e1a97e2ce3a0565f",
		"zv9d2961d1b4eb4af2d78cb9e29614756ab658671e453ea1f6ec26b4e918c79d00",
		"zv9d2961d1b4eb4af2d78cb9e29614756ab658671e453ea1f6ec26b4e918c79d02",
		"zv9e364fb7fa2b0e9b08d421f90d2a623905a5af37ae0f8620230417691a30fa00",
		"zv9e364fb7fa2b0e9b08d421f90d2a623905a5af37ae0f8620230417691a30fa8f",
		"zva123e364ea5e7c875e2794be613e310c167c9b9d93d3a32f26c04a18831e2b00",
		"zva123e364ea5e7c875e2794be613e310c167c9b9d93d3a32f26c04a18831e2b5b",
		"zvaa1dc19bce119fafaac96451c92600db1c2d5f5b56ed057652045b15ffd8dc00",
		"zvaa1dc19bce119fafaac96451c92600db1c2d5f5b56ed057652045b15ffd8dc8b",
		"zvaf9f16a75a4c4afa5e608d533e0b4eb9e67943bf179c4013916c246a15e0a500",
		"zvaf9f16a75a4c4afa5e608d533e0b4eb9e67943bf179c4013916c246a15e0a55e",
		"zvb00a3d28652aba54bfcb4a7427c22457c6c0076724102cdf7734f841be87ee00",
		"zvb00a3d28652aba54bfcb4a7427c22457c6c0076724102cdf7734f841be87ee73",
		"zvb286b2b4ba396d5f1476505c9725b43f6d0cd1bde392b4ceb0dcf32283850700",
		"zvb286b2b4ba396d5f1476505c9725b43f6d0cd1bde392b4ceb0dcf32283850788",
		"zvba6751e80f9c8ad978841f8ddd215fcfe1605e259c856e4345888f79c29b2600",
		"zvba6751e80f9c8ad978841f8ddd215fcfe1605e259c856e4345888f79c29b26a5",
		"zvc17fabd79191dbe2327adf54efeb5a46fc01dd57df5a1a1473b2857ebb792400",
		"zvc17fabd79191dbe2327adf54efeb5a46fc01dd57df5a1a1473b2857ebb792400",
		"zvc17fabd79191dbe2327adf54efeb5a46fc01dd57df5a1a1473b2857ebb792401",
		"zvcb7ca57650ba0f4b375adeece2eba4104c54c010e40d7e726b6b8b6519fcaa00",
		"zvcb7ca57650ba0f4b375adeece2eba4104c54c010e40d7e726b6b8b6519fcaa00",
		"zvcb7ca57650ba0f4b375adeece2eba4104c54c010e40d7e726b6b8b6519fcaa5b",
		"zvce843ee763f2b6f05217ad31c75e382f6fe70472a187ecbb6f45b0b4f55d1e00",
		"zvce843ee763f2b6f05217ad31c75e382f6fe70472a187ecbb6f45b0b4f55d1e00",
		"zvce843ee763f2b6f05217ad31c75e382f6fe70472a187ecbb6f45b0b4f55d1ecd",
		"zvd3d410ec7c917f084e0f4b604c7008f01a923676d0352940f68a97264d49fb00",
		"zvd3d410ec7c917f084e0f4b604c7008f01a923676d0352940f68a97264d49fb00",
		"zvd3d410ec7c917f084e0f4b604c7008f01a923676d0352940f68a97264d49fb76",
		"zvd5869bd928140cd4f21a359a0864e40a07a06752f39a6a2ba9be0bc640d52800",
		"zvd5869bd928140cd4f21a359a0864e40a07a06752f39a6a2ba9be0bc640d52800",
		"zvd5869bd928140cd4f21a359a0864e40a07a06752f39a6a2ba9be0bc640d528e2",
		"zvd6db0cf2ceb1c600a12dc7cf0b492a952d911607dc8c0810fe9800f37e462d00",
		"zvd6db0cf2ceb1c600a12dc7cf0b492a952d911607dc8c0810fe9800f37e462d00",
		"zvd6db0cf2ceb1c600a12dc7cf0b492a952d911607dc8c0810fe9800f37e462d27",
		"zvd983855d19e33df917c4b5d03886686ce2a0d02b0e913bfd8233544d2947e100",
		"zvd983855d19e33df917c4b5d03886686ce2a0d02b0e913bfd8233544d2947e100",
		"zvd983855d19e33df917c4b5d03886686ce2a0d02b0e913bfd8233544d2947e16d",
		"zve27532f917a2de44058d93b89d3a2f174be40a2dee89bae9fce65336d40a2400",
		"zve27532f917a2de44058d93b89d3a2f174be40a2dee89bae9fce65336d40a2400",
		"zve27532f917a2de44058d93b89d3a2f174be40a2dee89bae9fce65336d40a2456",
		"zve75051bf0048decaffa55e3a9fa33e87ed802aaba5038b0fd7f49401f5d8b000",
		"zve75051bf0048decaffa55e3a9fa33e87ed802aaba5038b0fd7f49401f5d8b000",
		"zve75051bf0048decaffa55e3a9fa33e87ed802aaba5038b0fd7f49401f5d8b019",
		"zvea31bcf9e87c2574d3c5ce10a08577e5eef11f9685b9d8ee69cb3adb86d02100",
		"zvea31bcf9e87c2574d3c5ce10a08577e5eef11f9685b9d8ee69cb3adb86d02100",
		"zvea31bcf9e87c2574d3c5ce10a08577e5eef11f9685b9d8ee69cb3adb86d021a4",
		"zvefb8f93125963547741915130ffdfa0df0549fa48b2440d7183ce9b0b3d37f00",
		"zvefb8f93125963547741915130ffdfa0df0549fa48b2440d7183ce9b0b3d37f00",
		"zvefb8f93125963547741915130ffdfa0df0549fa48b2440d7183ce9b0b3d37f3f",
		"zvfb29ccb9a49db4afffc892297e57ac22425df81291433977b78ca65574498500",
		"zvfb29ccb9a49db4afffc892297e57ac22425df81291433977b78ca65574498500",
		"zvfb29ccb9a49db4afffc892297e57ac22425df81291433977b78ca655744985b3",
		"zvfb8541c52d57cc00e898c8493fdf362c513683240f571bc99438219da80cd600",
		"zvfb8541c52d57cc00e898c8493fdf362c513683240f571bc99438219da80cd600",
		"zvfb8541c52d57cc00e898c8493fdf362c513683240f571bc99438219da80cd665",
		"zvff86d3a1102475e3a027d0f00347131483ef81b2d5abe8497551c9346ea23d00",
		"zvff86d3a1102475e3a027d0f00347131483ef81b2d5abe8497551c9346ea23d00",
		"zvff86d3a1102475e3a027d0f00347131483ef81b2d5abe8497551c9346ea23d98",
	}

	t.Run("TestProposersSort", func(t *testing.T) {
		proposers := make([]*Proposer, 0)
		for i := 0; i < len(nodes); i++ {
			ID := NewNodeID(nodes[i])
			if ID != nil {
				stake := i

				proposers = append(proposers, &Proposer{ID: *ID, Stake: uint64(stake)})
			}
		}
		netCore.proposerManager.Build(proposers)

		if netCore.proposerManager.fastBucket.proposers[0].Stake != uint64(len(nodes)-1) {
			t.Fatalf("fastBucket size is not right")
		}
	})

	t.Run("TestProposersTop5", func(t *testing.T) {
		proposers := make([]*Proposer, 0)
		for i := 0; i < len(nodes); i++ {
			ID := NewNodeID(nodes[i])
			if ID != nil {
				stake := 10000
				if i > 5 {
					stake = 10
				}
				proposers = append(proposers, &Proposer{ID: *ID, Stake: uint64(stake)})
			}
		}
		netCore.proposerManager.Build(proposers)

		if len(netCore.proposerManager.fastBucket.proposers) != 5 {
			t.Fatalf("fastBucket size is not right")
		}
	})

	t.Run("TestProposersTop30", func(t *testing.T) {
		proposers := make([]*Proposer, 0)
		for i := 0; i < len(nodes); i++ {
			ID := NewNodeID(nodes[i])
			if ID != nil {
				stake := 10000

				proposers = append(proposers, &Proposer{ID: *ID, Stake: uint64(stake)})
			}
		}
		netCore.proposerManager.Build(proposers)

		if len(netCore.proposerManager.fastBucket.proposers) != int(math.Ceil(float64(len(nodes))*0.3)) {
			t.Fatalf("fastBucket size is not right")
		}
	})

	t.Run("TestProposersTop30", func(t *testing.T) {
		proposers := make([]*Proposer, 0)
		for i := 0; i < len(nodes); i++ {
			ID := NewNodeID(nodes[i])
			if ID != nil {
				stake := 10000

				proposers = append(proposers, &Proposer{ID: *ID, Stake: uint64(stake)})
			}
		}
		netCore.proposerManager.Build(proposers)

		if len(netCore.proposerManager.fastBucket.proposers) != int(math.Ceil(float64(len(nodes))*0.3)) {
			t.Fatalf("fastBucket size is not right")
		}
	})

	t.Run("TestProposersAdd", func(t *testing.T) {
		proposers := make([]*Proposer, 0)
		for i := 0; i < 100; i++ {
			ID := NewNodeID(nodes[i])
			if ID != nil {
				stake := 10000
				if i > 5 {
					stake = 10
				}
				proposers = append(proposers, &Proposer{ID: *ID, Stake: uint64(stake)})
			}
		}
		netCore.proposerManager.Build(proposers)
		proposers = make([]*Proposer, 0)

		for i := 100; i < 106; i++ {
			ID := NewNodeID(nodes[i])
			if ID != nil {
				stake := 10000
				if i > 102 {
					stake = 10
				}
				proposers = append(proposers, &Proposer{ID: *ID, Stake: uint64(stake)})
			}
		}
		t.Logf("fast size:%v normal size :%v", len(netCore.proposerManager.fastBucket.proposers), len(netCore.proposerManager.normalBucket.proposers))
		netCore.proposerManager.AddProposers(proposers)
		t.Logf("after added fast size:%v normal size :%v", len(netCore.proposerManager.fastBucket.proposers), len(netCore.proposerManager.normalBucket.proposers))
		if len(netCore.proposerManager.fastBucket.proposers) != 8 {
			t.Fatalf("fastBucket size is not right")
		}
		if len(netCore.proposerManager.normalBucket.proposers) != 98 {
			t.Fatalf("normalBucket size is not right")
		}
	})

	t.Run("TestProposersAddContained", func(t *testing.T) {
		proposers := make([]*Proposer, 0)
		for i := 0; i < 100; i++ {
			ID := NewNodeID(nodes[i])
			if ID != nil {
				stake := 10000
				if i > 5 {
					stake = 10
				}
				proposers = append(proposers, &Proposer{ID: *ID, Stake: uint64(stake)})
			}
		}
		netCore.proposerManager.Build(proposers)
		proposers = make([]*Proposer, 0)

		for i := 90; i < 106; i++ {
			ID := NewNodeID(nodes[i])
			if ID != nil {
				stake := 10

				proposers = append(proposers, &Proposer{ID: *ID, Stake: uint64(stake)})
			}
		}
		t.Logf("fast size:%v normal size :%v", len(netCore.proposerManager.fastBucket.proposers), len(netCore.proposerManager.normalBucket.proposers))
		netCore.proposerManager.AddProposers(proposers)
		t.Logf("after added fast size:%v normal size :%v", len(netCore.proposerManager.fastBucket.proposers), len(netCore.proposerManager.normalBucket.proposers))
		if len(netCore.proposerManager.fastBucket.proposers) != 5 {
			t.Fatalf("fastBucket size is not right")
		}
		if len(netCore.proposerManager.normalBucket.proposers) != 100 {
			t.Fatalf("normalBucket size is not right")
		}
	})
}
