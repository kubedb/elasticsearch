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

import "testing"

func TestInUserConfigCompareEqual(t *testing.T) {
	type args struct {
		x string
		y string
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
				x: `admin:
  hash: $2a$12$Ll.V9nq.tudAV8F1c3r5xeAdR2x7iyBNym2gCp/tqggxBUCdFqchK
  reserved: true
  backend_roles:
  - admin
kibanaro:
  hash: $2a$12$u/Tz5OP2wqJ6cA5WRLiwy.FuUsytPSS0TCbD/G7iEedSLSOflYhea
kibanaserver:
  hash: $2a$12$EneEDJ8Q5rt5SWrOxUcb.uDR3CmLRzgfpbsaGDvUfL6OooN8y8qxC
  reserved: true
logstash:
  hash: $2a$12$OFtdp97X.FbsYFgUSq3GT.XPBH3Y3Gtzo4iP2KzAocaidCoVdoD52
readall:
  hash: $2a$12$NXBWGLcSwpvCCcW9c2cPIuFYeelJKSieY7QuiRkQ78GeDQ03qGATK
snapshotrestore:
  hash: $2a$12$URDJvuKvRQfNA1fEgrUKSeeL9XeCFKegO7AjX7yrOW8KuUR9zoI/G
`,
				y: `admin:
  hash: $2a$12$Ll.V9nq.tudAV8F1c3r5xeAdR2x7iyBNym2gCp/tqggxBUCdFqchK
  reserved: true
  backend_roles:
  - admin
kibanaro:
  hash: $2a$12$u/Tz5OP2wqJ6cA5WRLiwy.FuUsytPSS0TCbD/G7iEedSLSOflYhea
kibanaserver:
  hash: $2a$12$EneEDJ8Q5rt5SWrOxUcb.uDR3CmLRzgfpbsaGDvUfL6OooN8y8qxC
  reserved: true
logstash:
  hash: $2a$12$OFtdp97X.FbsYFgUSq3GT.XPBH3Y3Gtzo4iP2KzAocaidCoVdoD52
readall:
  hash: $2a$12$NXBWGLcSwpvCCcW9c2cPIuFYeelJKSieY7QuiRkQ78GeDQ03qGATK
snapshotrestore:
  hash: $2a$12$URDJvuKvRQfNA1fEgrUKSeeL9XeCFKegO7AjX7yrOW8KuUR9zoI/G
`,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "first one",
			args: args{
				x: `admin:
  hash: $2a$12$Ll.V9nq.tudAV8F1c3r5xeAdR2x7iyBNym2gCp/tqggxBUCdFqchK
  reserved: true
  backend_roles:
  - admin
kibanaro:
  hash: $2a$12$u/Tz5OP2wqJ6cA5WRLiwy.FuUsytPSS0TCbD/G7iEedSLSOflYhea
kibanaserver:
  hash: $2a$12$EneEDJ8Q5rt5SWrOxUcb.uDR3CmLRzgfpbsaGDvUfL6OooN8y8qxC
  reserved: true
logstash:
  hash: $2a$12$OFtdp97X.FbsYFgUSq3GT.XPBH3Y3Gtzo4iP2KzAocaidCoVdoD52
readall:
  hash: $2a$12$NXBWGLcSwpvCCcW9c2cPIuFYeelJKSieY7QuiRkQ78GeDQ03qGATK
snapshotrestore:
  hash: $2a$12$URDJvuKvRQfNA1fEgrUKSeeL9XeCFKegO7AjX7yrOW8KuUR9zoI/G
`,
				y: `admin:
  hash: $2a$12$Ll.V9nq.tudAV8F1c3r5xeAdR2x7iyBNym2gCp/tqggxBUCdFqchK
  reserved: true
  backend_roles:
  - admin
  - demo
kibanaro:
  hash: $2a$12$u/Tz5OP2wqJ6cA5WRLiwy.FuUsytPSS0TCbD/G7iEedSLSOflYhea
kibanaserver:
  hash: $2a$12$EneEDJ8Q5rt5SWrOxUcb.uDR3CmLRzgfpbsaGDvUfL6OooN8y8qxC
  reserved: true
logstash:
  hash: $2a$12$OFtdp97X.FbsYFgUSq3GT.XPBH3Y3Gtzo4iP2KzAocaidCoVdoD52
readall:
  hash: $2a$12$NXBWGLcSwpvCCcW9c2cPIuFYeelJKSieY7QuiRkQ78GeDQ03qGATK
snapshotrestore:
  hash: $2a$12$URDJvuKvRQfNA1fEgrUKSeeL9XeCFKegO7AjX7yrOW8KuUR9zoI/G
`,
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
