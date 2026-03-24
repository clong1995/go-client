package client

import (
	"testing"
)

func TestDo(t *testing.T) {
	type args struct {
		uid   uint64
		api   string
		param any
		type_ string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "测试请求",
			args: args{
				uid: 2502305606140997632,
				api: "http://127.0.0.1:9011/session",
				param: struct {
					Session uint64
					Path    string
				}{
					Session: 2502938842714124288,
					Path:    "/user/login/post",
				},
				type_: "GOB",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRes, err := Do(tt.args.uid, tt.args.api, tt.args.param, tt.args.type_)
			if (err != nil) != tt.wantErr {
				t.Errorf("Do() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			t.Logf("Do() gotRes = %v", gotRes)
		})
	}
}
