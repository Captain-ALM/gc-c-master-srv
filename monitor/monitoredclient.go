package monitor

import (
	"crypto/rsa"
	"errors"
	"golang.local/gc-c-com/packet"
	"golang.local/gc-c-com/packets"
	"golang.local/gc-c-com/transport"
	"golang.local/gc-c-db/db"
	"golang.local/gc-c-db/tables"
	"golang.local/master-srv/conf"
	"net/url"
	"strconv"
	"time"
)

type MonitoredClient struct {
	InstanceName       string
	InstanceGroupName  string
	BackendServiceName string
	idRecv             bool
	client             *transport.Client
	Metadata           tables.Server
	LastLoad           packets.CurrentStatusPayload
}

func (m *MonitoredClient) Activate(cnf conf.ConfigYaml, dbMan *db.Manager, prvk *rsa.PrivateKey) error {
	err := dbMan.Load(&m.Metadata)
	if err.Error() == db.TableRecordNonExistent {
		m.Metadata.Address = cnf.GCP.GetAppBasePrefix() + "/" + strconv.Itoa(int(m.Metadata.ID))
		m.Metadata.LastCheckTime = time.Now()
		err = dbMan.Save(&m.Metadata)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	m.client = &transport.Client{}
	m.client.SetTimeout(cnf.Balancer.GetCheckTimeout())
	m.client.SetKeepAlive(cnf.Balancer.GetCheckInterval())
	bsURL, err := url.Parse(cnf.GCP.GetAppScheme() + "://" + cnf.GCP.GetAppHost(cnf) + m.Metadata.Address)
	if err != nil {
		return err
	}
	m.client.Activate(bsURL.String()+"/ws", bsURL.String()+"/rs")
	go func() {
		defer func() { m.idRecv = false }()
		for m.client.IsActive() {
			pk, err := m.client.Receive()
			if err == nil {
				switch pk.GetCommand() {
				case packets.ID:
					var pyl packets.IDPayload
					err = pk.GetPayload(&pyl)
					if err != nil {
						_ = m.client.Close()
					} else {
						if pyl.ID == m.Metadata.ID {
							m.idRecv = true
						} else {
							_ = m.client.Close()
						}
					}
				case packets.CurrentStatus:
					if !m.idRecv {
						continue
					}
					err = pk.GetPayload(&m.LastLoad)
					if err == nil {
						m.Metadata.LastCheckTime = time.Now()
						_ = dbMan.Save(&m.Metadata)
					}
				case packets.Halt:
					_ = m.client.Close()
				}
			}
		}
	}()
	if m.client.IsActive() {
		return m.client.Send(packet.FromNew(packets.NewID(m.Metadata.ID, prvk)))
	}
	return nil
}

func (m *MonitoredClient) IsActive() bool {
	if m.client == nil {
		return false
	}
	return m.client.IsActive()
}

func (m *MonitoredClient) Monitor(checkInterval time.Duration) {
	if m.client == nil || time.Now().Add(checkInterval).Before(m.Metadata.LastCheckTime.Add(checkInterval)) {
		return
	}
	_ = m.client.Send(packet.FromNew(packets.NewQueryStatus(nil)))
}

func (m *MonitoredClient) Close() error {
	if m.client == nil {
		return errors.New("monitored client internal client is nil")
	}
	err := m.client.Close()
	m.idRecv = false
	return err
}

func (m *MonitoredClient) HasIDShaked() bool {
	if m == nil {
		return false
	}
	return m.idRecv
}