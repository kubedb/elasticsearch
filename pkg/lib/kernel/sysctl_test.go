/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kernel

import (
	"testing"

	core "k8s.io/api/core/v1"
)

func TestGetSysctlCommandString(t *testing.T) {
	type args struct {
		commands  []core.Sysctl
		separator rune
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "sh command",
			args: args{
				commands: []core.Sysctl{
					{
						Name:  "vm.max_map_count",
						Value: "23432432",
					},
					{
						Name:  "vm.x_y_z",
						Value: "35",
					},
				},
				separator: ';',
			},
			want: "sysctl -w vm.max_map_count=23432432;sysctl -w vm.x_y_z=35",
		},
		{
			name: "bash command",
			args: args{
				commands: []core.Sysctl{
					{
						Name:  "vm.max_map_count",
						Value: "23432432",
					},
					{
						Name:  "vm.x_y_z",
						Value: "35",
					},
				},
				separator: '\n',
			},
			want: `sysctl -w vm.max_map_count=23432432
sysctl -w vm.x_y_z=35`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSysctlCommandString(tt.args.commands, tt.args.separator); got != tt.want {
				t.Errorf("GetSysctlCommandString() = %v, want %v", got, tt.want)
			}
		})
	}
}
