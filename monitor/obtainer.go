package monitor

import (
	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"context"
	"errors"
	"golang.local/gc-c-db/tables"
	"golang.local/master-srv/conf"
	"golang.org/x/exp/slices"
	"strconv"
	"strings"
)

func NewObtainer(cnf conf.GCPYaml) *Obtainer {
	return &Obtainer{
		active:               true,
		clientInstanceGroup:  stripError(compute.NewInstanceGroupsRESTClient(context.Background())),
		clientBackendService: stripError(compute.NewBackendServicesRESTClient(context.Background())),
		clientURLMaps:        stripError(compute.NewUrlMapsRESTClient(context.Background())),
		cnf:                  cnf,
	}
}

type Obtainer struct {
	active               bool
	clientInstanceGroup  *compute.InstanceGroupsClient
	clientBackendService *compute.BackendServicesClient
	clientURLMaps        *compute.UrlMapsClient
	cnf                  conf.GCPYaml
}

func (o *Obtainer) Obtain() (map[uint32]*MonitoredClient, error) {
	if o.clientInstanceGroup == nil || o.clientBackendService == nil || o.clientURLMaps == nil {
		return nil, errors.New("a compute api client is nil")
	}
	if !o.active {
		return nil, errors.New("obtainer not active")
	}
	toRet := make(map[uint32]*MonitoredClient)
	ctx, cancel := context.WithTimeout(context.Background(), o.cnf.GetAPITimeout())
	urlMapReq := &computepb.GetUrlMapRequest{
		Project: o.cnf.ProjectID,
		UrlMap:  o.cnf.URLMap,
	}
	urlMapRsp, err := o.clientURLMaps.Get(ctx, urlMapReq)
	cancel()
	if err != nil {
		return nil, err
	}
	for _, pm := range urlMapRsp.GetPathMatchers() {
		for _, pr := range pm.GetPathRules() {
			cPaths := pr.GetPaths()
			if len(cPaths) < 1 {
				continue
			}
			cPath := cPaths[0]
			DebugPrintln("DEBUG: PATH :" + cPath)
			if !strings.HasSuffix(cPath, "/*") {
				continue
			}
			cPathSplt := strings.Split(cPath[:len(cPath)-2], "/")
			if len(cPathSplt) < 1 {
				continue
			}
			cPathID, err := strconv.Atoi(cPathSplt[len(cPathSplt)-1])
			if err != nil {
				continue
			}
			cBackendSplt := strings.Split(pr.GetService(), "/")
			if len(cBackendSplt) < 1 {
				continue
			}
			cBackendName := cBackendSplt[len(cBackendSplt)-1]
			DebugPrintln("DEBUG: BackEndNAME :" + cBackendName)
			backendSrvReq := &computepb.GetBackendServiceRequest{
				BackendService: cBackendName,
				Project:        o.cnf.ProjectID,
			}
			ctx, cancel := context.WithTimeout(context.Background(), o.cnf.GetAPITimeout())
			backendSrvRsp, err := o.clientBackendService.Get(ctx, backendSrvReq)
			cancel()
			if err != nil {
				DebugErrIsNil(err)
				continue
			}
			if len(backendSrvRsp.GetBackends()) < 1 {
				continue
			}
			cIGroupSplt := strings.Split(backendSrvRsp.GetBackends()[0].GetGroup(), "/")
			if len(cIGroupSplt) < 1 {
				continue
			}
			cIGroupName := cIGroupSplt[len(cIGroupSplt)-1]
			DebugPrintln("DEBUG: InstanceGroupNAME :" + cIGroupName)
			if slices.Contains(o.cnf.NonAppInstanceGroups, cIGroupName) {
				continue
			}
			iGroupReq := &computepb.ListInstancesInstanceGroupsRequest{
				InstanceGroup: cIGroupName,
				Project:       o.cnf.ProjectID,
				Zone:          o.cnf.Zone,
			}
			ctx, cancel = context.WithTimeout(context.Background(), o.cnf.GetAPITimeout())
			iGroupRsp := o.clientInstanceGroup.ListInstances(ctx, iGroupReq)
			cInstanceRsp, err := iGroupRsp.Next()
			cancel()
			if err != nil {
				DebugErrIsNil(err)
				continue
			}
			cInstanceSplt := strings.Split(cInstanceRsp.GetInstance(), "/")
			if len(cInstanceSplt) < 1 {
				continue
			}
			cInstanceName := cInstanceSplt[len(cInstanceSplt)-1]
			DebugPrintln("DEBUG: InstanceNAME :" + cInstanceName)
			toRet[uint32(cPathID)] = &MonitoredClient{
				InstanceName:       cInstanceName,
				InstanceGroupName:  cIGroupName,
				BackendServiceName: cBackendName,
				Metadata: tables.Server{
					ID: uint32(cPathID),
				},
			}
		}
	}
	return toRet, nil
}

func (o *Obtainer) Close() error {
	if o.clientInstanceGroup == nil || o.clientBackendService == nil || o.clientURLMaps == nil {
		return errors.New("a compute api client is nil")
	}
	if !o.active {
		return errors.New("obtainer not active")
	}
	o.active = false
	_ = o.clientInstanceGroup.Close()
	_ = o.clientBackendService.Close()
	_ = o.clientURLMaps.Close()
	return nil
}
