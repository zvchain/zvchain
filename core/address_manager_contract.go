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
