# ifname-sort

対応OS:RedHat系 5,6
このプログラムは、NICの番号割り振りがうまくできなかった時に有用である。
実行すると、PCIバス番号の若番の順に、ethの番号を振り直し、70-persistent-net.rulesとifcfg-ethxを
作成してくれる。
