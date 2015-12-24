package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"text/template"
)

/*
Iface構造体を宣言
iname：インターフェイスの名前(ethx)
macaddr：macaddress
pname：pciバス番号
*/
type Iface struct {
	Iname   string
	Macaddr string
	Pname   string
}

/*
Ifaceの集まりであるIfaceSlice構造体を宣言
Ifaces：インターフェイス構造体のSlice(sliceは他の言語で言うところの可変長の配列)
*/
type IfaceSlice struct {
	Ifaces []Iface
}

/*
Iface構造体にSort関数を使用する為に実装
IfaceSlice.Iface.Pnameを呼び出し、若番にソートする
ソートはASCII文字コードの若い順にソートを実行
*/
func (p IfaceSlice) Len() int           { return len(p.Ifaces) }
func (p IfaceSlice) Less(i, j int) bool { return p.Ifaces[i].Pname < p.Ifaces[j].Pname }
func (p IfaceSlice) Swap(i, j int)      { p.Ifaces[i], p.Ifaces[j] = p.Ifaces[j], p.Ifaces[i] }

/*
インターフェイス名のリストを作成する関数
ifconfig -aの標準出力を受け取り、実行時に渡されたs(正規表現の文字列)で標準出力の中からマッチしたものを
isのSliceに入れる処理をしている
※r.FindAllStringはマッチしたものを順にsliceに入れて返す実装がなされている
　また、今回は-1が指定してあるためマッチした数だけSliceに入れる
*/
func cInameSlice(s string) []string {
	ifconfig, err := exec.Command("ifconfig", "-a").Output()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	r := regexp.MustCompile(s)
	is := r.FindAllString(string(ifconfig), -1)

	return is
}

/*
macaddressのリストを作成する関数
macaddressのリストを作るにあたって、ethxのmacaddressに紐付いて、リストを作るためにethtool -Pコマンドを利用
for文でinameSliceの数だけ、繰り返し、ethtool -P [インターフェイス名]を実行し、
実行時に渡された標準出力からmacaddressを正規表現で抜き出し、mstのSliceに入れる処理をしている
その後、mstの0番目に入っているものを、msにappend(配列の後ろに足していく機能)してリストを作成している
*/
func cMacSlice(is []string) []string {
	var ms []string

	for i := 0; i < len(is); i++ {
		r := regexp.MustCompile(`[0-9A-Fa-f][0-9A-Fa-f]:[0-9A-Fa-f][0-9A-Fa-f]:[0-9A-Fa-f][0-9A-Fa-f]:[0-9A-Fa-f][0-9A-Fa-f]:[0-9A-Fa-f][0-9A-Fa-f]:[0-9A-Fa-f][0-9A-Fa-f]`)
		ethtool, err := exec.Command("ethtool", "-P", is[i]).Output()
		if err != nil {
			fmt.Println(err)
		}

		mst := r.FindAllString(string(ethtool), -1)
		ms = append(ms, mst[0])
	}

	return ms
}

/* PCIバス番号のリストを作成する関数 PCIバス番号のリストを作るにあたって、ethxのPCIバス番号に紐付いて、リストを作るためにethtool -iコマンドを利用
for文でinameSliceの数だけ、繰り返し、ethtool -i [インターフェイス名]を実行し、
実行時に渡された標準出力からPCIバス番号を正規表現で抜き出し、pstのSliceに入れる処理をしている
その後、pstの0番目に入っているものを、psにappend(配列の後ろに足していく機能)してリストを作成している
*/

func cPnameSlice(is []string) []string {
	var ps []string

	for i := 0; i < len(is); i++ {
		r := regexp.MustCompile(`0000:.*`)
		ethtool, err := exec.Command("ethtool", "-i", is[i]).Output()
		if err != nil {
			fmt.Println(err)
		}

		pst := r.FindAllString(string(ethtool), -1)
		ps = append(ps, pst[0])
	}

	return ps
}

/* 70-persisten-net.rulesを作成する関数
   使用するために必要なパラメータはpath(/path/to/dir/の形式)
   golangの標準ライブラリであるtemplateを使い、変数を代入しテキストを出力する
   ここれは与えられたPath配下に70-persistent-net.rulesというファイルを作成する処理になっている
*/

