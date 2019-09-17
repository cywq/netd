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

package ssg

import (
	"fmt"
	"io"
	"regexp"

	"github.com/sky-cloud-tec/netd/cli"
	"golang.org/x/crypto/ssh"
)

func init() {
	// register ssg
	cli.OperatorManagerInstance.Register(`(?i)juniper\.ssg\..*`, createOpScreenOS())
}

type opScreenOS struct {
	lineBeak    string // \r\n \n
	transitions map[string][]string
	prompts     map[string][]*regexp.Regexp
	errs        []*regexp.Regexp
}

func createOpScreenOS() cli.Operator {
	sixZeroLoginPrompt := regexp.MustCompile("[[:alnum:]]{1,}(-[[:alnum:]]+){0,}-> $")
	return &opScreenOS{
		transitions: map[string][]string{},
		prompts: map[string][]*regexp.Regexp{
			"login": {sixZeroLoginPrompt},
		},

		errs: []*regexp.Regexp{
			regexp.MustCompile("\\^-+unknown keyword \\s+"),
			regexp.MustCompile("\\^-+command not completed"),
			regexp.MustCompile("\\^$"),
		},
		lineBeak: "\n",
	}
}

func (s *opScreenOS) GetPrompts(k string) []*regexp.Regexp {
	if v, ok := s.prompts[k]; ok {
		return v
	}
	return nil
}
func (s *opScreenOS) GetTransitions(c, t string) []string {
	k := c + "->" + t
	if v, ok := s.transitions[k]; ok {
		return v
	}
	return nil
}

func (s *opScreenOS) GetErrPatterns() []*regexp.Regexp {
	return s.errs
}

func (s *opScreenOS) GetLinebreak() string {
	return s.lineBeak
}

func (s *opScreenOS) GetStartMode() string {
	return "login"
}

func (s *opScreenOS) GetSSHInitializer() cli.SSHInitializer {
	return func(c *ssh.Client) (io.Reader, io.WriteCloser, *ssh.Session, error) {
		var err error
		session, err := c.NewSession()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("new ssh session failed, %s", err)
		}
		// get stdout and stdin channel
		r, err := session.StdoutPipe()
		if err != nil {
			session.Close()
			return nil, nil, nil, fmt.Errorf("create stdout pipe failed, %s", err)
		}
		w, err := session.StdinPipe()
		if err != nil {
			session.Close()
			return nil, nil, nil, fmt.Errorf("create stdin pipe failed, %s", err)
		}
		if err := session.Shell(); err != nil {
			session.Close()
			return nil, nil, nil, fmt.Errorf("create shell failed, %s", err)
		}
		return r, w, session, nil
	}
}
