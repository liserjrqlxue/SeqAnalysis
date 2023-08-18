package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/liserjrqlxue/goUtil/simpleUtil"
)

// flag
var (
	port = flag.String("port", ":9091", "port")
)

func main() {
	flag.Parse()

	// mkdir work dir
	os.MkdirAll("public", 0755)

	http.HandleFunc("/", uploadHandler)

	// handle file serve
	http.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("public"))))

	log.Printf("start http://%v%v\n", GetOutboundIP(), *port)
	simpleUtil.CheckErr(http.ListenAndServe(*port, nil))
}

// Get preferred outbound ip of this machine
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}