func ePersistentNetTpl(path string, ia IfaceSlice) {

	var doc bytes.Buffer
	const persisten_net_template = `{{range $index, $ia := .Ifaces}}
SUBSYSTEM=="net", ACTION=="add", DRIVERS=="?*", ATTR{address}=="{{$ia.Macaddr}}", ATTR{type}=="1", KERNEL=="eth*", NAME="{{$ia.Iname}}"
{{end}}`

	tpl := template.Must(template.New("70-persisten-net.rules_template").Parse(persisten_net_template))
	tpl.Execute(&doc, ia)
	s := doc.String()

	//persistent_net := "/etc/udev/rules.d/70-persistent-net.rules"
	persistent_net := path + "70-persistent-net.rules"
	fo, err := os.Create(persistent_net)
	if err != nil {
		fmt.Println(persistent_net, err)
		return
	}
	defer fo.Close()
	fo.WriteString(s)

}

/* ifcfg-ethxを作成する関数
   使用するために必要なパラメータはpath(/path/to/dir/の形式)
   golangの標準ライブラリであるtemplateを使い、変数を代入しテキストを出力する
   ここれは与えられたPath配下にinterface ethxの数だけ、ifcfg-ethxというファイルを作成する処理になっている
*/

func eIfcfgTpl(path string, ia IfaceSlice) {

	const ifcfg_template = `DEVICE={{.Iname}}
HWADDR={{.Macaddr}}
TYPE=Ethernet
ONBOOT=yes
BOOTPROTO=none
`
	tpl := template.Must(template.New("ifcfg_template").Parse(ifcfg_template))

	for i := 0; i < len(ia.Ifaces); i++ {
		var doc bytes.Buffer
		tpl.Execute(&doc, ia.Ifaces[i])
		s := doc.String()

		//ifcfg := "/etc/sysconfig/network-scripts/ifcfg-" + ia.Ifaces[i].Iname
		ifcfg := path + "ifcfg-" + ia.Ifaces[i].Iname
		fo, err := os.Create(ifcfg)
		if err != nil {
			fmt.Println(ifcfg, err)
			return
		}
		defer fo.Close()

		fo.WriteString(s)

	}
}

/*
main関数
ここでの処理の流れは以下
1.cInameSlice関数に正規表現を渡し、実行結果をinameSliceに代入
2.cMacSlice関数の実行結果をmacSliceに代入
3.cPnameSlice関数の実行結果をpnamecSliceに代入
4.inameSliceの数だけ、処理を繰り返し、Iface構造体に各iname,macaddr,pnmaeの情報を紐付けて代入し、
  構造体をIfacesスライスに追加していく
5.Pnameの若番順にIfacesスライスのソートを実行し、pnameの若番順にethxの名前を振り直しをする
6.ePersistentNetTpl関数にpathを与えて、70-persistent-net.rulesを作成する
7.eIfcfgTpl関数にpathを与えて、ifcfg-ethxを作成する
*/
func main() {

	var iface Iface
	var ia IfaceSlice

	/*test data
	inameSlice := []string{"eth0", "eth1", "eth2", "eth3", "eth4", "eth5"}
	macSlice := []string{"cc:46:d6:4e:d6:68", "cc:46:d6:4e:d6:69", "cc:46:d6:4e:d6:6a", "cc:46:d6:4e:d6:6b", "cc:46:d6:58:c7:1a", "cc:46:d6:58:c7:1b"}
	pnameSlice := []string{"0000:04:00.0", "0000:04:00.1", "0000:04:00.2", "0000:04:00.3", "0000:01:00.0", "0000:01:00.1"}

	*/

	//1
	inameSlice := cInameSlice("eth[0-9]")
	//inameSlice := cInameSlice("")

	//2
	macSlice := cMacSlice(inameSlice)

	//3
	pnameSlice := cPnameSlice(inameSlice)

	//4
	for i := 0; i < len(inameSlice); i++ {
		iface.Iname = inameSlice[i]
		iface.Macaddr = macSlice[i]
		iface.Pname = pnameSlice[i]
		ia.Ifaces = append(ia.Ifaces, iface)
	}

	//debug
	for i := 0; i < len(ia.Ifaces); i++ {
		fmt.Println(ia.Ifaces[i])
	}

	//5
	sort.Sort(ia)
	for i := 0; i < len(ia.Ifaces); i++ {
		ia.Ifaces[i].Iname = "eth" + strconv.Itoa(i)
	}

	//debug
	for i := 0; i < len(ia.Ifaces); i++ {
		fmt.Println(ia.Ifaces[i])
	}

	//6
	//ePersistentNetTpl("/etc/udev/rules.d/", ia)
	ePersistentNetTpl("/", ia)

	//7
	//eIfcfgTpl("/etc/sysconfig/network-scripts/", ia)
	eIfcfgTpl("/", ia)

}
