package main

import (
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"golang.local/gc-c-db/db"
	"golang.local/master-srv/conf"
	"golang.local/master-srv/monitor"
	"golang.local/master-srv/web"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"
)

var (
	buildVersion = "develop"
	buildDate    = ""
)

func main() {
	log.Printf("[Main] Starting up Decide Quiz Master Server #%s (%s)\n", buildVersion, buildDate)
	y := time.Now()

	//Hold main thread till safe shutdown exit:
	wg := &sync.WaitGroup{}
	wg.Add(1)

	//Get working directory:
	cwdDir, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}

	//Load environment file:
	err = godotenv.Load()
	if err != nil {
		log.Fatalln("Error loading .env file")
	}

	//Data directory processing:
	dataDir := os.Getenv("DIR_DATA")
	if dataDir == "" {
		dataDir = path.Join(cwdDir, ".data")
	}

	check(os.MkdirAll(dataDir, 0777))

	//Config file processing:
	configLocation := os.Getenv("CONFIG_FILE")
	if configLocation == "" {
		configLocation = path.Join(dataDir, "config.yml")
	} else {
		if !filepath.IsAbs(configLocation) {
			configLocation = path.Join(dataDir, configLocation)
		}
	}

	//Config loading:
	configFile, err := os.Open(configLocation)
	if err != nil {
		log.Fatalln("Failed to open config.yml")
	}

	var configYml conf.ConfigYaml
	groupsDecoder := yaml.NewDecoder(configFile)
	err = groupsDecoder.Decode(&configYml)
	if err != nil {
		log.Fatalln("Failed to parse config.yml:", err)
	}

	//DB Connection:
	manager := &db.Manager{Path: configYml.DBPath}
	err = manager.Connect()
	if err != nil {
		log.Fatalln("Failed to open DB connection:", err)
	}

	if os.Getenv("DB_RESET") == "1" {
		err = manager.DropAllTables()
		if err != nil {
			log.Fatalln("Failed to reset DB:", err)
		}
	}

	err = manager.AssureAllTables()
	if err != nil {
		log.Fatalln("Failed to assure DB schema:", err)
	}

	//Load keys:
	prvkb, err := base64.StdEncoding.DecodeString(configYml.Identity.PrivateKey)
	if err != nil {
		log.Fatal("Failed to decode private key:", err)
	}
	prvk, err := jwt.ParseRSAPrivateKeyFromPEM(prvkb)
	if err != nil {
		log.Fatal("Failed to decode private key:", err)
	}
	var pubk *rsa.PublicKey
	if configYml.Identity.PublicKey == "" {
		pubk = &prvk.PublicKey
	} else {
		var pubkb []byte
		pubkb, err = base64.StdEncoding.DecodeString(configYml.Identity.PublicKey)
		if err != nil {
			log.Fatal("Failed to decode public key:", err)
		}
		pubk, err = jwt.ParseRSAPublicKeyFromPEM(pubkb)
		if err != nil {
			log.Fatal("Failed to decode public key:", err)
		}
	}
	runtime.KeepAlive(pubk) //TODO: Rem this

	//Server definitions:
	log.Println("[Main] Starting up Monitor service...")
	monService := monitor.NewMonitor(configYml, manager, prvk)
	monService.Start()
	log.Printf("[Main] Starting up HTTP server on %s...\n", configYml.Listen.Web)
	webServer := web.New(configYml, monService)

	//Safe Shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	//Startup complete:
	z := time.Now().Sub(y)
	log.Printf("[Main] Took '%s' to fully initialize modules\n", z.String())

	go func() {
		<-sigs
		fmt.Printf("\n")

		log.Printf("[Main] Attempting safe shutdown\n")
		a := time.Now()

		log.Printf("[Main] Shutting down HTTP server...\n")
		err := webServer.Close()
		if err != nil {
			log.Println(err)
		}

		log.Printf("[Main] Stopping Monitor Service...\n")
		monService.Stop()

		log.Printf("[Main] Signalling program exit...\n")
		b := time.Now().Sub(a)
		log.Printf("[Main] Took '%s' to fully shutdown modules\n", b.String())
		wg.Done()
	}()

	wg.Wait()
	log.Println("[Main] Goodbye")
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
