//   Copyright (C) 2018 ZVChain
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

package core

const addressManagerContract = `
class AddressManager(object):
    """docstring fos AddressManager"""

    def __init__(self):
        self.guard_lists = zdict()
    
        self.adminAddr = "zv556dca04a59808f1598f90fabb1fa8a061ed1a636d270ff1a0c809e8aeb000ed"
        self.stakePlatformAddr = "zv88200d8e51a63301911c19f72439cac224afc7076ee705391c16f203109c0ccf"
        self.circulatesAddr = "zv1d676136438ef8badbc59c89bae08ea3cdfccbbe8f4b22ac8d47361d6a3d510d"
        self.userNodeAddr = "zv9f03cdec76c6617a151c65d393e9f6149cec59f10df00bb8918be4873b314cf4"
        self.daemonNodeAddr = "zvb5344ed02ff6e01239c13e9e4a27b3c5caf28c1d9d2f1fa15b809607a33cb12d"

        gl = ["zv54d1b771b1f885d6b6b991762305792b769ffa3a1a1a97766ee6192e6c93b43d",
              "zv2536b269d24b4fc944a54bab9267f8d54ca1ad424ef6bac405cd2968a83d455c",
              "zvf3ed19c33efa437ece5c51725c91f4d2509e0d8c8a1d61457ccd96e9b8e43101",
              "zvc8d159bddaba81ba8f2af3db30a751af5b684a566404c8f9223769505a1090df",
              "zvb8d61348deaa8e21f95b1ea8868c8572019ec7fde34c86286d30f543d037f07d",
              "zvd4572d9478ae700f06e5007b3a05d104a93f65fa45a39ca4cce996f82b379fa1",
              "zv524a87fdf95ccf75ee239a20f1641a86a0c8a8c21a98d74a0a599c9cbe03735d",
              "zv13405eca8191206e03118857d1f1d4a712ff0ac0769d1f419ba275fa5031642e",
              "zveb06f95dc9a118cba954b08ee6b14cdb37f906802973422da9d8ba1590906453",
              "zvfd58ff6abaa0bb6156f96fb5be79fb32af09047f3d48cce32a9461a351ab9865",
              "zvcbfa04184de671373634f1457cb3ad069a08ecbd21d0e0e4e9052c89bc6b4caa",
              "zvcac134dab856c60e57a3479168bc0832a0dd4f3ec4cdedfe70ffce2012f3d1cd",
              "zv04e58e7b18076da932cabf3087801fe65e7f69c28b941e6a7feda0a6b880233f",
              "zvdff85358163462a037251c96546d933e69bff59eeff909b0202089ee183ad61f",
              "zv9e493db8e70a29a8bfd3d2e22c8e52ad6e19aa8331504a0c4b32bdf908dde9bc",
              "zv6c3ca4abbe2e8cbf5e52285c952ffaf9e03d1316ebc21ed9b991a7a8e3f4315a",
              "zv621295010cae842f24b39076612640b6fac9814a7ad7a0797142df20d407afcf",
              "zvc0d4619313c63c93e8cfe3bac9469131bed4afc10bc2d1655bd925ba135d36fb",
              "zvcfe312994eb0b247c83ee93380945071f0e4c36ff226dca5b7b680c6867974c1",
              "zv99f3652192f78f75b9260f049f6101ee80fb93e0ca212017c2328f69c2982fed",
              "zv784ba0f006551dc002a1565422c587464a36db8c7efceced08858b91241b2a37",
              "zva734b2ec4b4dcc08f759f0fc55b170db9db0050249c5c5103993f99c4daef931",
              "zv52cdc1e786b1d0b9a204d1e5ec729bd57489d9b63b89e20dd320e576c29913a2",
              "zv065db381e3cb6add5ae24562fcb0c34cf06f6d7533a9da2fa03f4b3af7414b77",
              "zv773efd812d761b402d3118bc40be01738b6abbd71560d559397bca4ab545bce6",
              "zve8d365c863ee9dbae9e32f2ca0f1be153410696acddc0bda5657453d4a0b4385",
              "zv071360b91dccc7e5de83a899efa7ff3de191cf09d6afb42b11bf5e8da3f8b9ea",
              "zvd52102ae17302276dd81af4a33d65c2eacd912839b8b192dd992cd6e5c7edabb",
              "zvd99cde29584217562e2dc1dcf22493227d0e671a439caaca29d86f951250a034",
              "zvfa4b67903b421227d976fcecf2b94c0c44f007939a35243118080f48f127cee3",
              "zv6ac7cb76c37ee73e07e8b492127af1412e5e34c68dd612dd33abefb94c2f081a",
              "zv3aed0db21eab14fd1c2e8aef8bdb8e027264358bb086211d484129a1068912d7",
              "zvb44c8856755de620edd08bbe917e6c3f826daa5c14f6af62ac98aaa229dc43f1",
              "zv13e1f674657bfdbc49cdcb822c6510c8f627c19d0e1e64e700f35e4333a1c6b3",
              "zv2c6fafe96ae672c0f80141e851c01cb09b4177ce62d85c07378ab79dbdc39106",
              "zv284b2e47df4131bf9aee1960dc2fe3e90495f441caa5a2f7c330cef754438bcd",
              "zve630bb791e21159b85d6c0e5e46eba5a3acb01e8edefc016e289b7575b3832e3",
              "zv9bd23378668424bd081f5b4b120692c5113c5b18c99c847a8b8d27a0ccb3d3a5",
              "zv63f1395e905137fa7df52d292b950dd8f63dd58232a0475002d4b5481e7e3b97",
              "zv6fd274f6d57ff001280e35f74d41ace8350dffff6a7aed9022469b536d5acb68",
              "zv78ed9d1c142a887f8b3c02b72381ab0d193c35ea9e938241862b95fd23bb3eb9",
              "zvabb7f885bc1dc1458c64a208015e869deea41ac5f95725114814bb62de4b0599",
              "zv6ffcc03f41fe91404ce9f4611b6b6ee256198241fe2f690d748ee29c38207fc8",
              "zv218962877d1a4e8340741ccef9eabed48a50cf54c56b5c0f2eea3a0b914b3242",
              "zv7c2197f325d87a7af29e721857fb72b03bb58fe78fc6b4daa24dd013ab01d0a4",
              "zv80b5f05b44d7ef2853c4382f6ee9cff0ffd8a6a7f439f6857e2259f94cb2e169",
              "zv48fa48f7ad3f00a7006374c6efbfff5689ac98cdf2c5f22f36d6432df95039d4",
              "zv16cc5581a18316b9f97698c546c4808aa0ab621a16080d6371e2940659d95928",
              "zv7293afa007e587fd4c90da232ce19da328ee5ddab695348d773c0ea63f8311d1",
              "zv90e32823718106b920bbd2d82713f4bb25519a61fb0cc331b340bdd102f3c684",
              "zv325707fb27bd055146608fc5ca6add39dbccc633324cfd6870473c9ef02d45e2",
              "zv046fda00f07c8875790f680102770bb18a5508e1c8e15d328fbaa76c3318c252",
              "zv595f326d40a68f60fe63ac5d30a15bd4eff128beff1a7ff223d6b71277e1ef66",
              "zv71f9bbe28a36abd9d973130af2e43a22ecd6e19e0e620d6a44aa6aaa6535eccf",
              "zvc95adb810de6500a3ba4938b64f41fdb5f9c7d86f6fe70c60aec7b36038412d2",
              "zv270054f66dbd89312ac6af9f48f40b381c3f218d86d077bc46eb745945627f22",
              "zv8417be4cc057c8d0b92067e4aeefb51fa053fbd93057cfa044f7b9692b307c0b",
              "zv4b050dc496a6ed550dc4d0d995f790eabd68b711449c31d654e5a9b8ba4a8e61",
              "zvdcaa1a74fadb585daf2344fd4226fbd744f4f99d9b29490da9e9dbb057465849",
              "zv6e6fe76686195f424d73f3364fb382db12e9798d38536a2f6c7065f9be38d557",
              "zvd8890e8fa52b40fbff77a1eecd54dc45db76cb6b8be4b82df98b772059376824",
              "zvd84f5be4ba4639c7b9db93ff9a76f31876c0da3f5c1718bd08305b70344dd720",
              "zv324a4cb21bcb907dbb04dbc4ac4a55925cc7d870f65afeb144dc4704d8a6991c",
              "zvc6699303a4f9e9c08a09e8d77f3fa343620db2303aa33db2316053bc7cc1e23a",
              "zvfaef5124a50f30c0116f870b94ce324fa0563cb2230d4445efc59ef69e972e12",
              "zv7ee213ce427173058b81fe6d0c3b54374f9f1e2a2f127cf3048b02d9e905ec72",
              "zvfade4287b953303b1163e499a03d25afd16020cf4c7258301c63e5c0da021c2c",
              "zvf5ae67eb6f2ea3ef8a30fa078492e5d7cc606ae324b89f4d2d4b26e3a7d8a0a1",
              "zv20e7196b8d8f86d7ff1f96b24fa18655c3b8d72dec0a8be430561acb0a3dcedd",
              "zv932b158e2fc25c71804450d3a6be12cf55119c16f5f63b391d0fb7396037eeff",
              "zvdb8d374270e0f308e25a6b07b51a441e5e5f11e4405b2e03a2c1d236075b02d3",
              "zvbd67ce1a6548f973141b0459b157dc4de17d52b2ad24a648da0443b9cc1b9734",
              "zv2f2414f422e687c850814fc02d13eb0e342917900ce8724b3e0252a47445c916",
              "zvfff4fda48903493e2b1dbf24510e35235b1156ecbb8fa72625a472aaa6e59376",
              "zv51120823db9985c0d84ebfa17f8f6d59e1454efa39d283d51f4d75d1122eea13",
              "zvba7953dc53bd4b6a7ba14c8411cf20b4783321f12774a01b0ce42be3a5f9e7a6",
              "zv03138472e031650f142d0e99d003b3a4b2f8c014b237add0bba2a85138e948cd",
              "zv9ddbf28c93832f52e823220751c6292cb348d80824b71ae9652079d15a07fac9",
              "zvd130b2a6a96a7ff61f41267436cd17ea7b5114116c970ca5383dd0fe5b392d27",
              "zv011e8b7c06917d3e7c3a164e066e126d03fb967f141698a8a2bf8de4d29a4857",
              "zvd1a2bebfe1934b3646a77e58768c0d09f6436d03821fd0bf59de468661a36bc9",
              "zv5d9d98b6dbc942518c2634a0f8579a6391b3cff029101acbfedd260575f3766c",
              "zva81ca6d01d0ae35fc021c673e2eb9411cbf77b05728da778b6b5235390a745f4",
              "zv95801781376be4703d11e26780879e05961d8cec07725e34d94893236003e17d",
              "zvbcca6ece322c588e1e23d382507fdd600ad3256a5633066e3e6f4a015ee601a0",
              "zv71fcbd30dba358be5e2b4a58eb4aa371987ec6549a6991f1e0b8f7e927d899ae",
              "zvfbb43821dd642ba6df8a967230d7f9067aec4ee25678997188f3d98804edf802",
              "zvfb430502ce41c03d182d80b1e3573ecabb0b6ad107296e944e8c43f1c49f3dbe",
              "zv1a316fb4f1d068bec62fcead93f8eed1075d227eee354f5711df489d1440f10b",
              "zvc1b080ec5d69496457fac1f538569c60e05f6dfe90073839b2cba1a6a0a4e30e",
              "zv6acfc85db3878bb8c632401562c9f1b8c84ae14c249eb72d886720160b3cddda",
              "zv232fb0beec1cb5f1c1b540e639a3f1b064babf32cd469639b4750cfa75a20b20",
              "zv0a5bbbdbcee2a3faea9340aae0d081571864642b6ab0b4e0ef8f1cae6f6886ca",
              "zvbc861573b561a07a54d1a4addd4c17001bd9fee05a999224d417baaa43f30ffb",
              "zvcefea0c721252c7ff3e786f9b54446f61664d52e4d305a96afba9f26930572e9",
              "zvac1340a49fd9d02020a4d595a918480bb8c8853f3a2f83881b66e5ce79d7c89e",
              "zvcee079fcc5ffafe392aeb859e73f645e98860f35b7674c12eb3011e47ca2a2f3",
              "zve9cb6ae8f94388b38b9fb427d2d4b7a3dfc4f45e5f8549781b67599ccdb7d3f9",
              "zv9aa7a0c4ed046e802f8591a92df61401e675212d12f76576bbb2f17412e05e2a",
              "zv08ef1fef2f7824752c0b1a547ac23a5b6f6dd8111c97d5de3d48d0deecd8946d",
              "zvb7c06d4df91e513100b5388ba66df8f802cc2b7672e7df240c32ced1f0dd0fd3",
              "zvaf66a7a99775c4f3633ad3dd5b27ee602f6a6ab803fb4f7b80d8b5fda272e12f",
              "zvf91f2f61bac4bbf82ff3f1a9ce587549a22ae46b5c05d44ce62c708e32b4712a",
              "zv76eb7fc1ffe67b86c11ec69d26c431633dcff4234b25b03b4b151cbdef1e5240",
              "zvd9781a11267030dd6e87f1fe81a756664b567c4029ddc0faea70c74715229ad4",
              "zv846b45a430fcd5fa822438d1bd2a8a864a094ebdf8276be4df78469e9b5224d7",
              "zvf93475f4f3ca6411f64f6ef868bee79a050cbb4d5f2f41a5b51fdf942e4f67de",
              "zvc9102297b102359405c78c53c09d6f7a884bcf0da0bd00ee7d7f472ebeb65ce2",
              "zv656ee3f5b14c58a4a5af0bc408e2fa16ad5c49e6f27ba19c3b5d68b8e82bba9e",
              "zvd0a0a4bce8ddb1416ce2cd5ebdfc85f86c724ce5d39d3f19005831888a3425ee",
              "zv55c487726af0d69a7200dc2816d0d6f280968d04fffaff3021a1113c2f1b99e8",
              "zv125a542e5f0cf1425b03d34eba30e8b311f094caaee057faf7ab46618b3c92c0",
              "zvf521262924f853104f578ad67743b3238db78e645210ef6f7c3975d8c91b5a50",
              "zv5fd91d00f0d0b8e91df48ee8b94c947532f103c80f1bedf66649ca9eae3e67f2",
              "zv173daf8b50a8a4533b4c1b42c861f5eba1319f13a4ba6c354f67f1f1672028be",
              "zv28e8b9a3bcf703ebbee51c1bf8caf62b112e45bb9655fdc967ec78928544b24f",
              "zvfaf52a17701392990db804322d14a8ba1ff934f60dc35ae60f1e18ff41385a83",
              "zv41896e3486a4039ddd04c2da4044533c5f5351177035e24e99ad46524f0c8155",
              "zv553276d98adb5e33d99f48bcb7e83eaa12c05d8b4a6159bfa881a00db5daa6d3",
              "zv9a80c747ddb06b81074c5e6dcc3f50a07e45c52099294d77b033b47216324c12",
              "zv5db3ea43fdbca8cf94fd09f7e37d6fa7358152aa466cc374bbf3a3e3f9d04efe",
              "zv56dfe04e44e60ebef758d952e21037272c5de0574bab1ce3ec8a53082ba0f991",
              "zvacab7fb97e86e4da8750fe1b344da24af258249daa7da7f7bf7e312cc66a906d",
              "zvf877e5cf693717097dd73ee2185eb195ea531746fc9d915210856364e9a47a44",
              "zv180c1c18432109af08de4dee497714feb52c99f294340f7d8919024f7d8eccef",
              "zv7cbf43e533a2d0ac2e2035ad7c791df54851a3cf842c6c9c5cd9d6d47cf58c64",
              "zv4517158f1416e1efe3272c3a735a583577dd3ae798b1bb6c8b599785c07b3ac1",
              "zv9b7c2a4949100e62b86d7c2cb40a4171b9e09edfc98b6da08dcfbc145e92117a",
              "zv3198d4a0c548cdc6ac7d0fd97836fda85fa9e95472634a0ade28e33e59745bc5",
              "zv83b558ccd4c384a9b4d579cb53b4e2b374e422bf9d8a79f8e0a2ee3e6eeb4961",
              "zv9351b421d1ac4dbbb6fdfb0c1663a40b535285d5f692279f85618713b12eaa6b",
              "zv6e06bb582810548c7d59d3c48975075361492ef5f496c8031208371ca26a7b78",
              "zv836fbd021c7df2a0bc17b6d8b699a8d605e501373264a76f618bece8eeb98762",
              "zv1311a88facaa02111b9f35da9d0b791b472a13d13484dad9aa9657fb0f6d35af",
              "zv42835066a4c1cc2f96ecc180f0dad1c48acd59785dc97c268e8a745e534411ff",
              "zv23c8d5d0861d6fb1dfa7eb650eadf54cb0306a6d31a8cbc9a012c3b1a3413d51",
              "zvca30cef7332c7b80bf439b9c56cac6739e6753efce2fcddd6c073932b22e8407",
              "zvffe2d73261fa4a932c1362d69ac29385f5c7f71e79452fa9c5eb3d7de317ad70",
              "zve7ca48daf81a4ceeba990ff8f90f70c95a7663a28ebf0fe9821a94815d835172",
              "zv881c5ac00ef1c0c734edfbeabd6b1ba2cafc27b72ca9e4da9cc105b899c1d58c",
              "zv3373f3cc48ee9e14469b6fed89f9dcb6126f5438922d9efec582b5a5ec1b0347",
              "zvc5606b56dc8bc8ef719eb1864a701d9c1970477031b74daf5309f20d4488ffaa",
              "zvfeec73451cea2fefdb9a195faeb4b846abe48ecaa9116bcc31626cce795d87db",
              "zvc431a17f6af042cac51f30c537006b6212127e0dad3859fd39198405fee69980",
              "zve94a4f0ab23bfc4d7d366dc0f59e172d74afd5d39629c895238ca273d0e43120",
              "zv61be4376744810587701dde84fe068dfc4410f85a507a56ae8f18f6020ddae55",
              "zv84aea421bdecc8d2e889a7849c052b6ff5b543c3a8869981cdcc07fb21241333",
              "zvbc8bbfbd54d4616b71a028855bc572a2164e487f629576b7e6f7552808de05db",
              "zv04c13e66ee50f2fa9cdf32167e39304a44a22431ad71b0d6bbcea56f1b9fa0d5",
              "zv484e30e8035c082fee021bee3d29bf2731f6222a2d67b6bc03f544769c77e830",
              "zvba2ad1b228bcc7cd4ae7f5c8635081389a3923b80d9dc5c92ec281622b27d982",
              "zva129c374ed9a86da60f0f7b8d65728b0e871f26a6c5e9527bb90b82cfe5070a4",
              "zve24553008b7d0cdfd61cc55db8c61f64c09e2a3216b8ecf51ba6eee0800fa46d",
              "zv9686bd06ec25b24fa62e9ad2d2965c707693225c8ceacde96084cf9802c6f866",
              "zv43d9cac73030702588a67ab3d85e2778d19e6a6de8a921c8cfd45a77743b8c3b",
              "zv75bb04ce097cf4851349fda8437ebc39b0dea4b407a1658539f5779e32bb3bc9",
              "zvd5578d7a9084a09f35f15934dc28076709cddbe7b28b6a76632fa5eb4d3c067a",
              "zvc998b43f0015fd14a84cf5ccc84ccf9d0933d994a2cef21f19ed2fc718b42729",
              "zvcdc84d8c9eb745ad1a50b63e861e8ccce65206c2434814b6104f060f74e6acef",
              "zv79fdc1db2e05a2b7f91c6a4a328d0a242a64e7f0de082ab556aff61121b3a53f"]
        for g in gl :
            self.guard_lists[g] = 1

    @register.public(str)
    def transfer_guard(self, guard):
        if msg.sender not in self.guard_lists:
            raise Exception('msg sender is not guard node!')
        del self.guard_lists[msg.sender]
        self.guard_lists[guard] = 1

    @register.public(str)
    def set_admin_addr(self, _addr):
        if msg.sender != self.adminAddr :
            raise Exception('msg sender is not admin!')
        self.adminAddr = _addr

    @register.public(str)
    def set_stake_platform_addr(self, _addr):
        if msg.sender != self.stakePlatformAddr :
            raise Exception('msg sender is not stake platform addr!')
        self.stakePlatformAddr = _addr
    
    @register.public(str)
    def set_circulates_addr(self, _addr):
        if msg.sender != self.circulatesAddr :
            raise Exception('msg sender is not circulates addr!')
        self.circulatesAddr = _addr
        
    @register.public(str)
    def set_user_node_addr(self, _addr):
        if msg.sender != self.userNodeAddr :
            raise Exception('msg sender is not node addr!')
        self.userNodeAddr = _addr
    
    @register.public(str)
    def set_daemon_node_addr(self, _addr):
        if msg.sender != self.daemonNodeAddr :
            raise Exception('msg sender is not daemon node addr!')
        self.daemonNodeAddr = _addr

`
const addressManagerContractTest = `
class AddressManager(object):
    """docstring fos AddressManager"""

    def __init__(self):
        self.guard_lists = zdict()

        self.adminAddr = "zv28f9849c1301a68af438044ea8b4b60496c056601efac0954ddb5ea09417031b"
        self.stakePlatformAddr = "zv01cf40d3a25d0a00bb6876de356e702ae5a2a379c95e77c5fd04f4cc6bb680c0"
        self.circulatesAddr = "zvebb50bcade66df3fcb8df1eeeebad6c76332f2aee43c9c11b5cd30187b45f6d3"
        self.userNodeAddr = "zve30c75b3fd8888f410ac38ec0a07d82dcc613053513855fb4dd6d75bc69e8139"
        self.daemonNodeAddr = "zvae1889182874d8dad3c3e033cde3229a3320755692e37cbe1caab687bf6a1122"

        gl = ["zva5f6e5e74b5d32a64231bfbc985b8793eef1ac7fa5879facdd17b3f3543116db",
          "zv46121ddfbe4bd40ee8433330b28e85677b2f9e911f5b265340abefc51c6e451e",
          "zvd08dca92917feee8d417bee8c4d887ccba5c4b3a333d805d296339e4ea569131",
          "zv320a7ab45070d8f9a1e75199ab621fed85b52d872a6757eaed17f3e5247ae6ea",
          "zv9aed9c5be0b4ad4cc9dcc8c3243e0fb2a1a93fcdbe68bd6ea50ed9f16ad7e586",
          "zv5d465f1275c6490f69374bc39e29ff384c037386845865e62b0fcb9172866b8a",
          "zv852058f1e8949cc04b9bae34fc68cf5773457874429d98703503d791bb27c2db",
          "zve4f0fd46d6056eff002bef4a83679e92239c838b115e6f0d87cb060c2697b83b",
          "zv9ce8342fb2104f8a13d2ead6318a4b40c1182ac1597df289257b0f0fb0ca2e56",
          "zv9459f7bf39adbb1d6d45cd5ba89dd88b3722c59683bd2205eadbe50ba01ca96a",
          "zvde20447b9e25020d4b7f558269591ffdd1c1b413c50e761743d441d3825fcc70",
          "zv8ce7ed725d555c7bfad61fd339d884e2b6772a15e0d14309afb26364f4a6b1ac",
          "zv1f0c3936ee3ec836b084959b4306e58acee1af0e0135665d9e1eee039c91c2b5",
          "zv50d4187297ee9a5aee111bc62d3458d832de994786a4c71707e13b4bcdce99b9",
          "zv72d4938decd5e3a10192d6a6bb40c7ea1b11907e5b49adc35d829d38b36d3ab0",
          "zv411d273e7c418b8a5708e2265b420d876400fba46aa66a0b9a99e897d73f888c",
          "zv195e99556c0c19d5791cd9c4848b6fb4762638e12f50a162044e1cb68b1af814",
          "zv43705f8283ceb8bbabee0e3401c9f7ffa4bcec65e46a7f7e3c989bcbbd5f929f",
          "zv892721fb0cb1c6678dc81f233dfeaac07201fbd6870a3c9e731f2b9c8a25617f",
          "zv8315747bec1b3ec3ffbe68602f948962235d7fd009ab1d5fd2ff01d8acc80b9a",
          "zvb5bf381507dda82f3fdc7b59a1d05580c6246d56164aa3120c8f4131a6b88052",
          "zv7136f7ab9edd855e13f4aa635a1eaeea5e567523c35a6bbe9cd9047f170392c2",
          "zv55f56288b58ba27427526925a3f4a9275114a98d7913348c8dd2eb8929dce719",
          "zvd32ab2b120a0d3032b532792e0b547b7d518a8a53273312181b88043e1133d0b",
          "zv508d5027ba7a30c99c5abf2e380f13b5617bea8cfe9367c1c4cb6764a23d4113",
          "zvb82df0547400ae3acfc15d1ef8cab3cc2ce020f56763942ff8cc8b7558452835",
          "zva53f39a968de38136bf6fe7e4e7615ed9352b111daa22fe9649acacb763f5734",
          "zv734d02c843069b085c681ab15459806dc62324678261b70506f1e579b1eb13ad",
          "zv787fe86df58256503279714a71329e63c207ad90360547883b2f3da2eb3d1d82",
          "zv821142752f645d9f96047c7b31da91eaf036349d04a60c0d15b7b62da1f5f526",
          "zv6c909dfd202801c22cbf7246fc72b6af6069e6074c7b4a0c65b236ed79423672",
          "zvfd2067ed83b1a2331aae73dca3d7fa56083a0092786f0b2df0fe2872957e641b",
          "zv3b0d886c529c9be95e1f24e5654756413d7dfb407a1ddfadac1fd7d29ccf125e",
          "zv05f0bac9a9f45d9ede0acd41a7d1a232889de6f018cdf4f7bb24a33b79e72436",
          "zvdcb35c5c2d4530c46544d20b1bcbb0c5fb3caa6109b7e49bcdc2af7f4ff1d79b",
          "zv7a845d02a8aa71af5b02b8817ec85226329844b70ce14c24f651c0b9ad4e3ebb",
          "zvb397c8bfcaf12c1a89722b3e8e4b8f07aa93dea46154cc12d4ed5c37ce57c309",
          "zv641a09ea7dbf08a1ee276d3ffb1cbe643f5605354050de2577c1f95d8a3b1292",
          "zva402ea06367c9ec88d850faf3a50b3ba033ffc04ce0ebb2b6521026fd8aca4ed",
          "zvf0f76d9e79f66c48a2c6bf980d90c8cdea4201a1c066d101a1a8b7213aa58505",
          "zv943273e93e116ed1a4d22920e5eb1aa8965d5be3b3d8b9b4b18aa766b20977eb",
          "zv4e20a3f1ee2c467072b057da0f53567471530b33c68a02a5390952b00003bbab",
          "zvba030321d1c9295267bc0454af0518d65854b4d61ece0cc8d100a48597b90e9f",
          "zvb76482ee38a67e215381a20d99e0bcba14e0e18b59a386da02e0afe02a67e357",
          "zvfcee910ffe4e93155562b83127740e40a97f1340919f5c1e3134f536140ff16e",
          "zvfc40b4422f2cf3d0d3aaba48373ec0b2798f9bb34a1b915ae6948b83bb9de814",
          "zv57a772b4d8bc034fca827afe356a1b79251c5b560f9474f3bd205ca89a438070",
          "zv770e968502cba14fee62581940ab77d910c0cd3b091299a466e965564630a617",
          "zv9d67ea37ebd4d4cf668732374054cd3958f6ac457fc4354f64d12a7e4a79a1a6",
          "zvbcf8c97a2ccb586bd1a6b3236718af340e7961b0d017d6311dc7c121b299d823",
          "zv3a4d03924b12bb6ca3e707e69eb42af6662c97f633e339aacb6f78cdc23f74f9",
          "zv7a1c18b3b6d700389b3c489c6535a586d8c0807cb120c9dc42e4be6648fa7080",
          "zv6ba8eb27a1aa0f771207b962263a783c03201364e0302e72d4287e4879400c4e",
          "zv148cdac54e7a49f0fa62f44bdd875ed04de62f37ccd5116955aa9507a83a5d2c",
          "zvfe6450946e3f2db6d98fcbab49d776543782c71535f1bda6bc65b4a02f589ca6",
          "zve88210867741a36baa84ed195ca68915d0763eb45bc4860a5f7d9df3caf85a1a",
          "zvad0a1abad12516580d9f93baaaaef91a0b72fe410db1dd9dc532d9bfb810bde0",
          "zvb7248ad1e069d558b9ca7e0049e7fc46e9aee3c0a6492156436ba2a2695440b8",
          "zv34c14ab35113813cf64cdb368b6a76b6decd3860e79f96c7c5e20661d23558fd",
          "zvf4cd6bee2c8bb04e58c573cce1628ebefd4df09de634495fb23473bcc9291041",
          "zv3582fda839bbbcb00a93b58df08074cfa29b57b22c40c24ea8e37d65f6af4be7",
          "zvbdc989c17d9224768542170605f47392328f7eb62db0457b3716bee0e25b0ac8",
          "zvf874a371e5d43d4975adb8f77fd0f146d06383dbcc8d92c34f5c9a915f673227",
          "zva87dd9e2eb3c7487df9f44411da1d53cf2f7afccb2fc204937fd488c681e3b1c",
          "zv76c08805a7cc9c5891ab2da48b72c9d08ea326361e8d896d0da555e22b394ac4",
          "zvffdb9bd4afe8fea62cbe9159fba0c068eae47a01fa8ff55162490c2086417f43",
          "zv1428bf99a49a1dea6e190e6dcc7285f712c44bd700c28eaf2c030f3823fe6486",
          "zv104c7a349c62ae84c3eca10f42e2fdb7060400a06fccf2b5ccdbb80ce2a04ff4",
          "zv1612125202446b01f02b8a77dd47f83e62a8593a86a16eec68ef2bf86892e638",
          "zva4f1fb759bebfe87388fb521d9e40e2716b8a3f68395edde21954930fa3216dd",
          "zv4b7c03a829222e5a66ca0e7bdb3df33a58e4cd98e36a17489e4a1928dfa2e965",
          "zv18ce032088d8c9ed7a71616f82dcd751b58155e8d978c68f355abc8b6ebe9c65",
          "zva4f9f334cfdbba691093de871f111c61034a59e61a9467dbfadd53365fa15376",
          "zv0cd9d48c7bee65c877c110cc13e1c98fcf4048e36807e416be8270bcdc20ca10",
          "zv4ab557d1098629c0dd544a881ce72b1188d4ae28b38c0ac6df393bb094142e76",
          "zv3f0fe205c8874959c8691e07522309e2883095471748420021af8b707fffc260",
          "zv1d4c1f1c8029652cc24d6f72e843c95979143e8be6e380b2e61a11a2be6adf45",
          "zv8b0180cd77b942a0ec15400571a7499a41733e79c37e24d9c0a929c2f28175b4",
          "zv1fa307b496cd91f051b5ab28226680b208df8aa2e213bbaa5be31e39730c8dc6",
          "zv2609debfa6734e4e2d1850c6d34ec5398e8daeaf7fa01993086495d456e2e508",
          "zvfafc17482145e132c33d1008c5c7f8003bb19b749f1f7bed1c443c89dfca0442",
          "zv7eb204b9e18c58cff5d51330ebe72763ad42a6cc9b600c0b32bcc28d49ef58e0",
          "zv59cc754ef09432937c4a29b5a8d2b7d0e9165514e4c40fadd7f7c1542fdeb3d8",
          "zv551fb87b67da4d56761e8f4e37311ea7b96403579fba0ad1a583b9fbf224a612",
          "zv73627afddfbf09bb7dd80fccbc9a1504386c9562d9be7e8f931fe945ac9f0da2",
          "zv3edaab4db8ae450178d1199d9cf8196278ea0fe6311315532a07edb8011a8fc0",
          "zv49a26b43dfb330c66a95c625fe4038f1e84027d97386b7f09c9382c23161eb76",
          "zv360b74c3c72929109112f6d85c4682c2eec901678f578936c2995e951f212d46",
          "zvfad695454c9654d866feb832b6dfaf3298f04629fb03ad63ac700e3be9885ddc",
          "zvdf20d4f9bddb2de38f1e4ceb51b645ba15cc586b4f3301e00267fad45b4cd0ed",
          "zvbe8d56d5564b6eac82eeca304b085e6c6d81fb8ed9a2a0006327944534bc595b",
          "zv7d7067d6af623cb520a5662bfdc8b06c23f068b46775d5f5904eb868e2586145",
          "zv3e05ea8584d8a596837517e14f15c033e83dab0224efb087000ba91bf87869b0",
          "zvc5a135221cfbbaf9212c72824ed1dd0c44a97cce187e1cbd70a652e16424621c",
          "zv70421cd6a0e33da0043be27910f2ded8f9fed470293c5c9de15311b84424c323",
          "zv16d90fb96b95bff740b89d5564058883f3cef0c81e2739f726e98a82a6a585fc",
          "zv9e9c63d2d4e37556d8a08565fe42870ce715a24de4dfbf6e945b6674b863ba0b",
          "zv261ccb59ea5de942e7e1e30a8498cbf32eaab89306fe9c5399a7807d6e0a410d",
          "zv08228104c85117ba0a958cad1bbc9344c5cf5f9bbfce22ee65b6cfdedd9e202e",
          "zv8dbec68697928951186486aa5bfb491dcaba5e68701216f46c9dca08477d65f0",
          "zva59347fc9e961e6924c2be013721442b6cd22ca44791c4cf9069fdf6125f6ac0",
          "zva59c2a94ebaab9bbd46b9ae01b95c7a37f84d3e8547f3b7afca4b871ae262c52",
          "zvae551b180b0e6fb92b1bc69795cb908b883701f167587bdd3067a369a75138b4",
          "zv27c12da625385a326577cadf79dadd8c16d8488b6a16a05ee185b12916dc7c73",
          "zv48b44bebfbfcd22e5716541216fa37536e557f9669fe616dad9cf141a1365fff",
          "zv486731fc6a20576350e5d3792d50bb9a18dbf5ea11e340fc8998dfff6848077c",
          "zvec2b4cf8a372812f6fc6949387c2b08cfc78b65a9c1c102007bda197cc4bc512",
          "zv9be7fcd9636eead11cb9dfa8a16c3d33bb91e5267e929571955702ffcec22b5b",
          "zvd35a154d88576eb9ed337ea2767407361e13699cb295b73a948be724b145cc73",
          "zvdedd3a4a62093a1e54b4208b1c32e2bc47c26543842bdf02cf44bb6f08447bde",
          "zv0f6cf8ac989ea52ead7f245fc836e297742432a73207d2b499c1f3abf6d65b53",
          "zv97bbc940e0cc4b419de9eeae2f1c90b2a42d40f1879a34783d95a1b6b3eb4c9e",
          "zv732a7ff925bdffd31a33f2695ebbb9b569a2a555279a6e23314a0454a0ca850f",
          "zv97140021e15989a3a8a601333b15ffe087fa156f7f1023990221985ad3133f13",
          "zvb7e88dd0d18c9e7218565f626beb03fa806a294669375d776d3569b44f0e832c",
          "zve26e71bb8fd28f495e81fa64414cf8a53eb84ca34b0a39ba0f423e387f7271c8",
          "zv2e543c7596be73c62e0f567ba8d990789ad05028c1fadc5d9e454be83362f411",
          "zva81ba93b4d71d90ceb786dd2e33cd6c13e387cb0173a28db9b4981d82cee9cc9",
          "zvfcd53f57a4e0905cb398ba3b952adf1e8ff6d898686687ee12c2a605674b1f5f",
          "zv03ef1ca26a6af62123e9f8e2ef2c6eba56dfae6134864561fb1e2a3dbc3570c8",
          "zvdca71b3ca82cd3750676969c83942ce9a90cc3baf578f6576540dfce59549fc3",
          "zv64adf7ad9b7545fb4cad471a3fe5369d41b02e0858f9e34697cc8e4dc543079c",
          "zv2cac10ed96f346e3abc60d968a8627cb849a2634ace14ef5e0fd36d372560469",
          "zvc21bdf00b0050bc7257a98d26cf7854231d05784e84d2be5c8890be0049990b2",
          "zva510f0f1e8300eb662871c21a3fb24720a8b80a04061f4444cb0cdde2e09692a",
          "zv35c1113386fa176a5587c8c4dfc1ae8cb9059e14863cbb761639029facdd442d",
          "zvbf5ef77bd24ebb06b1db5877dc52ecba07c4a6f9419300d10314a07d4204aea6",
          "zv3eaf4ea82f5e9a38e949d2a0ac68e721945cbba8b5d46e65766ab87741a8f1e3",
          "zv2befc19acfc95efd7da281403d2e343b1e8a238adbf05bb7343079b4e3c9f030",
          "zv4c22a68ebd875936d6f004ed4301487b3d097073aa9e5229c6aadf055e5c7e59",
          "zvb4a7d3b5856db9b6dbe4c6d1a24b2fa908244ad0369ae1a5fff281cfb7caffe4",
          "zv49c99fbbc3eae5512a71436923f5c8a34cc8007b9da2d2ef08de19da46be3bdc",
          "zv0b6a9841788f7784657c712adea834a88d8d431ccd0b01b84715f9854c5d914c",
          "zvefb27f4c9a8f25ab55eb7f600f521e9af41cf35f76c19192d830213e777e62cd",
          "zv567d994168c04fda7abc769ea7129f3fc69c707035557170369e2a1415e6bdee",
          "zvc1be331ed77b36a4deebfabece9aef10f129889a957095780560372ca245920b",
          "zva5e2e84eea0dac3fc76604a4afca3926ec0adf55b5bc8ad83bb47022ed7c4872",
          "zv517e0326f577fc14bfb999ed89c713c9ef59eb4c9ca76af2d23fd6605f7860a9",
          "zv7e2b18c0cac3074d048ec8a7e667030dd99cea3815a7022e09512588b8263fec",
          "zv5d4968eb3448c2de713f32a64ac8f6c6ee5a611d092abfc9fcdba8a5e35d6a8d",
          "zv6dcf63d369c93371c5f89e961dff654bf8dbc0e703c87a2a771793526622dadb",
          "zv3fd3613e3bae86cfa9ff2959433e1c540c0e5c8d4ad2d45e18b209002cc2a2f6",
          "zva1a8f239e62e815d9ad0830dfeddc161c224fd77644991c679859b80a0433003",
          "zv1d5b41b874fc2ddaf793eee7646866367ca0e35174cc0aae1313271eedb9a531",
          "zvc099c504cf82d732b55c71d3af9512f1e2436571a4f42427a02022c415b050ce",
          "zv42afdc8652ea4c9991eb4e2ef3d4166e661cfdbcb017c9b180e0ea36d22d5bdc",
          "zv7bdfaddae018b7d317b0427685c8268891e91a90ee52d46bddf851f9e33ae1e3",
          "zv220a2ce0c412829cda4f09d11cf6bf79c41700ff668e7165b2421e61a11c508c",
          "zv0cb7ef4d5a838e5230565013ef0c238e01ea850eeccada42b6144cd1e6d0b0f7",
          "zv4dcd723dfc52ae5f139a38f1f15d4a8de827806257563eb9d02d9536bf735bdb",
          "zv4d2f6a7cd5d933073c712c57bb8ca0ef365d4c5d87c2bf3fb13be533899c0cef",
          "zv185221bd2fd55f9a42a26307582e7775fd1ed6be086a246babdaaa4ac107c629",
          "zv05fc3eb017b6e60393beee79ab8a4611e02121c6c73d8fec8b30a99c9ec1bbfe",
          "zvb83c7b49f237ec9f99b4eccfc102ffd93df057b53f133b9dfbd1fa041ece146a",
          "zv5bf5edcbae3c8ccb822a78be6951f7b8463892361d2f2dd3e772672f7d75bb58",
          "zvbdf333e7737ac00fea81d553c7f082c7fa8ba09065d427ef8dc4bf71915ce988",
          "zv1c5551ad216ace771a55f9e97618f9c0e478093cf612b377777ca73d4102d01d",
          "zvfbb00c45f202c0a8c05b85c50da28f988c9135d2901beeb41914f9d9135c980f",
          "zv37ab95eb90305173471c155976b600a6225c910b0c39a97ae23835a14650c065",
          "zv143640e89da6764ae5168d9c57769b9f960b9f3f6db207f25bd43c2f1deed2ee"]
        for g in gl :
            self.guard_lists[g] = 1

    @register.public(str)
    def transfer_guard(self, guard):
        if msg.sender not in self.guard_lists:
            raise Exception('msg sender is not guard node!')
        del self.guard_lists[msg.sender]
        self.guard_lists[guard] = 1

    @register.public(str)
    def set_admin_addr(self, _addr):
        if msg.sender != self.adminAddr :
            raise Exception('msg sender is not admin!')
        self.adminAddr = _addr

    @register.public(str)
    def set_stake_platform_addr(self, _addr):
        if msg.sender != self.stakePlatformAddr :
            raise Exception('msg sender is not stake platform addr!')
        self.stakePlatformAddr = _addr

    @register.public(str)
    def set_circulates_addr(self, _addr):
        if msg.sender != self.circulatesAddr :
            raise Exception('msg sender is not circulates addr!')
        self.circulatesAddr = _addr

    @register.public(str)
    def set_user_node_addr(self, _addr):
        if msg.sender != self.userNodeAddr :
            raise Exception('msg sender is not node addr!')
        self.userNodeAddr = _addr

    @register.public(str)
    def set_daemon_node_addr(self, _addr):
        if msg.sender != self.daemonNodeAddr :
            raise Exception('msg sender is not daemon node addr!')
        self.daemonNodeAddr = _addr

`
