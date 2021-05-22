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

package user

import (
	"testing"

	api "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
)

func TestInUserConfigCompareEqual(t *testing.T) {
	type args struct {
		x map[string]api.ElasticsearchUserSpec
		y map[string]api.ElasticsearchUserSpec
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "first one",
			args: args{
				x: map[string]api.ElasticsearchUserSpec{
					"admin": {
						Hash:       "$2a$12$Ll.V9nq.tudAV8F1c3r5xeAdR2x7iyBNym2gCp/tqggxBUCdFqchK",
						SecretName: "admin-cred",
						Reserved:   true,
						Hidden:     false,
					},
				},
				y: map[string]api.ElasticsearchUserSpec{
					"admin": {
						Hash:       "$2a$12$Ll",
						SecretName: "admin-cred",
						Reserved:   true,
						Hidden:     false,
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "second one",
			args: args{
				x: map[string]api.ElasticsearchUserSpec{
					"admin": {
						Hash:       "$2a$12$Ll.V9nq.tudAV8F1c3r5xeAdR2x7iyBNym2gCp/tqggxBUCdFqchK",
						SecretName: "admin-cred-2",
						Reserved:   true,
						Hidden:     false,
					},
				},
				y: map[string]api.ElasticsearchUserSpec{
					"admin": {
						Hash:       "$2a$12$Ll",
						SecretName: "admin-cred",
						Reserved:   true,
						Hidden:     false,
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "third one",
			args: args{
				x: map[string]api.ElasticsearchUserSpec{
					"admin": {
						Reserved: false,
						Hidden:   false,
					},
				},
				y: map[string]api.ElasticsearchUserSpec{
					"admin": {
						Reserved: true,
						Hidden:   false,
					},
				},
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InUserConfigCompareEqual(tt.args.x, tt.args.y)
			if (err != nil) != tt.wantErr {
				t.Errorf("InUserConfigCompareEqual() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("InUserConfigCompareEqual() got = %v, want %v", got, tt.want)
			}
		})
	}
}
