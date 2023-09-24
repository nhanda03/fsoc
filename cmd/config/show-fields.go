// Copyright 2023 Cisco Systems, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"os"
	"reflect"
	"strings"

	"github.com/apex/log"
	"github.com/mitchellh/go-wordwrap"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	cfg "github.com/cisco-open/fsoc/config"
)

func newCmdConfigShowFields() *cobra.Command {

	var cmd = &cobra.Command{
		Use:         "show-fields",
		Short:       `Show fields that can be configured with "config set"`,
		Long:        `Show the names and meaning of fields that can be configured with the "config set" command`,
		Args:        cobra.NoArgs,
		Annotations: map[string]string{cfg.AnnotationForConfigBypass: ""},
		Run:         configShowFields,
	}

	return cmd
}

var fieldHelp = map[string]string{
	"auth":        `Authentication method, required. Must be of: "` + strings.Join(GetAuthMethodsStringList(), `", "`) + `".`,
	"url":         `URL to the tenant, scheme and host/port only; required. For example, https://mytenant.observe.appdynamics.com`,
	"tenant":      `Tenant ID that is required only for auth methods that cannot automatically obtain it. Not needed for "oauth" and "service-principal" auth methods.`,
	"secret-file": `File containing login credentials for "service-principal" and "agent-principal" auth methods. The file must remain available.`,
	"envtype":     `FSO environment type, optional. Used only for special development/test FSO environments. If specified, can be "dev" or "prod".`,
	"token":       `Authentication token needed only for the "token" auth method.`,
	cfg.AppdTid:   `Value of ` + cfg.AppdPid + ` to use with the "local" auth method.`,
	cfg.AppdPty:   `Value of ` + cfg.AppdPid + ` to use with the "local" auth method.`,
	cfg.AppdPid:   `Value of ` + cfg.AppdPid + ` to use with the "local" auth method.`,
	"server":      `Synonym for the "url" setting. Deprecated.`,
}

func configShowFields(cmd *cobra.Command, args []string) {
	cmd.Printf(`The following settings can be configured with the "config set" command.` + "\n\n" +
		`Example:
  fsoc config set auth=oauth url=mytenant.observe.appdynamics.com` + "\n\n")

	// display core fields
	fields := []string{}
	helps := []string{}
	cmd.Printf("Core configuration fields\n\n")
	for _, field := range configArgs {
		help, found := fieldHelp[field]
		if !found {
			help = "(no description available)"
		}

		fields = append(fields, field)
		helps = append(helps, help)
	}
	formatAndDisplayFields(cmd, fields, helps)

	// display subsystem-specific configuration fields
	fields = []string{}
	helps = []string{}
	for _, subsystemName := range cfg.GetRegisteredSubsystems() {
		cmd.Printf("Subsystem %q configuration fields\n", subsystemName)

		// get config template structure for the subsystem
		template, err := cfg.GetSubsytemConfigTemplate(subsystemName)
		if err != nil {
			log.Warnf("Could not obtain config settings for subsysem %q: %v; skipping subsystem", subsystemName, err)
			continue
		}

		// collect field names and helps
		typ := reflect.TypeOf(template).Elem()
		for i := 0; i < typ.NumField(); i++ {
			// get info for the field
			structField := typ.Field(i)
			mapstructTag := structField.Tag.Get("mapstructure")
			fsochelpTag := structField.Tag.Get("fsoc-help")
			mapstructElems := strings.SplitN(mapstructTag, ",", 2)
			fieldName := mapstructElems[0]
			if fieldName == "" {
				log.Warnf("(bug) cannot find name config field %v; skipping", structField.Name)
				continue
			}
			if fsochelpTag == "" {
				fsochelpTag = "(no description available)"
			}

			// append info
			fields = append(fields, subsystemName+"."+fieldName) // subsystem.setting
			helps = append(helps, fsochelpTag)
		}
		formatAndDisplayFields(cmd, fields, helps)
	}
}

func formatAndDisplayFields(cmd *cobra.Command, fields []string, helps []string) {
	// TODO: consider printing using output.Table (when detail tables support multi-line values)

	// determina terminal's size if outputting to a terminal
	terminalWidth := 0 // assume not outputting to a terminal
	if term.IsTerminal(int(os.Stdout.Fd())) {
		width, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err == nil {
			terminalWidth = width
		}
	}

	// determine max field width (field names are expected to be ASCII)
	fieldWidth := 0
	for _, field := range fields {
		if l := len(field); l > fieldWidth {
			fieldWidth = l
		}
	}

	// finalize width and format parameters
	prefixWidth := 2 // how many spaces before field name
	suffixWidth := 3 // how many spaces after the field name's max width
	helpWidth := 20  // minimal reasonable width
	if terminalWidth < 60 || terminalWidth < fieldWidth+prefixWidth+suffixWidth+helpWidth {
		terminalWidth = 0 // disable wrapping if helps will be very narrow
	} else {
		helpWidth = terminalWidth - prefixWidth - fieldWidth - suffixWidth
	}

	// display table
	if len(fields) != len(helps) {
		log.Fatal("(bug) fields and help texts must have the same dimensions")
	}
	for i := range fields {
		help := helps[i]

		// word-wrap long strings and indent non-first lines
		if terminalWidth > 0 {
			help = wordwrap.WrapString(help, uint(helpWidth))
			help = strings.ReplaceAll(help, "\n", "\n"+strings.Repeat(" ", prefixWidth+fieldWidth+suffixWidth))
		}

		// display fomatted line(s)
		cmd.Printf("%*v%*v%*v%v\n", prefixWidth, "", -fieldWidth, fields[i], suffixWidth, " : ", help)
	}
	cmd.Printf("\n")
}