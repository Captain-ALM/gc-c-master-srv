package monitor

import (
	compute "cloud.google.com/go/compute/apiv1"
	"context"
	"crypto/rsa"
	"golang.local/gc-c-db/db"
	"golang.local/gc-c-db/tables"
	"golang.local/master-srv/conf"
	"log"
	"net/http"
	"strconv"
	"time"
)

func stripError[t any](v t, e error) t {
	var n t
	if e == nil {
		return v
	}
	log.Println("[Monitor] ", e.Error())
	return n
}

func NewMonitor(cnf conf.ConfigYaml, dbMan *db.Manager, prvk *rsa.PrivateKey) *Monitor {
	return &Monitor{
		client:     stripError(compute.NewInstancesRESTClient(context.Background())),
		cnf:        cnf,
		privateKey: prvk,
		dbManager:  dbMan,
	}
}

type Monitor struct {
	active     bool
	client     *compute.InstancesClient
	clients    map[uint32]*MonitoredClient
	privateKey *rsa.PrivateKey
	dbManager  *db.Manager
	cnf        conf.ConfigYaml
	byeChan    chan bool
}

func (m *Monitor) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.Header().Set("Cache-Control", "max-age=0, no-cache, no-store, must-revalidate")
	writer.Header().Set("Pragma", "no-cache")
	if request.Method == http.MethodGet {
		kaDuration := -(m.cnf.Balancer.GetCheckInterval() + m.cnf.Balancer.GetCheckTimeout())
		if request.URL.Query().Has("i") {
			gid, err := strconv.Atoi(request.URL.Query().Get("i"))
			if err == nil && gid > 0 {
				theGame := tables.Game{ID: uint32(gid)}
				err := m.dbManager.Load(&theGame)
				if err == nil && theGame.ServerID > 0 {
					theCl, has := m.clients[theGame.ServerID]
					if has && theCl.HasIDShaked() &&
						theCl.LastLoad.Current < theCl.LastLoad.Max &&
						!theCl.Metadata.LastCheckTime.Before(time.Now().Add(kaDuration)) {
						writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
						writer.Header().Set("Content-Length", strconv.Itoa(len(theCl.Metadata.Address)))
						writer.WriteHeader(http.StatusOK)
						_, _ = writer.Write([]byte(theCl.Metadata.Address))
					} else {
						if has && !theCl.IsActive() {
							go func() { _ = theCl.Activate(m.cnf, m.dbManager, m.privateKey) }()
						}
						writer.WriteHeader(http.StatusNotFound)
					}
				} else {
					writer.WriteHeader(http.StatusNotFound)
				}
				return
			}
		}
		var lowestLoadCl *MonitoredClient
		for _, mcl := range m.clients {
			if !mcl.HasIDShaked() ||
				mcl.LastLoad.Current >= mcl.LastLoad.Max ||
				mcl.Metadata.LastCheckTime.Before(time.Now().Add(kaDuration)) {
				if !mcl.IsActive() {
					go func() { _ = mcl.Activate(m.cnf, m.dbManager, m.privateKey) }()
				}
				continue
			}
			if lowestLoadCl == nil || mcl.LastLoad.Max-mcl.LastLoad.Current > lowestLoadCl.LastLoad.Max-lowestLoadCl.LastLoad.Current {
				lowestLoadCl = mcl
			}
		}
		if lowestLoadCl == nil {
			writer.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		writer.Header().Set("Content-Length", strconv.Itoa(len(lowestLoadCl.Metadata.Address)))
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(lowestLoadCl.Metadata.Address))
	} else if request.Method == http.MethodOptions {
		writer.Header().Set("Allow", http.MethodOptions+", "+http.MethodGet)
		writer.WriteHeader(http.StatusOK)
	} else {
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (m *Monitor) Start() {
	if m.active {
		return
	}
	m.active = true
	m.byeChan = make(chan bool)
	obt := NewObtainer(m.cnf.GCP)
	var err error
	m.clients, err = obt.Obtain()
	if err != nil {
		log.Println("[Monitor]", err.Error())
		_ = obt.Close()
		return
	}
	go func() {
		tOutChan := make(chan bool, 1)
		defer func() {
			close(tOutChan)
			close(m.byeChan)
			for _, cc := range m.clients {
				_ = cc.Close()
			}
		}()
		for m.active {
			go func() {
				<-time.After(m.cnf.Balancer.GetCheckInterval())
				tOutChan <- true
			}()
			for _, cc := range m.clients {
				cc.Monitor(m.cnf.Balancer.GetCheckInterval())
			}
			select {
			case <-m.byeChan:
				return
			case <-tOutChan:
			}
		}
	}()
	for _, cc := range m.clients {
		err = cc.Activate(m.cnf, m.dbManager, m.privateKey)
		/*if err != nil {
			log.Println("[Monitor]", err.Error())
			_ = obt.Close()
			return
		}*/
	}
}

func (m *Monitor) IsActive() bool {
	return m.active
}

func (m *Monitor) Stop() {
	if m.active {
		m.active = false
		m.byeChan <- true
		if m.client != nil {
			_ = m.client.Close()
		}
	}
}
