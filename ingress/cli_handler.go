// NetD makes network device operations easy.
// Copyright (C) 2019  sky-cloud.net
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package ingress

import (
	"strings"
	"time"

	"github.com/songtianyi/rrframework/utils"

	"github.com/sky-cloud-tec/netd/cli"
	_ "github.com/sky-cloud-tec/netd/cli/brocade/g600" //  load linux centos
	_ "github.com/sky-cloud-tec/netd/cli/cisco/asa"    // load juniper srx
	_ "github.com/sky-cloud-tec/netd/cli/cisco/ios"    // load cisco switch ios
	_ "github.com/sky-cloud-tec/netd/cli/cisco/nxos"   // load cisco switch nxos
	"github.com/sky-cloud-tec/netd/cli/conn"
	_ "github.com/sky-cloud-tec/netd/cli/dptech/fw1000"      // load dptech fw1000
	_ "github.com/sky-cloud-tec/netd/cli/fortinet/fortigate" // load fortinet fortigate
	_ "github.com/sky-cloud-tec/netd/cli/h3c/secpath"        // load h3c secpath
	_ "github.com/sky-cloud-tec/netd/cli/hillstone/sg6000"   // load hillstone SG6000
	_ "github.com/sky-cloud-tec/netd/cli/huawei/usg"         // load huawei USG
	_ "github.com/sky-cloud-tec/netd/cli/juniper/srx"        // load cisco asa
	_ "github.com/sky-cloud-tec/netd/cli/juniper/ssg"        // load juniper ssg
	_ "github.com/sky-cloud-tec/netd/cli/linux/centos"       // load linux centos
	_ "github.com/sky-cloud-tec/netd/cli/paloalto/panos"     // load paloalto panos
	_ "github.com/sky-cloud-tec/netd/cli/topsec/ngfw4000"    // load topsec ngfw4000

	"github.com/sky-cloud-tec/netd/common"
	"github.com/sky-cloud-tec/netd/protocol"
	"github.com/songtianyi/rrframework/logs"
)

// CliHandler run cli commands and return result to caller
type CliHandler struct {
	req *protocol.CliRequest
}

// Handle cli request
func (s *CliHandler) Handle(req *protocol.CliRequest, res *protocol.CliResponse) error {
	if req.Session == "" {
		req.Session = rrutils.NewV4().String()
	}
	logs.Info("Received req", req)
	// build timeout
	if req.Timeout == 0 {
		req.Timeout = common.DefaultTimeout
	} else {
		req.Timeout = req.Timeout * time.Second
	}

	// build log prefix
	if req.LogPrefix == "" {
		req.LogPrefix = "[ " + req.Device + " ]"
	}

	s.req = req

	ch := make(chan error, 1)

	go func() {
		req.LogPrefix = req.LogPrefix + " [ " + req.Session + " ] "
		logs.Info(req.LogPrefix, "==========START==========")
		ch <- s.doHandle(req, res)
		logs.Info(req.LogPrefix, "==========END==========")
	}()

	// timeout
	select {
	case res := <-ch:
		return res
	case <-time.After(req.Timeout):
		*res = s.makeCliErrRes(common.ErrTimeout, "handle req timeout")
	}
	return nil
}

func (s *CliHandler) doHandle(req *protocol.CliRequest, res *protocol.CliResponse) error {
	// build device operator type
	t := strings.Join([]string{req.Vendor, req.Type, req.Version}, ".")
	// get operator by type
	op := cli.OperatorManagerInstance.Get(t)
	if op == nil {
		logs.Error(req.LogPrefix, "no operator match", t)
		*res = s.makeCliErrRes(common.ErrNoOpFound, "no operator match "+t)
		return nil
	}
	// acquire cli connection, it could be blocked here for concurrency
	c, err := conn.Acquire(req, op)
	defer conn.Release(req)
	if err != nil {
		logs.Error(req.LogPrefix, "new operator fail,", err)
		*res = s.makeCliErrRes(common.ErrAcquireConn, "acquire cli conn fail, "+err.Error())
		return nil
	}
	// execute cli commands
	out, err := c.Exec()
	if err != nil {
		logs.Error(req.LogPrefix, "exec error,", err)
		*res = s.makeCliErrRes(common.ErrCliExec, "exec cli cmds fail, "+err.Error())
		return nil
	}
	// make reponse
	*res = protocol.CliResponse{
		Retcode: common.OK,
		Message: "OK",
		Device:  req.Device,
		CmdsStd: out,
	}
	return nil
}

func (s *CliHandler) makeCliErrRes(code int, msg string) protocol.CliResponse {
	return protocol.CliResponse{Retcode: code, Message: msg, Device: s.req.Device, CmdsStd: nil}
}
