package monitor

import (
	compute "cloud.google.com/go/compute/apiv1"
	"context"
	"crypto/rsa"
	"golang.local/gc-c-db/db"
	"golang.local/master-srv/conf"
	"log"
	"net/http"
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
	clients    []*MonitoredClient
	privateKey *rsa.PrivateKey
	dbManager  *db.Manager
	cnf        conf.ConfigYaml
	byeChan    chan bool
}

func (m *Monitor) ServeHTTP(writer http.ResponseWriter, request *http.Request) {

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
				<-time.After(m.cnf.Balancer.CheckInterval)
				tOutChan <- true
			}()
			for _, cc := range m.clients {
				cc.Monitor(m.cnf.Balancer.CheckInterval)
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
		if err != nil {
			log.Println("[Monitor]", err.Error())
			_ = obt.Close()
			return
		}
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
