package main

import "github.com/jxo-me/ddns/x/app"

func main() {
	a := app.Runtime.DDNSRegistry().Get("ddns")
	a.RunTimer(1 * 60 * 60)
}
