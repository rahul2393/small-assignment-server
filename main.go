package main

import (
	"fmt"
	"log"
	"net/http"

	"gopkg.in/BurntSushi/toml.v0"

	"github.com/rahul2393/small-assignment-server/router"
)

const (
	port = "8000"
)

func main() {
	handler := router.SetupRouters()
	fmt.Println("Starting https server")
	type configFile struct {
		CertPath string `toml:"cert_path"`
		KeyPath  string `toml:"key_path"`
	}
	cfg := &configFile{}
	if _, err := toml.DecodeFile("./conf.toml", &cfg); err != nil {
		fmt.Printf("errors is %v\n", err)
	}
	go http.ListenAndServeTLS(":443", cfg.CertPath, cfg.KeyPath, handler)
	err := http.ListenAndServe(":"+port, handler)
	if err != nil {
		log.Fatal(err)
	}
}
